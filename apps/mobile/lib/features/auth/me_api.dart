import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/api_client.dart';
import 'auth_models.dart';
import 'auth_session.dart';

final meApiProvider = Provider<MeApi>((ref) {
  return MeApi(ref.watch(dioProvider));
});

class CustomerProfile {
  const CustomerProfile({
    required this.id,
    required this.phoneMasked,
    this.fullName,
  });

  final String id;
  final String phoneMasked;
  final String? fullName;

  bool get hasName => fullName != null && fullName!.trim().isNotEmpty;

  factory CustomerProfile.fromJson(Map<String, dynamic> json) {
    final name = (json['full_name'] as String?)?.trim();
    return CustomerProfile(
      id: json['id'] as String? ?? '',
      phoneMasked: json['phone_masked'] as String? ?? '',
      fullName: (name == null || name.isEmpty) ? null : name,
    );
  }
}

/// Client for `GET/PATCH /v1/me` (auth-service via gateway).
class MeApi {
  MeApi(this._dio);

  final Dio _dio;

  Future<CustomerProfile> getMe() async {
    try {
      final res = await _dio.get<Map<String, dynamic>>('/v1/me');
      final data = res.data;
      if (data == null) {
        throw AuthApiException(
          code: 'EMPTY',
          message: 'empty response',
          statusCode: res.statusCode,
        );
      }
      return CustomerProfile.fromJson(data);
    } on DioException catch (e) {
      throw _mapDio(e);
    }
  }

  Future<CustomerProfile> patchFullName(String fullName) async {
    try {
      final res = await _dio.patch<Map<String, dynamic>>(
        '/v1/me',
        data: {'full_name': fullName.trim()},
      );
      final data = res.data;
      if (data == null) {
        throw AuthApiException(
          code: 'EMPTY',
          message: 'empty response',
          statusCode: res.statusCode,
        );
      }
      return CustomerProfile.fromJson(data);
    } on DioException catch (e) {
      throw _mapDio(e);
    }
  }

  AuthApiException _mapDio(DioException e) => mapDioToAuthException(e);
}

/// Cached profile for the current customer session (null if logged out / admin).
final customerProfileProvider = FutureProvider<CustomerProfile?>((ref) async {
  final session = ref.watch(authSessionProvider);
  if (session == null || !session.isCustomer) return null;
  try {
    return await ref.read(meApiProvider).getMe();
  } catch (_) {
    return null;
  }
});
