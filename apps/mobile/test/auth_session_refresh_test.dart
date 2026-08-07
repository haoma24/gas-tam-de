import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gas_tam_de/core/api_client.dart';
import 'package:gas_tam_de/features/auth/auth_api.dart';
import 'package:gas_tam_de/features/auth/auth_models.dart';
import 'package:gas_tam_de/features/auth/auth_session.dart';
import 'package:shared_preferences/shared_preferences.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  late SharedPreferences prefs;

  setUp(() async {
    SharedPreferences.setMockInitialValues({});
    prefs = await SharedPreferences.getInstance();
  });

  ProviderContainer containerWithRefreshApi(int Function() refreshCalls) {
    final refreshDio = Dio();
    refreshDio.interceptors.add(
      InterceptorsWrapper(
        onRequest: (options, handler) {
          refreshCalls();
          handler.resolve(
            Response<Map<String, dynamic>>(
              requestOptions: options,
              statusCode: 200,
              data: const {
                'access_token': 'new-access-token',
                'refresh_token': 'new-refresh-token',
                'token_type': 'Bearer',
                'expires_in': 900,
                'user': {
                  'id': 'customer-1',
                  'role': 'customer',
                  'phone_masked': '090***7020',
                },
              },
            ),
          );
        },
      ),
    );
    return ProviderContainer(
      overrides: [
        sharedPreferencesProvider.overrideWithValue(prefs),
        authApiProvider.overrideWithValue(AuthApi(refreshDio)),
      ],
    );
  }

  AuthSession oldSession() => const AuthSession(
        accessToken: 'old-access-token',
        refreshToken: 'old-refresh-token',
        tokenType: 'Bearer',
        expiresIn: 900,
        user: AuthUser(
          id: 'customer-1',
          role: 'customer',
          phoneMasked: '090***7020',
        ),
      );

  test('concurrent refreshes share one refresh-token request', () async {
    var refreshCalls = 0;
    final container = containerWithRefreshApi(() => refreshCalls++);
    addTearDown(container.dispose);
    final notifier = container.read(authSessionProvider.notifier);
    await notifier.setSession(oldSession());

    final results = await Future.wait([notifier.refresh(), notifier.refresh()]);

    expect(results, [true, true]);
    expect(refreshCalls, 1);
    expect(
      container.read(authSessionProvider)?.accessToken,
      'new-access-token',
    );
  });

  test('a 401 refreshes a legacy session then retries the original request',
      () async {
    var refreshCalls = 0;
    final container = containerWithRefreshApi(() => refreshCalls++);
    addTearDown(container.dispose);
    await container.read(authSessionProvider.notifier).setSession(oldSession());

    var apiCalls = 0;
    final dio = container.read(dioProvider);
    dio.interceptors.add(
      InterceptorsWrapper(
        onRequest: (options, handler) {
          apiCalls++;
          if (options.headers['Authorization'] == 'Bearer new-access-token') {
            handler.resolve(
              Response<Map<String, dynamic>>(
                requestOptions: options,
                statusCode: 200,
                data: const {'ok': true},
              ),
            );
            return;
          }
          handler.reject(
            DioException(
              requestOptions: options,
              response:
                  Response<void>(requestOptions: options, statusCode: 401),
              type: DioExceptionType.badResponse,
            ),
            true,
          );
        },
      ),
    );

    final response = await dio.get<Map<String, dynamic>>('/v1/me');

    expect(response.data, const {'ok': true});
    expect(apiCalls, 2);
    expect(refreshCalls, 1);
  });
}
