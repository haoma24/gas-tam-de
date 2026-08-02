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
      onRequest: (options, handler) {
        final session = ref.read(authSessionProvider);
        if (session != null) {
          options.headers['Authorization'] =
              '${session.tokenType} ${session.accessToken}';
        }
        handler.next(options);
      },
    ),
  );

  return dio;
});
