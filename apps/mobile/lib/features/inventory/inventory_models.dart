/// One stock row from `GET /v1/admin/inventory`.
class StockItem {
  const StockItem({
    required this.productId,
    required this.sku,
    required this.name,
    required this.onHand,
    required this.costPrice,
    required this.reorderLevel,
    required this.updatedAt,
  });

  final String productId;
  final String sku;
  final String name;
  final int onHand;
  final int costPrice;
  final int reorderLevel;
  final String updatedAt;

  bool get isLowStock => onHand <= reorderLevel;

  factory StockItem.fromJson(Map<String, dynamic> json) {
    return StockItem(
      productId: json['product_id'] as String? ?? '',
      sku: json['sku'] as String? ?? '',
      name: json['name'] as String? ?? '',
      onHand: (json['on_hand'] as num?)?.toInt() ?? 0,
      costPrice: (json['cost_price'] as num?)?.toInt() ?? 0,
      reorderLevel: (json['reorder_level'] as num?)?.toInt() ?? 0,
      updatedAt: json['updated_at'] as String? ?? '',
    );
  }
}

/// Movement types for `POST /v1/admin/inventory`.
enum StockMovementType {
  inn('IN'),
  out('OUT'),
  adjust('ADJUST');

  const StockMovementType(this.apiValue);
  final String apiValue;

  String get labelVi {
    switch (this) {
      case StockMovementType.inn:
        return 'Nhập kho';
      case StockMovementType.out:
        return 'Xuất kho';
      case StockMovementType.adjust:
        return 'Điều chỉnh';
    }
  }
}

/// Persisted movement from POST response.
class StockMovement {
  const StockMovement({
    required this.id,
    required this.productId,
    required this.movementType,
    required this.qty,
    required this.delta,
    required this.createdAt,
    this.unitCost,
    this.note,
  });

  final String id;
  final String productId;
  final String movementType;
  final int qty;
  final int delta;
  final int? unitCost;
  final String? note;
  final String createdAt;

  factory StockMovement.fromJson(Map<String, dynamic> json) {
    return StockMovement(
      id: json['id'] as String? ?? '',
      productId: json['product_id'] as String? ?? '',
      movementType: json['movement_type'] as String? ?? '',
      qty: (json['qty'] as num?)?.toInt() ?? 0,
      delta: (json['delta'] as num?)?.toInt() ?? 0,
      unitCost: (json['unit_cost'] as num?)?.toInt(),
      note: json['note'] as String?,
      createdAt: json['created_at'] as String? ?? '',
    );
  }
}

/// List payload from `GET /v1/admin/inventory`.
class StockList {
  const StockList({
    required this.items,
    required this.count,
  });

  final List<StockItem> items;
  final int count;

  factory StockList.fromJson(Map<String, dynamic> json) {
    final raw = json['items'];
    final items = <StockItem>[];
    if (raw is List) {
      for (final e in raw) {
        if (e is Map) {
          items.add(StockItem.fromJson(Map<String, dynamic>.from(e)));
        }
      }
    }
    return StockList(
      items: items,
      count: (json['count'] as num?)?.toInt() ?? items.length,
    );
  }
}

/// Result of `POST /v1/admin/inventory`.
class StockMovementResult {
  const StockMovementResult({
    required this.item,
    required this.movement,
  });

  final StockItem item;
  final StockMovement movement;

  factory StockMovementResult.fromJson(Map<String, dynamic> json) {
    final itemRaw = json['item'];
    final movRaw = json['movement'];
    return StockMovementResult(
      item: StockItem.fromJson(
        itemRaw is Map
            ? Map<String, dynamic>.from(itemRaw)
            : <String, dynamic>{},
      ),
      movement: StockMovement.fromJson(
        movRaw is Map
            ? Map<String, dynamic>.from(movRaw)
            : <String, dynamic>{},
      ),
    );
  }
}
