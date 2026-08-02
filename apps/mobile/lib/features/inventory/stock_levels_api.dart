import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/api_client.dart';

final stockLevelsApiProvider = Provider<StockLevelsApi>((ref) {
  return StockLevelsApi(ref.watch(dioProvider));
});

class StockLevelsApi {
  StockLevelsApi(this._dio);
  final Dio _dio;

  /// `GET /v1/stock/levels` → map productId → on_hand.
  Future<Map<String, int>> levels() async {
    final res = await _dio.get<Map<String, dynamic>>('/v1/stock/levels');
    final data = res.data;
    final raw = data?['items'];
    final out = <String, int>{};
    if (raw is List) {
      for (final e in raw) {
        if (e is Map) {
          final id = '${e['product_id'] ?? ''}';
          final onHand = (e['on_hand'] as num?)?.toInt() ?? 0;
          if (id.isNotEmpty) out[id] = onHand;
        }
      }
    }
    return out;
  }
}
