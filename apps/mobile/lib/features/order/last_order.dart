import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../auth/auth_session.dart';
import 'order_api.dart';
import 'order_models.dart';

/// The customer's most recent order, condensed for the reorder card.
class LastOrderSummary {
  const LastOrderSummary({
    required this.items,
    required this.addressText,
    required this.total,
  });

  final List<OrderItemView> items;
  final String addressText;
  final int total;

  /// e.g. `Gas 12kg × 1 · Van điều áp × 1`
  String get summaryLine =>
      items.map((i) => '${i.productName} × ${i.qty}').join(' · ');
}

/// Newest order from `GET /v1/orders/me`, or null when the customer has none.
///
/// The endpoint already returns the line items, so a repeat order needs no new
/// API — it just needs to be surfaced.
final lastOrderProvider = FutureProvider<LastOrderSummary?>((ref) async {
  final session = ref.watch(authSessionProvider);
  if (session == null || !session.isCustomer) return null;

  try {
    final orders = await ref.read(orderApiProvider).listMyOrders();
    if (orders.isEmpty) return null;
    // The API sorts oldest-first for the admin desk; the customer wants newest.
    final newest = orders.reduce(
      (a, b) => a.createdAt.compareTo(b.createdAt) >= 0 ? a : b,
    );
    if (newest.items.isEmpty) return null;
    return LastOrderSummary(
      items: newest.items,
      addressText: newest.addressText,
      total: newest.total,
    );
  } catch (_) {
    // The reorder card is an accelerator, never a blocker.
    return null;
  }
});
