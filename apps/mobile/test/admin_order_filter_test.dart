import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gas_tam_de/features/order/admin_orders_page.dart';
import 'package:gas_tam_de/features/order/order_api.dart';
import 'package:gas_tam_de/features/order/order_models.dart';

/// The reported bug: a completed order dropped off the Order Desk with no way
/// to look at it again, because the page always asked for PENDING. These tests
/// pin the status filter that fixes it.

Map<String, dynamic> _order({
  required String id,
  required String status,
  String customerName = 'Chi Lan',
  String customerPhone = '0909777020',
  String phoneMasked = '090***7020',
  String completedAt = '',
  String paymentType = '',
  int amountPaid = 0,
  int stt = 0,
}) {
  return {
    'stt': stt,
    'id': id,
    'user_id': 'user-$id',
    'customer_name': customerName,
    'phone_masked': phoneMasked,
    if (customerPhone.isNotEmpty) 'customer_phone': customerPhone,
    'address_text': '1 Lê Lợi',
    'lat': 10.78,
    'lng': 106.70,
    'distance_km': 2.5,
    'delivery_fee': 0,
    'subtotal': 450000,
    'total': 450000,
    'status': status,
    'created_at': '2026-08-02T09:00:00Z',
    if (completedAt.isNotEmpty) 'completed_at': completedAt,
    if (paymentType.isNotEmpty) 'payment_type': paymentType,
    if (amountPaid > 0) 'amount_paid': amountPaid,
    'items': const <Map<String, dynamic>>[],
  };
}

/// Dio that answers the desk endpoints and records every `status` asked for.
Dio _fakeDio({required List<String> statusLog}) {
  final dio = Dio();
  dio.interceptors.add(
    InterceptorsWrapper(
      onRequest: (options, handler) {
        Response<Map<String, dynamic>> ok(Map<String, dynamic> data) {
          return Response<Map<String, dynamic>>(
            requestOptions: options,
            statusCode: 200,
            data: data,
          );
        }

        if (options.path == '/v1/admin/orders') {
          final status = '${options.queryParameters['status'] ?? ''}';
          statusLog.add(status);
          final orders = switch (status) {
            'COMPLETED' => [
                _order(
                  id: 'done',
                  status: OrderStatus.completed,
                  customerName: 'Khach Da Giao',
                  completedAt: '2026-08-02T10:00:00Z',
                  paymentType: OrderPaymentType.partial,
                  amountPaid: 200000,
                ),
              ],
            'CANCELLED' => [
                _order(
                  id: 'void',
                  status: OrderStatus.cancelled,
                  customerName: 'Khach Bi Huy',
                ),
              ],
            'ALL' => [
                _order(id: 'p1', status: OrderStatus.pending, stt: 1),
                _order(id: 'done', status: OrderStatus.completed),
              ],
            _ => [
                _order(
                  id: 'p1',
                  status: OrderStatus.pending,
                  customerName: 'Khach Cho Giao',
                  stt: 1,
                ),
              ],
          };
          handler.resolve(ok({'orders': orders, 'count': orders.length}));
          return;
        }
        if (options.path == '/v1/admin/desk-settings') {
          handler.resolve(ok({
            'wait_blue_max_min': 5,
            'wait_orange_max_min': 15,
            'wait_red_max_min': 30,
            'alert_enabled': false,
            'alert_interval_sec': 300,
            'updated_at': '2026-08-01T00:00:00Z',
          }));
          return;
        }
        handler.reject(
          DioException(
            requestOptions: options,
            response: Response<void>(requestOptions: options, statusCode: 404),
          ),
          true,
        );
      },
    ),
  );
  return dio;
}

Widget _host(Dio dio) {
  return ProviderScope(
    overrides: [
      orderApiProvider.overrideWith((ref) => OrderApi(dio, ref)),
    ],
    child: const MaterialApp(home: AdminOrdersPage()),
  );
}

/// The page runs periodic timers, so `pumpAndSettle` would never return —
/// pump a few explicit frames instead, then tear the tree down so `dispose`
/// cancels them.
Future<void> _settle(WidgetTester tester) async {
  for (var i = 0; i < 4; i++) {
    await tester.pump(const Duration(milliseconds: 10));
  }
}

Future<void> _teardown(WidgetTester tester) async {
  await tester.pumpWidget(const SizedBox.shrink());
  await tester.pump();
}

void main() {
  test('filter chips map to the API status values', () {
    expect(AdminOrderFilter.pending.apiStatus, 'PENDING');
    expect(AdminOrderFilter.completed.apiStatus, 'COMPLETED');
    expect(AdminOrderFilter.cancelled.apiStatus, 'CANCELLED');
    expect(AdminOrderFilter.all.apiStatus, 'ALL');
  });

  test('only the pending queue polls and announces new orders', () {
    expect(AdminOrderFilter.pending.isLiveQueue, isTrue);
    for (final f in [
      AdminOrderFilter.completed,
      AdminOrderFilter.cancelled,
      AdminOrderFilter.all,
    ]) {
      expect(f.isLiveQueue, isFalse, reason: '${f.name} is history, not a queue');
    }
  });

  testWidgets('opens on the pending queue', (tester) async {
    final log = <String>[];
    await tester.pumpWidget(_host(_fakeDio(statusLog: log)));
    await _settle(tester);

    expect(log, contains('PENDING'));
    expect(find.text('Khach Cho Giao'), findsOneWidget);
    await _teardown(tester);
  });

  testWidgets('tapping «Đã giao» refetches completed orders', (tester) async {
    final log = <String>[];
    await tester.pumpWidget(_host(_fakeDio(statusLog: log)));
    await _settle(tester);

    await tester.tap(find.text('Đã giao'));
    await _settle(tester);

    expect(log.last, 'COMPLETED');
    expect(find.text('Khach Da Giao'), findsOneWidget);
    // The pending order must be gone — this is a different listing, not a merge.
    expect(find.text('Khach Cho Giao'), findsNothing);
    await _teardown(tester);
  });

  testWidgets('tapping «Bị hủy» and «Tất cả» hit their own filters',
      (tester) async {
    final log = <String>[];
    await tester.pumpWidget(_host(_fakeDio(statusLog: log)));
    await _settle(tester);

    await tester.tap(find.text('Bị hủy'));
    await _settle(tester);
    expect(log.last, 'CANCELLED');
    expect(find.text('Khach Bi Huy'), findsOneWidget);

    await tester.tap(find.text('Tất cả'));
    await _settle(tester);
    expect(log.last, 'ALL');
    await _teardown(tester);
  });

  testWidgets('history rows carry a status badge', (tester) async {
    final log = <String>[];
    await tester.pumpWidget(_host(_fakeDio(statusLog: log)));
    await _settle(tester);

    await tester.tap(find.text('Đã giao'));
    await _settle(tester);

    expect(find.text('Hoàn tất'), findsOneWidget);
    await _teardown(tester);
  });
}
