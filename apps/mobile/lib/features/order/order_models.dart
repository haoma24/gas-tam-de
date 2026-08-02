/// One line in `POST /v1/orders` / `POST /v1/orders/quote` request body.
class CreateOrderItem {
  const CreateOrderItem({
    required this.productId,
    required this.qty,
  });

  final String productId;
  final int qty;

  Map<String, dynamic> toJson() => {
        'product_id': productId,
        'qty': qty,
      };
}

/// Request body for `POST /v1/orders/quote` (T4.2.1) — no name/address text.
class QuoteOrderRequest {
  const QuoteOrderRequest({
    required this.lat,
    required this.lng,
    required this.items,
  });

  final double lat;
  final double lng;
  final List<CreateOrderItem> items;

  Map<String, dynamic> toJson() => {
        'lat': lat,
        'lng': lng,
        'items': items.map((e) => e.toJson()).toList(),
      };
}

/// Preview from `POST /v1/orders/quote` (200) — distance, fee, totals.
class OrderQuote {
  const OrderQuote({
    required this.distanceKm,
    required this.inRange,
    required this.maxRadiusKm,
    required this.deliveryFee,
    required this.subtotal,
    required this.total,
  });

  final double distanceKm;
  final bool inRange;
  final double maxRadiusKm;
  final int deliveryFee;
  final int subtotal;
  final int total;

  factory OrderQuote.fromJson(Map<String, dynamic> json) {
    return OrderQuote(
      distanceKm: _asDouble(json['distance_km']),
      inRange: json['in_range'] == true,
      maxRadiusKm: _asDouble(json['max_radius_km']),
      deliveryFee: (json['delivery_fee'] as num?)?.toInt() ?? 0,
      subtotal: (json['subtotal'] as num?)?.toInt() ?? 0,
      total: (json['total'] as num?)?.toInt() ?? 0,
    );
  }

  static double _asDouble(Object? v) {
    if (v is num) return v.toDouble();
    if (v is String) return double.tryParse(v) ?? 0;
    return 0;
  }
}

/// Last order snapshot for returning customers (`GET /v1/orders/me/defaults`).
class OrderDefaults {
  const OrderDefaults({
    required this.hasDefaults,
    this.customerName,
    this.addressText,
    this.lat,
    this.lng,
    this.orderedAt,
  });

  final bool hasDefaults;
  final String? customerName;
  final String? addressText;
  final double? lat;
  final double? lng;
  final String? orderedAt;

  factory OrderDefaults.fromJson(Map<String, dynamic> json) {
    final has = json['has_defaults'] == true;
    if (!has) return const OrderDefaults(hasDefaults: false);
    final name = (json['customer_name'] as String?)?.trim();
    final address = (json['address_text'] as String?)?.trim();
    return OrderDefaults(
      hasDefaults: true,
      customerName: (name == null || name.isEmpty) ? null : name,
      addressText: (address == null || address.isEmpty) ? null : address,
      lat: _optDouble(json['lat']),
      lng: _optDouble(json['lng']),
      orderedAt: json['ordered_at'] as String?,
    );
  }

  static double? _optDouble(Object? v) {
    if (v == null) return null;
    if (v is num) return v.toDouble();
    if (v is String) return double.tryParse(v);
    return null;
  }
}

/// Request body for `POST /v1/orders`.
class CreateOrderRequest {
  const CreateOrderRequest({
    required this.customerName,
    required this.addressText,
    required this.lat,
    required this.lng,
    required this.items,
  });

  final String customerName;
  final String addressText;
  final double lat;
  final double lng;
  final List<CreateOrderItem> items;

  Map<String, dynamic> toJson() => {
        'customer_name': customerName,
        'address_text': addressText,
        'lat': lat,
        'lng': lng,
        'items': items.map((e) => e.toJson()).toList(),
      };
}

/// Line item from place-order response.
class OrderItemView {
  const OrderItemView({
    required this.id,
    required this.productId,
    required this.productSku,
    required this.productName,
    required this.unitPrice,
    required this.qty,
    required this.lineTotal,
  });

  final String id;
  final String productId;
  final String productSku;
  final String productName;
  final int unitPrice;
  final int qty;
  final int lineTotal;

  factory OrderItemView.fromJson(Map<String, dynamic> json) {
    return OrderItemView(
      id: json['id'] as String? ?? '',
      productId: json['product_id'] as String? ?? '',
      productSku: json['product_sku'] as String? ?? '',
      productName: json['product_name'] as String? ?? '',
      unitPrice: (json['unit_price'] as num?)?.toInt() ?? 0,
      qty: (json['qty'] as num?)?.toInt() ?? 0,
      lineTotal: (json['line_total'] as num?)?.toInt() ?? 0,
    );
  }
}

/// Created order from `POST /v1/orders` (201).
class PlacedOrder {
  const PlacedOrder({
    required this.id,
    required this.userId,
    required this.customerName,
    required this.phoneMasked,
    required this.addressText,
    required this.lat,
    required this.lng,
    required this.distanceKm,
    required this.deliveryFee,
    required this.subtotal,
    required this.total,
    required this.status,
    required this.createdAt,
    required this.items,
  });

  final String id;
  final String userId;
  final String customerName;
  final String phoneMasked;
  final String addressText;
  final double lat;
  final double lng;
  final double distanceKm;
  final int deliveryFee;
  final int subtotal;
  final int total;
  final String status;
  final String createdAt;
  final List<OrderItemView> items;

  factory PlacedOrder.fromJson(Map<String, dynamic> json) {
    final rawItems = json['items'];
    final items = rawItems is List
        ? rawItems
            .whereType<Map>()
            .map((e) => OrderItemView.fromJson(Map<String, dynamic>.from(e)))
            .toList()
        : const <OrderItemView>[];

    return PlacedOrder(
      id: json['id'] as String? ?? '',
      userId: json['user_id'] as String? ?? '',
      customerName: json['customer_name'] as String? ?? '',
      phoneMasked: json['phone_masked'] as String? ?? '',
      addressText: json['address_text'] as String? ?? '',
      lat: _asDouble(json['lat']),
      lng: _asDouble(json['lng']),
      distanceKm: _asDouble(json['distance_km']),
      deliveryFee: (json['delivery_fee'] as num?)?.toInt() ?? 0,
      subtotal: (json['subtotal'] as num?)?.toInt() ?? 0,
      total: (json['total'] as num?)?.toInt() ?? 0,
      status: json['status'] as String? ?? '',
      createdAt: json['created_at'] as String? ?? '',
      items: items,
    );
  }

  static double _asDouble(Object? v) {
    if (v is num) return v.toDouble();
    if (v is String) return double.tryParse(v) ?? 0;
    return 0;
  }
}

/// Admin Order Desk row from `GET /v1/admin/orders` (FIFO, oldest first).
class AdminOrder {
  const AdminOrder({
    required this.stt,
    required this.id,
    required this.userId,
    required this.customerName,
    required this.phoneMasked,
    required this.addressText,
    required this.lat,
    required this.lng,
    required this.distanceKm,
    required this.deliveryFee,
    required this.subtotal,
    required this.total,
    required this.status,
    required this.createdAt,
    required this.items,
  });

  final int stt;
  final String id;
  final String userId;
  final String customerName;
  final String phoneMasked;
  final String addressText;
  final double lat;
  final double lng;
  final double distanceKm;
  final int deliveryFee;
  final int subtotal;
  final int total;
  final String status;
  final String createdAt;
  final List<OrderItemView> items;

  factory AdminOrder.fromJson(Map<String, dynamic> json) {
    final rawItems = json['items'];
    final items = rawItems is List
        ? rawItems
            .whereType<Map>()
            .map((e) => OrderItemView.fromJson(Map<String, dynamic>.from(e)))
            .toList()
        : const <OrderItemView>[];

    return AdminOrder(
      stt: (json['stt'] as num?)?.toInt() ?? 0,
      id: json['id'] as String? ?? '',
      userId: json['user_id'] as String? ?? '',
      customerName: json['customer_name'] as String? ?? '',
      phoneMasked: json['phone_masked'] as String? ?? '',
      addressText: json['address_text'] as String? ?? '',
      lat: _asDouble(json['lat']),
      lng: _asDouble(json['lng']),
      distanceKm: _asDouble(json['distance_km']),
      deliveryFee: (json['delivery_fee'] as num?)?.toInt() ?? 0,
      subtotal: (json['subtotal'] as num?)?.toInt() ?? 0,
      total: (json['total'] as num?)?.toInt() ?? 0,
      status: json['status'] as String? ?? '',
      createdAt: json['created_at'] as String? ?? '',
      items: items,
    );
  }

  static double _asDouble(Object? v) {
    if (v is num) return v.toDouble();
    if (v is String) return double.tryParse(v) ?? 0;
    return 0;
  }
}

/// Payment types for `POST /v1/admin/orders/{id}/complete` (PRD M6).
abstract final class OrderPaymentType {
  static const full = 'FULL';
  static const partial = 'PARTIAL';
  static const unpaid = 'UNPAID';
}

/// Request body for admin complete (T6.1.4 → T6.1.1).
class CompleteOrderRequest {
  const CompleteOrderRequest({
    required this.paymentType,
    this.amountPaid,
  });

  final String paymentType;

  /// Required for [OrderPaymentType.partial]; omit for FULL / UNPAID.
  final int? amountPaid;

  Map<String, dynamic> toJson() {
    final m = <String, dynamic>{'payment_type': paymentType};
    if (amountPaid != null) {
      m['amount_paid'] = amountPaid;
    }
    return m;
  }
}

/// Response from `POST /v1/admin/orders/{id}/complete`.
class CompletedOrder {
  const CompletedOrder({
    required this.id,
    required this.status,
    required this.completedAt,
    required this.paymentType,
    required this.amountDue,
    required this.amountPaid,
    required this.debt,
    required this.total,
  });

  final String id;
  final String status;
  final String completedAt;
  final String paymentType;
  final int amountDue;
  final int amountPaid;
  final int debt;
  final int total;

  factory CompletedOrder.fromJson(Map<String, dynamic> json) {
    return CompletedOrder(
      id: json['id'] as String? ?? '',
      status: json['status'] as String? ?? '',
      completedAt: json['completed_at'] as String? ?? '',
      paymentType: json['payment_type'] as String? ?? '',
      amountDue: (json['amount_due'] as num?)?.toInt() ?? 0,
      amountPaid: (json['amount_paid'] as num?)?.toInt() ?? 0,
      debt: (json['debt'] as num?)?.toInt() ?? 0,
      total: (json['total'] as num?)?.toInt() ?? 0,
    );
  }
}
