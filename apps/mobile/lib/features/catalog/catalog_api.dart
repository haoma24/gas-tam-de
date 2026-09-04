import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/api_client.dart';
import 'catalog_models.dart';

final catalogApiProvider = Provider<CatalogApi>((ref) {
  return CatalogApi(ref.watch(dioProvider));
});

class CatalogApi {
  CatalogApi(this._dio);

  final Dio _dio;

  /// Customer catalog — `GET /v1/products` (active only).
  Future<List<Product>> listActiveProducts() async {
    return _listProducts('/v1/products');
  }

  Future<List<Product>> listAdminProducts() async {
    return _listProducts('/v1/admin/products');
  }

  Future<List<Product>> _listProducts(String path) async {
    try {
      final res = await _dio.get<Map<String, dynamic>>(path);
      final data = res.data;
      if (data == null) {
        throw CatalogApiException(
          code: 'EMPTY',
          message: 'empty response',
          statusCode: res.statusCode,
        );
      }
      final items = data['items'];
      if (items is! List) return const [];
      return items
          .whereType<Map>()
          .map((e) => Product.fromJson(Map<String, dynamic>.from(e)))
          .toList();
    } on DioException catch (e) {
      throw _mapDio(e);
    }
  }

  Future<Product> getAdminProduct(String id) async {
    try {
      final res =
          await _dio.get<Map<String, dynamic>>('/v1/admin/products/$id');
      final data = res.data;
      if (data == null) {
        throw CatalogApiException(
          code: 'EMPTY',
          message: 'empty response',
          statusCode: res.statusCode,
        );
      }
      return Product.fromJson(data);
    } on DioException catch (e) {
      throw _mapDio(e);
    }
  }

  Future<Product> createProduct({
    required String sku,
    required String name,
    required int salePrice,
    String unit = 'binh',
    String? description,
    bool active = true,
    String? imageUrl,
    List<String>? imageUrls,
  }) async {
    try {
      final body = <String, dynamic>{
        'sku': sku,
        'name': name,
        'sale_price': salePrice,
        'unit': unit,
        'active': active,
      };
      if (description != null && description.trim().isNotEmpty) {
        body['description'] = description.trim();
      }
      if (imageUrl != null && imageUrl.trim().isNotEmpty) {
        body['image_url'] = imageUrl.trim();
      }
      if (imageUrls != null) body['image_urls'] = imageUrls;
      final res = await _dio.post<Map<String, dynamic>>(
        '/v1/admin/products',
        data: body,
      );
      final data = res.data;
      if (data == null) {
        throw CatalogApiException(
          code: 'EMPTY',
          message: 'empty response',
          statusCode: res.statusCode,
        );
      }
      return Product.fromJson(data);
    } on DioException catch (e) {
      throw _mapDio(e);
    }
  }

  Future<Product> patchProduct(
    String id, {
    String? sku,
    String? name,
    String? description,
    String? unit,
    int? salePrice,
    bool? active,
    String? imageUrl,
    List<String>? imageUrls,
  }) async {
    try {
      final body = <String, dynamic>{};
      if (sku != null) body['sku'] = sku;
      if (name != null) body['name'] = name;
      if (description != null) body['description'] = description;
      if (unit != null) body['unit'] = unit;
      if (salePrice != null) body['sale_price'] = salePrice;
      if (active != null) body['active'] = active;
      if (imageUrl != null) body['image_url'] = imageUrl;
      if (imageUrls != null) body['image_urls'] = imageUrls;

      final res = await _dio.patch<Map<String, dynamic>>(
        '/v1/admin/products/$id',
        data: body,
      );
      final data = res.data;
      if (data == null) {
        throw CatalogApiException(
          code: 'EMPTY',
          message: 'empty response',
          statusCode: res.statusCode,
        );
      }
      return Product.fromJson(data);
    } on DioException catch (e) {
      throw _mapDio(e);
    }
  }

  CatalogApiException _mapDio(DioException e) {
    if (e.type == DioExceptionType.connectionError ||
        e.type == DioExceptionType.connectionTimeout ||
        e.type == DioExceptionType.receiveTimeout ||
        e.type == DioExceptionType.sendTimeout) {
      return CatalogApiException(
        code: 'NETWORK',
        message: e.message ?? 'network error',
      );
    }

    final data = e.response?.data;
    if (data is Map<String, dynamic>) {
      final err = data['error'];
      if (err is Map<String, dynamic>) {
        return CatalogApiException(
          code: err['code'] as String? ?? 'UNKNOWN',
          message: err['message'] as String? ?? 'request failed',
          statusCode: e.response?.statusCode,
        );
      }
    }

    return CatalogApiException(
      code: 'HTTP',
      message: e.message ?? 'request failed',
      statusCode: e.response?.statusCode,
    );
  }
}

/// API error from catalog-service `{"error":{"code","message"}}`.
class CatalogApiException implements Exception {
  CatalogApiException({
    required this.code,
    required this.message,
    this.statusCode,
  });

  final String code;
  final String message;
  final int? statusCode;

  @override
  String toString() => 'CatalogApiException($code: $message)';

  String get displayMessage {
    switch (code) {
      case 'VALIDATION_ERROR':
        return message.isNotEmpty
            ? message
            : 'Dữ liệu không hợp lệ. Kiểm tra lại.';
      case 'SKU_EXISTS':
        return 'Mã SKU đã tồn tại. Chọn mã khác.';
      case 'NOT_FOUND':
        return 'Không tìm thấy sản phẩm.';
      case 'INVALID_ID':
        return 'Mã sản phẩm không hợp lệ.';
      case 'NETWORK':
        return 'Không kết nối được máy chủ. Kiểm tra API đang chạy.';
      default:
        return message.isNotEmpty ? message : 'Có lỗi xảy ra. Thử lại.';
    }
  }
}
