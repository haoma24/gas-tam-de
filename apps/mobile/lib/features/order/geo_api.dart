import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/api_client.dart';
import 'geo_models.dart';

final geoApiProvider = Provider<GeoApi>((ref) {
  return GeoApi(ref.watch(dioProvider));
});

/// Client for geo-service search proxy (never call OSM/Photon from the app).
class GeoApi {
  GeoApi(this._dio);

  final Dio _dio;

  /// `GET /v1/geo/search?q=&limit=` → list of [GeoPlace].
  Future<List<GeoPlace>> search(String query, {int limit = 5}) async {
    final q = query.trim();
    if (q.length < 2) return const [];

    try {
      final res = await _dio.get<Map<String, dynamic>>(
        '/v1/geo/search',
        queryParameters: {'q': q, 'limit': limit},
      );
      final data = res.data;
      if (data == null) {
        throw GeoApiException(
          code: 'EMPTY',
          message: 'empty response',
          statusCode: res.statusCode,
        );
      }
      final items = data['items'];
      if (items is! List) return const [];
      return items
          .whereType<Map>()
          .map((e) => GeoPlace.fromJson(Map<String, dynamic>.from(e)))
          .where((p) => p.label.isNotEmpty)
          .toList();
    } on DioException catch (e) {
      throw _mapDio(e);
    }
  }

  /// `POST /v1/geo/check` `{ lat, lng }` → [GeoCheckResult].
  /// Gateway requires customer JWT; local may point Dio at geo-service directly.
  Future<GeoCheckResult> check({
    required double lat,
    required double lng,
  }) async {
    try {
      final res = await _dio.post<Map<String, dynamic>>(
        '/v1/geo/check',
        data: {'lat': lat, 'lng': lng},
      );
      final data = res.data;
      if (data == null) {
        throw GeoApiException(
          code: 'EMPTY',
          message: 'empty response',
          statusCode: res.statusCode,
        );
      }
      return GeoCheckResult.fromJson(data);
    } on DioException catch (e) {
      throw _mapDio(e);
    }
  }

  /// `GET /v1/geo/store` — public shop coords / radius.
  Future<StoreSettings> getStore() async {
    try {
      final res = await _dio.get<Map<String, dynamic>>('/v1/geo/store');
      final data = res.data;
      if (data == null) {
        throw GeoApiException(
          code: 'EMPTY',
          message: 'empty response',
          statusCode: res.statusCode,
        );
      }
      return StoreSettings.fromJson(data);
    } on DioException catch (e) {
      throw _mapDio(e);
    }
  }

  /// `PUT /v1/admin/geo/store` — admin JWT required.
  Future<StoreSettings> putAdminStore({
    String? name,
    double? lat,
    double? lng,
    double? maxRadiusKm,
    String? addressText,
  }) async {
    try {
      final body = <String, dynamic>{};
      if (name != null) body['name'] = name;
      if (lat != null) body['lat'] = lat;
      if (lng != null) body['lng'] = lng;
      if (maxRadiusKm != null) body['max_radius_km'] = maxRadiusKm;
      if (addressText != null) body['address_text'] = addressText;
      final res = await _dio.put<Map<String, dynamic>>(
        '/v1/admin/geo/store',
        data: body,
      );
      final data = res.data;
      if (data == null) {
        throw GeoApiException(
          code: 'EMPTY',
          message: 'empty response',
          statusCode: res.statusCode,
        );
      }
      return StoreSettings.fromJson(data);
    } on DioException catch (e) {
      throw _mapDio(e);
    }
  }

  GeoApiException _mapDio(DioException e) {
    if (e.type == DioExceptionType.connectionError ||
        e.type == DioExceptionType.connectionTimeout ||
        e.type == DioExceptionType.receiveTimeout ||
        e.type == DioExceptionType.sendTimeout) {
      return GeoApiException(
        code: 'NETWORK',
        message: e.message ?? 'network error',
      );
    }

    final data = e.response?.data;
    if (data is Map<String, dynamic>) {
      final err = data['error'];
      if (err is Map<String, dynamic>) {
        return GeoApiException(
          code: err['code'] as String? ?? 'UNKNOWN',
          message: err['message'] as String? ?? 'request failed',
          statusCode: e.response?.statusCode,
        );
      }
    }

    return GeoApiException(
      code: 'HTTP',
      message: e.message ?? 'request failed',
      statusCode: e.response?.statusCode,
    );
  }
}

/// API error from geo-service `{"error":{"code","message"}}`.
class GeoApiException implements Exception {
  GeoApiException({
    required this.code,
    required this.message,
    this.statusCode,
  });

  final String code;
  final String message;
  final int? statusCode;

  @override
  String toString() => 'GeoApiException($code: $message)';

  String get displayMessage {
    switch (code) {
      case 'INVALID_QUERY':
        return 'Nhập ít nhất 2 ký tự để tìm địa chỉ.';
      case 'RATE_LIMITED':
        return 'Tìm kiếm quá nhanh. Đợi giây lát rồi thử lại.';
      case 'GEOCODE_UPSTREAM':
        return 'Dịch vụ tìm địa chỉ tạm thời không khả dụng. Thử lại sau.';
      case 'INVALID_COORDS':
      case 'INVALID_BODY':
        return 'Tọa độ địa chỉ không hợp lệ. Chọn lại vị trí trên bản đồ.';
      case 'STORE_NOT_CONFIGURED':
        return 'Cửa hàng chưa cấu hình phạm vi giao. Thử lại sau.';
      case 'INVALID_STORE':
      case 'INVALID_NAME':
        return 'Thông tin cửa hàng không hợp lệ. Kiểm tra tọa độ và bán kính.';
      case 'NETWORK':
        return 'Không kết nối được máy chủ geo. Kiểm tra geo-service đang chạy (:8083).';
      default:
        return message.isNotEmpty
            ? message
            : 'Có lỗi khi gọi dịch vụ địa lý. Thử lại.';
    }
  }
}
