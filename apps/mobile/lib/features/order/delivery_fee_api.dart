import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/api_client.dart';
import 'delivery_fee_models.dart';

final deliveryFeeApiProvider = Provider<DeliveryFeeApi>((ref) {
  return DeliveryFeeApi(ref.watch(dioProvider));
});

/// Admin client for `GET/PUT /v1/admin/delivery-fee` (order-service).
class DeliveryFeeApi {
  DeliveryFeeApi(this._dio);

  final Dio _dio;

  Future<DeliveryFeeConfig> getConfig() async {
    try {
      final res =
          await _dio.get<Map<String, dynamic>>('/v1/admin/delivery-fee');
      final data = res.data;
      if (data == null) {
        throw DeliveryFeeApiException(
          code: 'EMPTY',
          message: 'empty response',
          statusCode: res.statusCode,
        );
      }
      return DeliveryFeeConfig.fromJson(data);
    } on DioException catch (e) {
      throw _mapDio(e);
    }
  }

  /// Partial update: pass [enabled] and/or [rules] (non-null replaces all bands).
  Future<DeliveryFeeConfig> putConfig({
    bool? enabled,
    List<DeliveryFeeRule>? rules,
  }) async {
    if (enabled == null && rules == null) {
      throw DeliveryFeeApiException(
        code: 'INVALID_BODY',
        message: 'provide enabled and/or rules',
      );
    }
    try {
      final body = <String, dynamic>{};
      if (enabled != null) body['enabled'] = enabled;
      if (rules != null) {
        body['rules'] = rules.map((r) => r.toPutJson()).toList();
      }
      final res = await _dio.put<Map<String, dynamic>>(
        '/v1/admin/delivery-fee',
        data: body,
      );
      final data = res.data;
      if (data == null) {
        throw DeliveryFeeApiException(
          code: 'EMPTY',
          message: 'empty response',
          statusCode: res.statusCode,
        );
      }
      return DeliveryFeeConfig.fromJson(data);
    } on DioException catch (e) {
      throw _mapDio(e);
    }
  }

  DeliveryFeeApiException _mapDio(DioException e) {
    if (e.type == DioExceptionType.connectionError ||
        e.type == DioExceptionType.connectionTimeout ||
        e.type == DioExceptionType.receiveTimeout ||
        e.type == DioExceptionType.sendTimeout) {
      return DeliveryFeeApiException(
        code: 'NETWORK',
        message: e.message ?? 'network error',
      );
    }

    final data = e.response?.data;
    if (data is Map<String, dynamic>) {
      final err = data['error'];
      if (err is Map<String, dynamic>) {
        return DeliveryFeeApiException(
          code: err['code'] as String? ?? 'UNKNOWN',
          message: err['message'] as String? ?? 'request failed',
          statusCode: e.response?.statusCode,
        );
      }
    }

    return DeliveryFeeApiException(
      code: 'HTTP',
      message: e.message ?? 'request failed',
      statusCode: e.response?.statusCode,
    );
  }
}

/// API error from order-service `{"error":{"code","message"}}`.
class DeliveryFeeApiException implements Exception {
  DeliveryFeeApiException({
    required this.code,
    required this.message,
    this.statusCode,
  });

  final String code;
  final String message;
  final int? statusCode;

  @override
  String toString() => 'DeliveryFeeApiException($code: $message)';

  String get displayMessage {
    switch (code) {
      case 'INVALID_RULES':
        return message.isNotEmpty
            ? message
            : 'Bậc khoảng cách không hợp lệ (trùng / overlap).';
      case 'INVALID_BODY':
        return 'Dữ liệu gửi lên không hợp lệ.';
      case 'FEE_NOT_CONFIGURED':
        return 'Chưa cấu hình phí giao trên máy chủ (cần seed).';
      case 'UNAUTHORIZED':
      case 'FORBIDDEN':
        return 'Cần đăng nhập admin để cấu hình phí giao.';
      case 'NETWORK':
        return 'Không kết nối được máy chủ. Kiểm tra API đang chạy.';
      default:
        return message.isNotEmpty ? message : 'Có lỗi xảy ra. Thử lại.';
    }
  }
}
