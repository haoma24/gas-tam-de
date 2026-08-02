import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../catalog/catalog_models.dart';

/// One line in the local order cart (qty ≥ 1).
class CartLine {
  const CartLine({
    required this.product,
    required this.quantity,
  });

  final Product product;
  final int quantity;

  int get lineTotal => product.salePrice * quantity;

  CartLine copyWith({Product? product, int? quantity}) {
    return CartLine(
      product: product ?? this.product,
      quantity: quantity ?? this.quantity,
    );
  }
}

/// In-memory cart for the customer order flow (product pick → later geo/place).
class OrderCart {
  const OrderCart([this._byId = const {}]);

  final Map<String, CartLine> _byId;

  List<CartLine> get lines {
    final list = _byId.values.toList();
    list.sort((a, b) => a.product.name.compareTo(b.product.name));
    return list;
  }

  int quantityOf(String productId) => _byId[productId]?.quantity ?? 0;

  int get totalQuantity =>
      _byId.values.fold(0, (sum, line) => sum + line.quantity);

  int get totalAmount =>
      _byId.values.fold(0, (sum, line) => sum + line.lineTotal);

  bool get isEmpty => _byId.isEmpty;

  bool get isNotEmpty => _byId.isNotEmpty;

  OrderCart setQuantity(Product product, int quantity) {
    final next = Map<String, CartLine>.from(_byId);
    if (quantity <= 0) {
      next.remove(product.id);
    } else {
      next[product.id] = CartLine(product: product, quantity: quantity);
    }
    return OrderCart(next);
  }

  OrderCart clear() => const OrderCart();
}

class OrderCartNotifier extends StateNotifier<OrderCart> {
  OrderCartNotifier() : super(const OrderCart());

  void setQuantity(Product product, int quantity) {
    state = state.setQuantity(product, quantity);
  }

  void increment(Product product) {
    state = state.setQuantity(product, state.quantityOf(product.id) + 1);
  }

  void decrement(Product product) {
    state = state.setQuantity(product, state.quantityOf(product.id) - 1);
  }

  void clear() {
    state = state.clear();
  }
}

final orderCartProvider =
    StateNotifierProvider<OrderCartNotifier, OrderCart>((ref) {
  return OrderCartNotifier();
});
