import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gas_tam_de/features/home/home_page.dart';

void main() {
  testWidgets('guest home shows brand CTA without admin button', (tester) async {
    await tester.pumpWidget(
      MaterialApp(
        home: HomePage(
          onStartOrder: () {},
          onMyOrders: () {},
        ),
      ),
    );

    expect(find.text('Gas Tam Đệ'), findsOneWidget);
    expect(find.text('Đặt giao gas'), findsOneWidget);
    expect(find.text('Dành cho cửa hàng'), findsNothing);
  });
}
