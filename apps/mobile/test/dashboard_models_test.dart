import 'package:flutter_test/flutter_test.dart';

import 'package:gas_tam_de/features/dashboard/dashboard_models.dart';

void main() {
  test('DashboardSummary.fromJson maps snake_case fields', () {
    final s = DashboardSummary.fromJson({
      'from': '2026-08-01',
      'to': '2026-08-02',
      'timezone': 'Asia/Ho_Chi_Minh',
      'revenue_vnd': 1_000_000,
      'cogs_vnd': 400_000,
      'delivery_fee_vnd': 50_000,
      'profit_vnd': 600_000,
      'orders_completed': 3,
      'orders_placed': 5,
      'debt_total': 350_000,
    });
    expect(s.from, '2026-08-01');
    expect(s.to, '2026-08-02');
    expect(s.revenueVnd, 1_000_000);
    expect(s.profitVnd, 600_000);
    expect(s.ordersCompleted, 3);
    expect(s.debtTotal, 350_000);
  });

  test('queryForPeriod today uses day=', () {
    final q = queryForPeriod(DashboardPeriod.today);
    expect(q.day, isNotNull);
    expect(q.from, isNull);
    expect(q.to, isNull);
    expect(RegExp(r'^\d{4}-\d{2}-\d{2}$').hasMatch(q.day!), isTrue);
  });

  test('queryForPeriod last7Days and thisMonth use from/to', () {
    final week = queryForPeriod(DashboardPeriod.last7Days);
    expect(week.day, isNull);
    expect(week.from, isNotNull);
    expect(week.to, isNotNull);
    expect(week.from!.compareTo(week.to!), lessThanOrEqualTo(0));

    final month = queryForPeriod(DashboardPeriod.thisMonth);
    expect(month.day, isNull);
    expect(month.from, endsWith('-01'));
    expect(month.to, isNotNull);
  });
}
