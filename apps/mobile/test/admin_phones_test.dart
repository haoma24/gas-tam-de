import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gas_tam_de/features/auth/admin_admin_phones_page.dart';
import 'package:gas_tam_de/features/auth/admin_phones_api.dart';
import 'package:gas_tam_de/features/auth/auth_models.dart';

/// Canned auth-service responses keyed by "METHOD path".
Dio fakeDio(Map<String, Response<dynamic> Function(RequestOptions)> routes) {
  final dio = Dio();
  dio.interceptors.add(
    InterceptorsWrapper(
      onRequest: (options, handler) {
        final route = routes['${options.method} ${options.path}'];
        if (route == null) {
          handler.reject(
            DioException(
              requestOptions: options,
              response: Response<void>(
                requestOptions: options,
                statusCode: 404,
              ),
            ),
            true,
          );
          return;
        }
        handler.resolve(route(options));
      },
    ),
  );
  return dio;
}

Response<Map<String, dynamic>> ok(
  RequestOptions options,
  Map<String, dynamic> data,
) {
  return Response<Map<String, dynamic>>(
    requestOptions: options,
    statusCode: 200,
    data: data,
  );
}

Widget hostPage(Dio dio) {
  return ProviderScope(
    overrides: [
      adminPhonesApiProvider.overrideWithValue(AdminPhonesApi(dio)),
    ],
    child: MaterialApp(home: AdminPhonesPage(onBack: () {})),
  );
}

void main() {
  test('AdminPhone.fromJson reads the auth-service shape', () {
    final phone = AdminPhone.fromJson(const {
      'id': 'p1',
      'phone_masked': '090***7020',
      'label': ' Chủ cửa hàng ',
      'created_at': '2026-08-07T05:00:00Z',
      'is_self': true,
    });

    expect(phone.id, 'p1');
    expect(phone.phoneMasked, '090***7020');
    expect(phone.label, 'Chủ cửa hàng');
    expect(phone.isSelf, isTrue);
  });

  test('a blank label is dropped rather than shown as an empty line', () {
    final phone = AdminPhone.fromJson(const {'id': 'p1', 'label': '  '});
    expect(phone.label, isNull);
    expect(phone.isSelf, isFalse);
  });

  test('LAST_ADMIN_PHONE gets Vietnamese copy', () {
    final e = AuthApiException(code: 'LAST_ADMIN_PHONE', message: 'nope');
    expect(e.displayMessage, 'Phải giữ lại ít nhất một số điện thoại admin.');
  });

  testWidgets('the page lists the allow-list and marks the caller',
      (tester) async {
    final dio = fakeDio({
      'GET /v1/admin/admin-phones': (o) => ok(o, const {
            'admin_phones': [
              {
                'id': 'p1',
                'phone_masked': '090***7020',
                'label': 'Chủ tiệm',
                'is_self': true,
              },
              {'id': 'p2', 'phone_masked': '091***5678'},
            ],
          }),
    });

    await tester.pumpWidget(hostPage(dio));
    await tester.pumpAndSettle();

    expect(find.text('090***7020'), findsOneWidget);
    expect(find.text('Chủ tiệm'), findsOneWidget);
    expect(find.text('091***5678'), findsOneWidget);
    expect(find.text('(bạn)'), findsOneWidget);
  });

  testWidgets('adding a number posts the phone and reloads the list',
      (tester) async {
    var listCalls = 0;
    String? postedPhone;
    final dio = fakeDio({
      'GET /v1/admin/admin-phones': (o) {
        listCalls++;
        return ok(o, {
          'admin_phones': [
            if (listCalls > 1)
              {'id': 'p2', 'phone_masked': '091***5678'},
          ],
        });
      },
      'POST /v1/admin/admin-phones': (o) {
        postedPhone = (o.data as Map)['phone'] as String;
        return ok(o, const {'id': 'p2', 'phone_masked': '091***5678'});
      },
    });

    await tester.pumpWidget(hostPage(dio));
    await tester.pumpAndSettle();

    await tester.enterText(
      find.widgetWithText(TextField, 'Số điện thoại'),
      '0912345678',
    );
    await tester.tap(find.text('Thêm số admin'));
    await tester.pumpAndSettle();

    expect(postedPhone, '0912345678');
    expect(listCalls, 2);
    expect(find.text('091***5678'), findsOneWidget);
  });

  testWidgets('refusing to remove the last entry surfaces the backend reason',
      (tester) async {
    final dio = fakeDio({
      'GET /v1/admin/admin-phones': (o) => ok(o, const {
            'admin_phones': [
              {'id': 'p1', 'phone_masked': '090***7020'},
            ],
          }),
    });
    // DELETE is unmapped above, so give it the real 409 the service returns.
    dio.interceptors.clear();
    dio.interceptors.add(
      InterceptorsWrapper(
        onRequest: (options, handler) {
          if (options.method == 'DELETE') {
            handler.reject(
              DioException(
                requestOptions: options,
                response: Response<Map<String, dynamic>>(
                  requestOptions: options,
                  statusCode: 409,
                  data: const {
                    'error': {
                      'code': 'LAST_ADMIN_PHONE',
                      'message': 'cannot remove the last admin phone',
                    },
                  },
                ),
              ),
              true,
            );
            return;
          }
          handler.resolve(ok(options, const {
            'admin_phones': [
              {'id': 'p1', 'phone_masked': '090***7020'},
            ],
          }));
        },
      ),
    );

    await tester.pumpWidget(hostPage(dio));
    await tester.pumpAndSettle();

    await tester.tap(find.byIcon(Icons.delete_outline));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Bỏ quyền'));
    await tester.pumpAndSettle();

    expect(
      find.text('Phải giữ lại ít nhất một số điện thoại admin.'),
      findsOneWidget,
    );
  });
}
