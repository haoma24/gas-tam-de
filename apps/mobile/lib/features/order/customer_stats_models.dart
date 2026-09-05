/// Một khách hàng trong `GET /v1/admin/orders/customers`.
///
/// Trả lời câu hỏi của chủ shop: «khách nào đã đặt bao nhiêu đơn».
class CustomerStat {
  const CustomerStat({
    required this.userId,
    required this.customerName,
    required this.customerPhone,
    required this.phoneMasked,
    required this.addressText,
    required this.ordersTotal,
    required this.ordersCompleted,
    required this.ordersCancelled,
    required this.ordersPending,
    required this.spentVnd,
    required this.paidVnd,
    required this.debtVnd,
    required this.firstOrderAt,
    required this.lastOrderAt,
  });

  final String userId;
  final String customerName;

  /// Số thật (rỗng khi auth-service chưa có số của khách).
  final String customerPhone;
  final String phoneMasked;
  final String addressText;

  final int ordersTotal;
  final int ordersCompleted;
  final int ordersCancelled;
  final int ordersPending;

  /// Tiền chỉ tính đơn đã hoàn tất — cùng luật với báo cáo doanh thu.
  final int spentVnd;
  final int paidVnd;
  final int debtVnd;

  final String firstOrderAt;
  final String lastOrderAt;

  String get displayName =>
      customerName.trim().isEmpty ? 'Khách chưa đặt tên' : customerName.trim();

  String get displayPhone {
    if (customerPhone.isNotEmpty) return customerPhone;
    if (phoneMasked.isNotEmpty) return phoneMasked;
    return '—';
  }

  factory CustomerStat.fromJson(Map<String, dynamic> json) {
    int i(String key) => (json[key] as num?)?.toInt() ?? 0;
    String s(String key) => json[key] as String? ?? '';

    return CustomerStat(
      userId: s('user_id'),
      customerName: s('customer_name'),
      customerPhone: s('customer_phone'),
      phoneMasked: s('phone_masked'),
      addressText: s('address_text'),
      ordersTotal: i('orders_total'),
      ordersCompleted: i('orders_completed'),
      ordersCancelled: i('orders_cancelled'),
      ordersPending: i('orders_pending'),
      spentVnd: i('spent_vnd'),
      paidVnd: i('paid_vnd'),
      debtVnd: i('debt_vnd'),
      firstOrderAt: s('first_order_at'),
      lastOrderAt: s('last_order_at'),
    );
  }
}

/// Toàn bộ phản hồi của `GET /v1/admin/orders/customers`.
class CustomerStatsList {
  const CustomerStatsList({
    required this.from,
    required this.to,
    required this.customers,
    required this.total,
  });

  final String from;
  final String to;
  final List<CustomerStat> customers;

  /// Tổng số khách trong kỳ trước khi cắt theo `limit`.
  final int total;

  bool get isEmpty => customers.isEmpty;

  factory CustomerStatsList.fromJson(Map<String, dynamic> json) {
    final raw = json['customers'];
    final customers = raw is List
        ? raw
            .whereType<Map>()
            .map((e) => CustomerStat.fromJson(Map<String, dynamic>.from(e)))
            .toList()
        : const <CustomerStat>[];
    return CustomerStatsList(
      from: json['from'] as String? ?? '',
      to: json['to'] as String? ?? '',
      customers: customers,
      total: (json['total'] as num?)?.toInt() ?? customers.length,
    );
  }
}
