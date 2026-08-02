import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/api_client.dart';
import 'dashboard_models.dart';

final dashboardApiProvider = Provider<DashboardApi>((ref) {
  return DashboardApi(ref.watch(dioProvider));
});

/// Admin client for report dashboard (`GET /v1/admin/dashboard/summary`).
class DashboardApi {
  DashboardApi(this._dio);

  final Dio _dio;

  /// `GET /v1/admin/dashboard/summary` — omit params = today (VN).
  /// Use either [day] **or** [from]+[to] (inclusive).
  Future<DashboardSummary> fetchSummary({
    String? day,
    String? from,
    String? to,
  }) async {
    final query = <String, dynamic>{};
    if (day != null && day.isNotEmpty) {
      query['day'] = day;
    } else if (from != null &&
        from.isNotEmpty &&
        to != null &&
        to.isNotEmpty) {
      query['from'] = from;
      query['to'] = to;
    }

    try {
      final res = await _dio.get<Map<String, dynamic>>(
        '/v1/admin/dashboard/summary',
        queryParameters: query.isEmpty ? null : query,
      );
      final data = res.data;
      if (data == null) {
        throw DashboardApiException(
          code: 'EMPTY',
          message: 'empty response',
          statusCode: res.statusCode,
        );
      }
      return DashboardSummary.fromJson(data);
    } on DioException catch (e) {
      throw _mapDio(e);
    }
  }

  Future<DashboardSummary> fetchForPeriod(DashboardPeriod period) {
    final q = queryForPeriod(period);
    return fetchSummary(day: q.day, from: q.from, to: q.to);
  }

  DashboardApiException _mapDio(DioException e) {
    if (e.type == DioExceptionType.connectionError ||
        e.type == DioExceptionType.connectionTimeout ||
        e.type == DioExceptionType.receiveTimeout ||
        e.type == DioExceptionType.sendTimeout) {
      return DashboardApiException(
        code: 'NETWORK',
        message: e.message ?? 'network error',
      );
    }

    final data = e.response?.data;
    if (data is Map<String, dynamic>) {
      final err = data['error'];
      if (err is Map<String, dynamic>) {
        return DashboardApiException(
          code: err['code'] as String? ?? 'UNKNOWN',
          message: err['message'] as String? ?? 'request failed',
          statusCode: e.response?.statusCode,
        );
      }
    }

    return DashboardApiException(
      code: 'HTTP',
      message: e.message ?? 'request failed',
      statusCode: e.response?.statusCode,
    );
  }
}

/// API error from report-service `{"error":{"code","message"}}`.
class DashboardApiException implements Exception {
  DashboardApiException({
    required this.code,
    required this.message,
    this.statusCode,
  });

  final String code;
  final String message;
  final int? statusCode;

  @override
  String toString() => 'DashboardApiException($code: $message)';

  String get displayMessage {
    switch (code) {
      case 'FORBIDDEN':
      case 'UNAUTHORIZED':
        return 'Cần đăng nhập admin để xem dashboard.';
      case 'NETWORK':
        return 'Không kết nối được máy chủ. Kiểm tra API đang chạy.';
      case 'BAD_REQUEST':
        return message.isNotEmpty ? message : 'Khoảng ngày không hợp lệ.';
      default:
        return message.isNotEmpty ? message : 'Có lỗi xảy ra. Thử lại.';
    }
  }
}
