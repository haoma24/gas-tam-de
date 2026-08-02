import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/api_client.dart';
import 'inventory_models.dart';

final inventoryApiProvider = Provider<InventoryApi>((ref) {
  return InventoryApi(ref.watch(dioProvider));
});

/// Admin client for inventory (`GET/POST /v1/admin/inventory`).
class InventoryApi {
  InventoryApi(this._dio);

  final Dio _dio;

  /// `GET /v1/admin/inventory` — all stock rows.
  Future<StockList> listStock() async {
    try {
      final res = await _dio.get<Map<String, dynamic>>('/v1/admin/inventory');
      final data = res.data;
      if (data == null) {
        throw InventoryApiException(
          code: 'EMPTY',
          message: 'empty response',
          statusCode: res.statusCode,
        );
      }
      return StockList.fromJson(data);
    } on DioException catch (e) {
      throw _mapDio(e);
    }
  }

  /// `POST /v1/admin/inventory` — IN / OUT / ADJUST.
  Future<StockMovementResult> postMovement({
    required StockMovementType movementType,
    required String productId,
    int? qty,
    int? delta,
    int? unitCost,
    String? sku,
    String? name,
    int? reorderLevel,
    String? note,
  }) async {
    try {
      final body = <String, dynamic>{
        'movement_type': movementType.apiValue,
        'product_id': productId,
      };
      if (qty != null) body['qty'] = qty;
      if (delta != null) body['delta'] = delta;
      if (unitCost != null) body['unit_cost'] = unitCost;
      if (sku != null && sku.trim().isNotEmpty) body['sku'] = sku.trim();
      if (name != null && name.trim().isNotEmpty) body['name'] = name.trim();
      if (reorderLevel != null) body['reorder_level'] = reorderLevel;
      if (note != null && note.trim().isNotEmpty) body['note'] = note.trim();

      final res = await _dio.post<Map<String, dynamic>>(
        '/v1/admin/inventory',
        data: body,
      );
      final data = res.data;
      if (data == null) {
        throw InventoryApiException(
          code: 'EMPTY',
          message: 'empty response',
          statusCode: res.statusCode,
        );
      }
      return StockMovementResult.fromJson(data);
    } on DioException catch (e) {
      throw _mapDio(e);
    }
  }

  InventoryApiException _mapDio(DioException e) {
    if (e.type == DioExceptionType.connectionError ||
        e.type == DioExceptionType.connectionTimeout ||
        e.type == DioExceptionType.receiveTimeout ||
        e.type == DioExceptionType.sendTimeout) {
      return InventoryApiException(
        code: 'NETWORK',
        message: e.message ?? 'network error',
      );
    }

    final data = e.response?.data;
    if (data is Map<String, dynamic>) {
      final err = data['error'];
      if (err is Map<String, dynamic>) {
        return InventoryApiException(
          code: err['code'] as String? ?? 'UNKNOWN',
          message: err['message'] as String? ?? 'request failed',
          statusCode: e.response?.statusCode,
        );
      }
    }

    return InventoryApiException(
      code: 'HTTP',
      message: e.message ?? 'request failed',
      statusCode: e.response?.statusCode,
    );
  }
}

/// API error from inventory-service `{"error":{"code","message"}}`.
class InventoryApiException implements Exception {
  InventoryApiException({
    required this.code,
    required this.message,
    this.statusCode,
  });

  final String code;
  final String message;
  final int? statusCode;

  @override
  String toString() => 'InventoryApiException($code: $message)';

  String get displayMessage {
    switch (code) {
      case 'FORBIDDEN':
      case 'UNAUTHORIZED':
        return 'Cần đăng nhập admin để quản lý tồn kho.';
      case 'INVALID_QTY':
        return 'Số lượng phải lớn hơn 0.';
      case 'INVALID_DELTA':
        return 'Delta điều chỉnh phải khác 0.';
      case 'INVALID_UNIT_COST':
        return 'Giá nhập không hợp lệ (≥ 0).';
      case 'INVALID_PRODUCT':
        return message.isNotEmpty
            ? message
            : 'Thiếu mã sản phẩm / SKU / tên khi tạo tồn.';
      case 'INVALID_TYPE':
        return 'Loại phiếu không hợp lệ (IN / OUT / ADJUST).';
      case 'NOT_FOUND':
        return 'Chưa có tồn kho cho sản phẩm này — dùng Nhập kho trước.';
      case 'SKU_CONFLICT':
        return 'Mã SKU đã tồn tại trong kho.';
      case 'NETWORK':
        return 'Không kết nối được máy chủ. Kiểm tra API đang chạy.';
      default:
        return message.isNotEmpty ? message : 'Có lỗi xảy ra. Thử lại.';
    }
  }
}
