import 'package:flutter_test/flutter_test.dart';
import 'package:gas_tam_de/features/dashboard/dashboard_models.dart';
import 'package:gas_tam_de/features/order/customer_stats_models.dart';

/// Models behind the «Khách hàng» section of the Báo cáo tab — the shop owner's
/// «khách nào đã đặt bao nhiêu đơn».

void main() {
  test('parses a customer row', () {
    final stat = CustomerStat.fromJson({
      'user_id': 'user-a',
      'customer_name': 'Chị Lan',
      'customer_phone': '0909777020',
      'phone_masked': '090***7020',
      'address_text': '1 Lê Lợi',
      'orders_total': 9,
      'orders_completed': 7,
      'orders_cancelled': 1,
      'orders_pending': 1,
      'spent_vnd': 2450000,
      'paid_vnd': 2100000,
      'debt_vnd': 350000,
      'first_order_at': '2026-08-01T03:00:00Z',
      'last_order_at': '2026-08-30T03:00:00Z',
    });

    expect(stat.displayName, 'Chị Lan');
    expect(stat.displayPhone, '0909777020');
    expect(stat.ordersTotal, 9);
    expect(stat.ordersCompleted, 7);
    expect(stat.debtVnd, 350000);
    expect(stat.lastOrderAt, '2026-08-30T03:00:00Z');
  });

  test('falls back to the masked number, then to a dash', () {
    final masked = CustomerStat.fromJson({
      'user_id': 'user-b',
      'customer_name': '',
      'phone_masked': '090***7020',
    });
    expect(masked.displayPhone, '090***7020');
    expect(masked.customerPhone, isEmpty);
    expect(masked.displayName, 'Khách chưa đặt tên');

    final nothing = CustomerStat.fromJson({'user_id': 'user-c'});
    expect(nothing.displayPhone, '—');
  });

  test('parses the list envelope and tolerates a missing total', () {
    final list = CustomerStatsList.fromJson({
      'from': '2026-08-01',
      'to': '2026-08-31',
      'customers': [
        {'user_id': 'user-a', 'orders_total': 3},
        {'user_id': 'user-b', 'orders_total': 1},
      ],
    });

    expect(list.from, '2026-08-01');
    expect(list.customers, hasLength(2));
    expect(list.total, 2);
    expect(list.isEmpty, isFalse);

    expect(CustomerStatsList.fromJson(const {}).isEmpty, isTrue);
  });

  test('period maps to an inclusive from/to range', () {
    final today = rangeForPeriod(DashboardPeriod.today);
    expect(today.from, today.to, reason: 'a single day is from==to');

    final week = rangeForPeriod(DashboardPeriod.last7Days);
    expect(week.from.compareTo(week.to) < 0, isTrue);

    final month = rangeForPeriod(DashboardPeriod.thisMonth);
    expect(month.from.endsWith('-01'), isTrue,
        reason: 'the month range starts on the 1st');
  });
}
