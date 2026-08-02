import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/api_client.dart';
import 'billing_models.dart';

final billingApiProvider = Provider<BillingApi>((ref) {
  return BillingApi(ref.watch(dioProvider));
});

/// Admin client for billing debts (`GET /v1/admin/debts`).
class BillingApi {
  BillingApi(this._dio);

  final Dio _dio;

  /// `GET /v1/admin/debts` — outstanding balances + aggregate total.
  Future<DebtsList> listDebts() async {
    try {
      final res = await _dio.get<Map<String, dynamic>>('/v1/admin/debts');
      final data = res.data;
      if (data == null) {
        throw BillingApiException(
          code: 'EMPTY',
          message: 'empty response',
          statusCode: res.statusCode,
        );
      }
      return DebtsList.fromJson(data);
    } on DioException catch (e) {
      throw _mapDio(e);
    }
  }

  BillingApiException _mapDio(DioException e) {
    if (e.type == DioExceptionType.connectionError ||
        e.type == DioExceptionType.connectionTimeout ||
        e.type == DioExceptionType.receiveTimeout ||
        e.type == DioExceptionType.sendTimeout) {
      return BillingApiException(
        code: 'NETWORK',
        message: e.message ?? 'network error',
      );
    }

    final data = e.response?.data;
    if (data is Map<String, dynamic>) {
      final err = data['error'];
      if (err is Map<String, dynamic>) {
        return BillingApiException(
          code: err['code'] as String? ?? 'UNKNOWN',
          message: err['message'] as String? ?? 'request failed',
          statusCode: e.response?.statusCode,
        );
      }
    }

    return BillingApiException(
      code: 'HTTP',
      message: e.message ?? 'request failed',
      statusCode: e.response?.statusCode,
    );
  }
}

/// API error from billing-service `{"error":{"code","message"}}`.
class BillingApiException implements Exception {
  BillingApiException({
    required this.code,
    required this.message,
    this.statusCode,
  });

  final String code;
  final String message;
  final int? statusCode;

  @override
  String toString() => 'BillingApiException($code: $message)';

  String get displayMessage {
    switch (code) {
      case 'FORBIDDEN':
      case 'UNAUTHORIZED':
        return 'Cần đăng nhập admin để xem công nợ.';
      case 'NETWORK':
        return 'Không kết nối được máy chủ. Kiểm tra API đang chạy.';
      default:
        return message.isNotEmpty ? message : 'Có lỗi xảy ra. Thử lại.';
    }
  }
}
