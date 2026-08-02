import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/api_client.dart';
import '../auth/auth_session.dart';
import 'order_models.dart';

final orderApiProvider = Provider<OrderApi>((ref) {
  return OrderApi(ref.watch(dioProvider), ref);
});

/// Client for orders: place/quote (customer) + admin Order Desk list.
///
/// Local: point `API_BASE_URL` at order-service (`:8084`). Customer identity
/// headers `X-User-*` are filled from [authSessionProvider] so direct calls
/// work without a real gateway reverse-proxy (gateway route is still stub).
/// Admin list uses Bearer from the same Dio interceptor.
class OrderApi {
  OrderApi(this._dio, this._ref);

  final Dio _dio;
  final Ref _ref;

  Map<String, dynamic> _customerHeaders() {
    final session = _ref.read(authSessionProvider);
    final headers = <String, dynamic>{};
    if (session != null) {
      headers['X-User-Id'] = session.user.id;
      headers['X-User-Role'] = session.user.role;
      if (session.user.phoneMasked.isNotEmpty) {
        headers['X-Phone-Masked'] = session.user.phoneMasked;
      }
    }
    return headers;
  }

  /// `GET /v1/admin/orders` — FIFO desk list (oldest first). Default PENDING.
  Future<List<AdminOrder>> listAdminOrders({String? status}) async {
    try {
      final query = <String, dynamic>{};
      if (status != null && status.trim().isNotEmpty) {
        query['status'] = status.trim().toUpperCase();
      }
      final res = await _dio.get<Map<String, dynamic>>(
        '/v1/admin/orders',
        queryParameters: query.isEmpty ? null : query,
      );
      final data = res.data;
      if (data == null) {
        throw OrderApiException(
          code: 'EMPTY',
          message: 'empty response',
          statusCode: res.statusCode,
        );
      }
      final raw = data['orders'];
      if (raw is! List) return const [];
      return raw
          .whereType<Map>()
          .map((e) => AdminOrder.fromJson(Map<String, dynamic>.from(e)))
          .toList();
    } on DioException catch (e) {
      throw _mapDio(e);
    }
  }

  /// `GET /v1/orders/me` — customer's order history.
  Future<List<AdminOrder>> listMyOrders() async {
    try {
      final res = await _dio.get<Map<String, dynamic>>(
        '/v1/orders/me',
        options: Options(headers: _customerHeaders()),
      );
      final data = res.data;
      if (data == null) {
        throw OrderApiException(
          code: 'EMPTY',
          message: 'empty response',
          statusCode: res.statusCode,
        );
      }
      final raw = data['orders'];
      if (raw is! List) return const [];
      return raw
          .whereType<Map>()
          .map((e) => AdminOrder.fromJson(Map<String, dynamic>.from(e)))
          .toList();
    } on DioException catch (e) {
      throw _mapDio(e);
    }
  }

  /// `POST /v1/orders/{id}/cancel` — cancel own PENDING order.
  Future<void> cancelMyOrder(String orderId) async {
    final id = orderId.trim();
    if (id.isEmpty) {
      throw OrderApiException(code: 'INVALID_ID', message: 'order id required');
    }
    try {
      await _dio.post<Map<String, dynamic>>(
        '/v1/orders/$id/cancel',
        options: Options(headers: _customerHeaders()),
      );
    } on DioException catch (e) {
      throw _mapDio(e);
    }
  }

  /// `POST /v1/orders/quote` — preview distance + fee + totals (no persist).
  Future<OrderQuote> quoteOrder(QuoteOrderRequest request) async {
    try {
      final res = await _dio.post<Map<String, dynamic>>(
        '/v1/orders/quote',
        data: request.toJson(),
        options: Options(headers: _customerHeaders()),
      );
      final data = res.data;
      if (data == null) {
        throw OrderApiException(
          code: 'EMPTY',
          message: 'empty response',
          statusCode: res.statusCode,
        );
      }
      return OrderQuote.fromJson(data);
    } on DioException catch (e) {
      throw _mapDio(e);
    }
  }

  /// `GET /v1/orders/me/defaults` — last name + address for returning customer.
  Future<OrderDefaults> getMyDefaults() async {
    try {
      final res = await _dio.get<Map<String, dynamic>>(
        '/v1/orders/me/defaults',
        options: Options(headers: _customerHeaders()),
      );
      final data = res.data;
      if (data == null) {
        throw OrderApiException(
          code: 'EMPTY',
          message: 'empty response',
          statusCode: res.statusCode,
        );
      }
      return OrderDefaults.fromJson(data);
    } on DioException catch (e) {
      throw _mapDio(e);
    }
  }

  Future<PlacedOrder> createOrder(CreateOrderRequest request) async {
    try {
      final res = await _dio.post<Map<String, dynamic>>(
        '/v1/orders',
        data: request.toJson(),
        options: Options(headers: _customerHeaders()),
      );
      final data = res.data;
      if (data == null) {
        throw OrderApiException(
          code: 'EMPTY',
          message: 'empty response',
          statusCode: res.statusCode,
        );
      }
      return PlacedOrder.fromJson(data);
    } on DioException catch (e) {
      throw _mapDio(e);
    }
  }

  /// `POST /v1/admin/orders/{id}/complete` — settle FULL / PARTIAL / UNPAID.
  Future<CompletedOrder> completeOrder(
    String orderId,
    CompleteOrderRequest request,
  ) async {
    final id = orderId.trim();
    if (id.isEmpty) {
      throw OrderApiException(
        code: 'INVALID_ID',
        message: 'order id is required',
      );
    }
    try {
      final res = await _dio.post<Map<String, dynamic>>(
        '/v1/admin/orders/$id/complete',
        data: request.toJson(),
      );
      final data = res.data;
      if (data == null) {
        throw OrderApiException(
          code: 'EMPTY',
          message: 'empty response',
          statusCode: res.statusCode,
        );
      }
      return CompletedOrder.fromJson(data);
    } on DioException catch (e) {
      throw _mapDio(e);
    }
  }

  OrderApiException _mapDio(DioException e) {
    if (e.type == DioExceptionType.connectionError ||
        e.type == DioExceptionType.connectionTimeout ||
        e.type == DioExceptionType.receiveTimeout ||
        e.type == DioExceptionType.sendTimeout) {
      return OrderApiException(
        code: 'NETWORK',
        message: e.message ?? 'network error',
      );
    }

    final data = e.response?.data;
    if (data is Map<String, dynamic>) {
      final err = data['error'];
      if (err is Map<String, dynamic>) {
        return OrderApiException(
          code: err['code'] as String? ?? 'UNKNOWN',
          message: err['message'] as String? ?? 'request failed',
          statusCode: e.response?.statusCode,
          distanceKm: _asDouble(err['distance_km']),
          maxRadiusKm: _asDouble(err['max_radius_km']),
        );
      }
    }

    return OrderApiException(
      code: 'HTTP',
      message: e.message ?? 'request failed',
      statusCode: e.response?.statusCode,
    );
  }

  static double? _asDouble(Object? v) {
    if (v == null) return null;
    if (v is num) return v.toDouble();
    if (v is String) return double.tryParse(v);
    return null;
  }
}

/// API error from order-service `{"error":{"code","message",...}}`.
class OrderApiException implements Exception {
  OrderApiException({
    required this.code,
    required this.message,
    this.statusCode,
    this.distanceKm,
    this.maxRadiusKm,
  });

  final String code;
  final String message;
  final int? statusCode;
  final double? distanceKm;
  final double? maxRadiusKm;

  @override
  String toString() => 'OrderApiException($code: $message)';

  String get displayMessage {
    switch (code) {
      case 'UNAUTHORIZED':
        return 'Phiên đăng nhập hết hạn. Vui lòng đăng nhập lại.';
      case 'FORBIDDEN':
        return 'Không có quyền thực hiện thao tác này.';
      case 'OUT_OF_RANGE':
        final dist = distanceKm;
        final max = maxRadiusKm;
        if (dist != null && max != null) {
          return 'Địa chỉ ngoài phạm vi giao '
              '(khoảng ${_fmtKm(dist)} km, tối đa ${_fmtKm(max)} km). '
              'Quay lại chọn vị trí gần hơn.';
        }
        return 'Địa chỉ ngoài phạm vi giao hàng. Quay lại chọn vị trí gần hơn.';
      case 'PRODUCT_NOT_FOUND':
        return 'Một sản phẩm không còn bán. Quay lại chọn lại giỏ hàng.';
      case 'GEO_UNAVAILABLE':
      case 'CATALOG_UNAVAILABLE':
        return 'Không kiểm tra được đơn. Thử lại sau.';
      case 'VALIDATION_ERROR':
        return message.isNotEmpty
            ? message
            : 'Thông tin đơn không hợp lệ. Kiểm tra lại.';
      case 'ORDER_ALREADY_COMPLETED':
        return 'Đơn đã hoàn tất trước đó.';
      case 'ORDER_NOT_COMPLETABLE':
        return 'Không thể hoàn tất đơn này.';
      case 'NOT_FOUND':
        return 'Không tìm thấy đơn hàng.';
      case 'INVALID_COORDS':
        return 'Tọa độ địa chỉ không hợp lệ. Quay lại chọn lại.';
      case 'NETWORK':
        return 'Không kết nối được máy chủ. Kiểm tra API đang chạy.';
      default:
        return message.isNotEmpty ? message : 'Có lỗi xảy ra. Thử lại.';
    }
  }

  static String _fmtKm(double km) {
    if (km == km.roundToDouble()) return km.toStringAsFixed(0);
    return km.toStringAsFixed(2);
  }
}
