/// Aggregate from `GET /v1/admin/dashboard/summary` (T8.1.2 / T8.1.3).
class DashboardSummary {
  const DashboardSummary({
    required this.from,
    required this.to,
    required this.timezone,
    required this.revenueVnd,
    required this.cogsVnd,
    required this.deliveryFeeVnd,
    required this.profitVnd,
    required this.ordersCompleted,
    required this.ordersPlaced,
    required this.debtTotal,
  });

  final String from;
  final String to;
  final String timezone;
  final int revenueVnd;
  final int cogsVnd;
  final int deliveryFeeVnd;
  final int profitVnd;
  final int ordersCompleted;
  final int ordersPlaced;
  final int debtTotal;

  factory DashboardSummary.fromJson(Map<String, dynamic> json) {
    return DashboardSummary(
      from: json['from'] as String? ?? '',
      to: json['to'] as String? ?? '',
      timezone: json['timezone'] as String? ?? 'Asia/Ho_Chi_Minh',
      revenueVnd: (json['revenue_vnd'] as num?)?.toInt() ?? 0,
      cogsVnd: (json['cogs_vnd'] as num?)?.toInt() ?? 0,
      deliveryFeeVnd: (json['delivery_fee_vnd'] as num?)?.toInt() ?? 0,
      profitVnd: (json['profit_vnd'] as num?)?.toInt() ?? 0,
      ordersCompleted: (json['orders_completed'] as num?)?.toInt() ?? 0,
      ordersPlaced: (json['orders_placed'] as num?)?.toInt() ?? 0,
      debtTotal: (json['debt_total'] as num?)?.toInt() ?? 0,
    );
  }
}

/// Preset ranges for dashboard filter chips (VN calendar days).
enum DashboardPeriod {
  today,
  last7Days,
  thisMonth,
}

extension DashboardPeriodLabel on DashboardPeriod {
  String get labelVi {
    switch (this) {
      case DashboardPeriod.today:
        return 'Hôm nay';
      case DashboardPeriod.last7Days:
        return '7 ngày';
      case DashboardPeriod.thisMonth:
        return 'Tháng này';
    }
  }
}

/// Approximate Asia/Ho_Chi_Minh as UTC+7 (no DST).
DateTime nowVn() {
  final utc = DateTime.now().toUtc();
  return utc.add(const Duration(hours: 7));
}

String dayKeyVn(DateTime d) {
  final y = d.year.toString().padLeft(4, '0');
  final m = d.month.toString().padLeft(2, '0');
  final day = d.day.toString().padLeft(2, '0');
  return '$y-$m-$day';
}

/// Query params for [DashboardPeriod] against `GET /v1/admin/dashboard/summary`.
({String? day, String? from, String? to}) queryForPeriod(DashboardPeriod period) {
  final now = nowVn();
  final today = dayKeyVn(now);
  switch (period) {
    case DashboardPeriod.today:
      return (day: today, from: null, to: null);
    case DashboardPeriod.last7Days:
      final from = now.subtract(const Duration(days: 6));
      return (day: null, from: dayKeyVn(from), to: today);
    case DashboardPeriod.thisMonth:
      final from = DateTime(now.year, now.month, 1);
      return (day: null, from: dayKeyVn(from), to: today);
  }
}
