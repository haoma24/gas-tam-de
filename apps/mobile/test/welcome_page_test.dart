import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gas_tam_de/features/home/welcome_page.dart';

void main() {
  testWidgets('guest welcome offers exactly one action: Đăng nhập', (
    tester,
  ) async {
    await tester.pumpWidget(const MaterialApp(home: WelcomePage()));

    expect(find.text('Gas Tâm Đệ'), findsOneWidget);
    expect(find.widgetWithText(FilledButton, 'Đăng nhập'), findsOneWidget);

    // PRD §3.1: a signed-out visitor gets one button — no order CTA, no admin
    // entry point (admin arrives via the /#/admin/login deep link).
    expect(find.byType(FilledButton), findsOneWidget);
    expect(find.byType(OutlinedButton), findsNothing);
    expect(find.byType(TextButton), findsNothing);
    expect(find.text('Dành cho cửa hàng'), findsNothing);
    expect(find.textContaining('Đơn của tôi'), findsNothing);
  });
}
