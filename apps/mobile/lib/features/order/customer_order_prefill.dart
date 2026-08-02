import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../auth/auth_session.dart';
import '../auth/me_api.dart';
import 'order_api.dart';
import 'order_models.dart';

/// Prefill for customer order flow: saved name + last delivery address.
class CustomerOrderPrefill {
  const CustomerOrderPrefill({
    this.fullName,
    this.lastAddress,
  });

  final String? fullName;
  final OrderDefaults? lastAddress;

  bool get hasName => fullName != null && fullName!.trim().isNotEmpty;
  bool get hasLastAddress =>
      lastAddress != null &&
      lastAddress!.hasDefaults &&
      (lastAddress!.addressText?.isNotEmpty ?? false) &&
      lastAddress!.lat != null &&
      lastAddress!.lng != null;
}

final customerOrderPrefillProvider =
    FutureProvider<CustomerOrderPrefill>((ref) async {
  final session = ref.watch(authSessionProvider);
  if (session == null || !session.isCustomer) {
    return const CustomerOrderPrefill();
  }

  String? name;
  OrderDefaults? defaults;

  try {
    final profile = await ref.read(meApiProvider).getMe();
    name = profile.fullName;
  } catch (_) {}

  try {
    defaults = await ref.read(orderApiProvider).getMyDefaults();
  } catch (_) {}

  if ((name == null || name.isEmpty) &&
      defaults != null &&
      defaults.hasDefaults &&
      (defaults.customerName?.isNotEmpty ?? false)) {
    name = defaults.customerName;
  }

  return CustomerOrderPrefill(
    fullName: name,
    lastAddress: defaults,
  );
});
