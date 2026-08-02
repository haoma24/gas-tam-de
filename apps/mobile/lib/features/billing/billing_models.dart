/// One customer debt row from `GET /v1/admin/debts`.
class DebtItem {
  const DebtItem({
    required this.customerKey,
    required this.phoneMasked,
    required this.balance,
    required this.updatedAt,
  });

  final String customerKey;
  final String phoneMasked;
  final int balance;
  final String updatedAt;

  factory DebtItem.fromJson(Map<String, dynamic> json) {
    return DebtItem(
      customerKey: json['customer_key'] as String? ?? '',
      phoneMasked: json['phone_masked'] as String? ?? '',
      balance: (json['balance'] as num?)?.toInt() ?? 0,
      updatedAt: json['updated_at'] as String? ?? '',
    );
  }
}

/// Aggregate list from `GET /v1/admin/debts`.
class DebtsList {
  const DebtsList({
    required this.items,
    required this.totalBalance,
    required this.count,
  });

  final List<DebtItem> items;
  final int totalBalance;
  final int count;

  factory DebtsList.fromJson(Map<String, dynamic> json) {
    final raw = json['items'];
    final items = <DebtItem>[];
    if (raw is List) {
      for (final e in raw) {
        if (e is Map) {
          items.add(DebtItem.fromJson(Map<String, dynamic>.from(e)));
        }
      }
    }
    return DebtsList(
      items: items,
      totalBalance: (json['total_balance'] as num?)?.toInt() ?? 0,
      count: (json['count'] as num?)?.toInt() ?? items.length,
    );
  }
}
