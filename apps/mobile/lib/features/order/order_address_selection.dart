import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'geo_models.dart';

/// In-memory selected delivery address for the order flow (T3.1.3).
/// Place order (US-3.3) reads this + [orderGeoCheckProvider].
class OrderAddressNotifier extends StateNotifier<SelectedAddress?> {
  OrderAddressNotifier() : super(null);

  void select(SelectedAddress address) {
    state = address;
  }

  void clear() {
    state = null;
  }
}

final orderAddressProvider =
    StateNotifierProvider<OrderAddressNotifier, SelectedAddress?>((ref) {
  return OrderAddressNotifier();
});

/// Last successful geo check for the selected pin (T3.2.3). Cleared on new select.
class OrderGeoCheckNotifier extends StateNotifier<GeoCheckResult?> {
  OrderGeoCheckNotifier() : super(null);

  void set(GeoCheckResult? result) {
    state = result;
  }

  void clear() {
    state = null;
  }
}

final orderGeoCheckProvider =
    StateNotifierProvider<OrderGeoCheckNotifier, GeoCheckResult?>((ref) {
  return OrderGeoCheckNotifier();
});
