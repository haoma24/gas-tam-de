import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/api_client.dart';
import 'auth_models.dart';

final adminPhonesApiProvider = Provider<AdminPhonesApi>((ref) {
  return AdminPhonesApi(ref.watch(dioProvider));
});

/// One number on the allow-list that grants admin after an OTP login.
class AdminPhone {
  const AdminPhone({
    required this.id,
    required this.phoneMasked,
    required this.createdAt,
    this.label,
    this.isSelf = false,
  });

  final String id;
  final String phoneMasked;
  final String createdAt;
  final String? label;

  /// True for the number the current admin signed in with — removing it would
  /// revoke their own access.
  final bool isSelf;

  factory AdminPhone.fromJson(Map<String, dynamic> json) {
    final label = (json['label'] as String?)?.trim();
    return AdminPhone(
      id: json['id'] as String? ?? '',
      phoneMasked: json['phone_masked'] as String? ?? '',
      createdAt: json['created_at'] as String? ?? '',
      label: (label == null || label.isEmpty) ? null : label,
      isSelf: json['is_self'] == true,
    );
  }
}

/// Client for `/v1/admin/admin-phones` (auth-service via gateway).
class AdminPhonesApi {
  AdminPhonesApi(this._dio);

  final Dio _dio;

  Future<List<AdminPhone>> list() async {
    try {
      final res =
          await _dio.get<Map<String, dynamic>>('/v1/admin/admin-phones');
      final raw = res.data?['admin_phones'];
      if (raw is! List) return const [];
      return raw
          .whereType<Map>()
          .map((e) => AdminPhone.fromJson(Map<String, dynamic>.from(e)))
          .toList();
    } on DioException catch (e) {
      throw mapDioToAuthException(e);
    }
  }

  Future<AdminPhone> add({required String phone, String? label}) async {
    try {
      final res = await _dio.post<Map<String, dynamic>>(
        '/v1/admin/admin-phones',
        data: {
          'phone': phone.trim(),
          if (label != null && label.trim().isNotEmpty) 'label': label.trim(),
        },
      );
      final data = res.data;
      if (data == null) {
        throw AuthApiException(
          code: 'EMPTY',
          message: 'empty response',
          statusCode: res.statusCode,
        );
      }
      return AdminPhone.fromJson(data);
    } on DioException catch (e) {
      throw mapDioToAuthException(e);
    }
  }

  Future<void> remove(String id) async {
    try {
      await _dio.delete<Map<String, dynamic>>('/v1/admin/admin-phones/$id');
    } on DioException catch (e) {
      throw mapDioToAuthException(e);
    }
  }
}
