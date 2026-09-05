import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/api_client.dart';
import 'auth_models.dart';

final adminAccountsApiProvider = Provider<AdminAccountsApi>((ref) {
  return AdminAccountsApi(ref.watch(dioProvider));
});

class AdminAccount {
  const AdminAccount({
    required this.id,
    required this.username,
    required this.createdAt,
    required this.isSelf,
    this.displayName,
  });

  final String id;
  final String username;
  final String createdAt;
  final bool isSelf;
  final String? displayName;

  factory AdminAccount.fromJson(Map<String, dynamic> json) {
    final displayName = (json['display_name'] as String?)?.trim();
    return AdminAccount(
      id: json['id'] as String? ?? '',
      username: json['username'] as String? ?? '',
      createdAt: json['created_at'] as String? ?? '',
      isSelf: json['is_self'] == true,
      displayName:
          displayName == null || displayName.isEmpty ? null : displayName,
    );
  }
}

class AdminAccountsApi {
  AdminAccountsApi(this._dio);

  final Dio _dio;

  Future<List<AdminAccount>> list() async {
    try {
      final response =
          await _dio.get<Map<String, dynamic>>('/v1/admin/admin-accounts');
      final raw = response.data?['admin_accounts'];
      if (raw is! List) return const [];
      return raw
          .whereType<Map>()
          .map((item) => AdminAccount.fromJson(Map<String, dynamic>.from(item)))
          .toList();
    } on DioException catch (e) {
      throw mapDioToAuthException(e);
    }
  }

  Future<AdminAccount> create({
    required String username,
    required String password,
    String? displayName,
  }) async {
    try {
      final response = await _dio.post<Map<String, dynamic>>(
        '/v1/admin/admin-accounts',
        data: {
          'username': username.trim(),
          'password': password,
          'display_name': displayName?.trim() ?? '',
        },
      );
      return _accountFromResponse(response);
    } on DioException catch (e) {
      throw mapDioToAuthException(e);
    }
  }

  Future<AdminAccount> update({
    required String id,
    required String username,
    required String displayName,
    String? newPassword,
    String? currentPassword,
  }) async {
    try {
      final response = await _dio.patch<Map<String, dynamic>>(
        '/v1/admin/admin-accounts/$id',
        data: {
          'username': username.trim(),
          'display_name': displayName.trim(),
          if (newPassword != null && newPassword.isNotEmpty)
            'new_password': newPassword,
          if (currentPassword != null && currentPassword.isNotEmpty)
            'current_password': currentPassword,
        },
      );
      return _accountFromResponse(response);
    } on DioException catch (e) {
      throw mapDioToAuthException(e);
    }
  }

  AdminAccount _accountFromResponse(Response<Map<String, dynamic>> response) {
    final data = response.data;
    if (data == null) {
      throw AuthApiException(
        code: 'EMPTY',
        message: 'empty response',
        statusCode: response.statusCode,
      );
    }
    return AdminAccount.fromJson(data);
  }
}
