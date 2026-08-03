import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gas_tam_de/features/home/home_page.dart';

void main() {
  testWidgets('guest home shows only Đăng nhập', (tester) async {
    await tester.pumpWidget(
      MaterialApp(
        home: HomePage(onLogin: () {}),
      ),
    );

    // Brand text is split across newline — find by prefix.
    expect(find.textContaining('Gas'), findsWidgets);
    expect(find.text('Đăng nhập'), findsOneWidget);
    expect(find.text('Đặt giao gas'), findsNothing);
    expect(find.text('Dành cho cửa hàng'), findsNothing);
    expect(find.textContaining('Đơn của tôi'), findsNothing);
  });
}
