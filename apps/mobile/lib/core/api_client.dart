import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'api_config.dart';
import '../features/auth/auth_session.dart';

/// Shared Dio client for gateway (or auth-service via `API_BASE_URL`).
final dioProvider = Provider<Dio>((ref) {
  final dio = Dio(
    BaseOptions(
      baseUrl: ApiConfig.gatewayBaseUrl,
      connectTimeout: const Duration(seconds: 15),
      receiveTimeout: const Duration(seconds: 15),
      headers: const {'Content-Type': 'application/json'},
    ),
  );

  dio.interceptors.add(
    InterceptorsWrapper(
      onRequest: (options, handler) async {
        if (options.extra['skipAuthRefresh'] == true) {
          handler.next(options);
          return;
        }

        var session = ref.read(authSessionProvider);
        // Refresh before sending when expiry is known. Sessions saved by older
        // releases have no expiry timestamp and are recovered from a 401 below.
        if (session?.accessExpired == true) {
          await ref.read(authSessionProvider.notifier).refresh();
          session = ref.read(authSessionProvider);
        }
        if (session != null) {
          options.headers['Authorization'] =
              '${session.tokenType} ${session.accessToken}';
        }
        handler.next(options);
      },
      onError: (error, handler) async {
        final request = error.requestOptions;
        final alreadyRetried = request.extra['authRefreshed'] == true;
        final skipRefresh = request.extra['skipAuthRefresh'] == true;
        if (error.response?.statusCode != 401 ||
            alreadyRetried ||
            skipRefresh) {
          handler.next(error);
          return;
        }

        final refreshed =
            await ref.read(authSessionProvider.notifier).refresh();
        final session = ref.read(authSessionProvider);
        if (!refreshed || session == null) {
          handler.next(error);
          return;
        }

        final retry = request.copyWith(
          headers: <String, dynamic>{
            ...request.headers,
            'Authorization': '${session.tokenType} ${session.accessToken}',
          },
          extra: <String, dynamic>{
            ...request.extra,
            'authRefreshed': true,
          },
        );
        try {
          handler.resolve(await dio.fetch<dynamic>(retry));
        } on DioException catch (retryError) {
          handler.next(retryError);
        }
      },
    ),
  );

  return dio;
});
