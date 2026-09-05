import 'package:flutter_test/flutter_test.dart';
import 'package:gas_tam_de/core/phone_link.dart';
import 'package:gas_tam_de/features/order/order_models.dart';

/// The shop was shown `090***7020` for every order and could not phone anyone.
/// These pin the number that reaches the admin screens and the `tel:` link
/// behind it.

AdminOrder _order({
  String customerPhone = '',
  String phoneMasked = '090***7020',
  String status = OrderStatus.pending,
  int total = 450000,
  int amountPaid = 0,
  String paymentType = '',
}) {
  return AdminOrder.fromJson({
    'id': 'ord-1',
    'user_id': 'user-a',
    'customer_name': 'Chị Lan',
    'phone_masked': phoneMasked,
    if (customerPhone.isNotEmpty) 'customer_phone': customerPhone,
    'address_text': '1 Lê Lợi',
    'total': total,
    'subtotal': total,
    'status': status,
    'created_at': '2026-08-02T09:00:00Z',
    if (paymentType.isNotEmpty) 'payment_type': paymentType,
    if (amountPaid > 0) 'amount_paid': amountPaid,
    'items': const <Map<String, dynamic>>[],
  });
}

void main() {
  test('admin order prefers the full phone over the masked one', () {
    final withPhone = _order(customerPhone: '0909777020');
    expect(withPhone.displayPhone, '0909777020');
    expect(withPhone.dialablePhone, '0909777020');
  });

  test('falls back to masked, and offers no dialer for it', () {
    final masked = _order();
    expect(masked.displayPhone, '090***7020');
    // A masked number must never be dialled — it is not a real number.
    expect(masked.dialablePhone, isEmpty);
    expect(telUri('090***7020'), isNull);
  });

  test('shows a dash when there is no number at all', () {
    expect(_order(phoneMasked: '').displayPhone, '—');
  });

  test('tel: strips formatting and refuses non-numbers', () {
    expect(telDigits('0909 777 020'), '0909777020');
    expect(telDigits('+84 909 777 020'), '+84909777020');
    expect(telDigits('090***7020'), isEmpty);
    expect(telDigits('12345'), isEmpty, reason: 'too short to be a phone');
    expect(telDigits(''), isEmpty);

    expect(telUri('0909777020')?.toString(), 'tel:0909777020');
  });

  test('debt is derived from the settlement of a completed order', () {
    final partial = _order(
      status: OrderStatus.completed,
      paymentType: OrderPaymentType.partial,
      amountPaid: 200000,
    );
    expect(partial.debt, 250000);
    expect(partial.isPending, isFalse);

    final unpaid = _order(
      status: OrderStatus.completed,
      paymentType: OrderPaymentType.unpaid,
    );
    expect(unpaid.debt, 450000);

    final paid = _order(
      status: OrderStatus.completed,
      paymentType: OrderPaymentType.full,
      amountPaid: 450000,
    );
    expect(paid.debt, 0);

    // A pending order owes nothing yet — it has not been handed over.
    expect(_order().debt, 0);
  });

  test('line profit needs a purchase price to exist', () {
    final withCost = OrderItemView.fromJson({
      'unit_price': 450000,
      'qty': 2,
      'line_total': 900000,
      'unit_cost': 380000,
    });
    expect(withCost.unitCost, 380000);
    expect(withCost.lineProfit, 140000);

    // No cost on file: say "unknown" rather than claim the whole line is profit.
    final noCost = OrderItemView.fromJson({
      'unit_price': 450000,
      'qty': 2,
      'line_total': 900000,
    });
    expect(noCost.unitCost, 0);
    expect(noCost.lineProfit, isNull);
  });

  test('payment type has a Vietnamese label', () {
    expect(orderPaymentLabelVi(OrderPaymentType.full), 'Thu đủ');
    expect(orderPaymentLabelVi(OrderPaymentType.partial), 'Thu một phần');
    expect(orderPaymentLabelVi(OrderPaymentType.unpaid), 'Chưa thu');
    expect(orderPaymentLabelVi(''), '—');
  });
}
