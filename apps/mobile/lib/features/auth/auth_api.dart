import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/api_client.dart';
import 'auth_models.dart';

final authApiProvider = Provider<AuthApi>((ref) {
  return AuthApi(ref.watch(dioProvider));
});

class AuthApi {
  AuthApi(this._dio);

  final Dio _dio;

  Future<OtpRequestResult> requestOtp(String phone) async {
    try {
      final res = await _dio.post<Map<String, dynamic>>(
        '/v1/auth/otp/request',
        data: {'phone': phone},
      );
      final data = res.data;
      if (data == null) {
        throw AuthApiException(
          code: 'EMPTY',
          message: 'empty response',
          statusCode: res.statusCode,
        );
      }
      return OtpRequestResult.fromJson(data);
    } on DioException catch (e) {
      throw _mapDio(e);
    }
  }

  Future<OtpVerifyResult> verifyOtp({
    required String phone,
    required String code,
  }) async {
    try {
      final res = await _dio.post<Map<String, dynamic>>(
        '/v1/auth/otp/verify',
        data: {'phone': phone, 'code': code},
      );
      final data = res.data;
      if (data == null) {
        throw AuthApiException(
          code: 'EMPTY',
          message: 'empty response',
          statusCode: res.statusCode,
        );
      }
      return OtpVerifyResult.fromJson(data);
    } on DioException catch (e) {
      throw _mapDio(e);
    }
  }

  Future<AdminLoginResult> adminLogin({
    required String username,
    required String password,
  }) async {
    try {
      final res = await _dio.post<Map<String, dynamic>>(
        '/v1/auth/admin/login',
        data: {'username': username, 'password': password},
      );
      final data = res.data;
      if (data == null) {
        throw AuthApiException(
          code: 'EMPTY',
          message: 'empty response',
          statusCode: res.statusCode,
        );
      }
      return AdminLoginResult.fromJson(data);
    } on DioException catch (e) {
      throw _mapDio(e);
    }
  }

  /// `POST /v1/auth/refresh` — rotate refresh + issue new access token.
  Future<AuthTokenResult> refresh(String refreshToken) async {
    try {
      final res = await _dio.post<Map<String, dynamic>>(
        '/v1/auth/refresh',
        data: {'refresh_token': refreshToken},
        options: Options(extra: const {'skipAuthRefresh': true}),
      );
      final data = res.data;
      if (data == null) {
        throw AuthApiException(
          code: 'EMPTY',
          message: 'empty response',
          statusCode: res.statusCode,
        );
      }
      return AuthTokenResult.fromJson(data);
    } on DioException catch (e) {
      throw _mapDio(e);
    }
  }

  AuthApiException _mapDio(DioException e) => mapDioToAuthException(e);
}
