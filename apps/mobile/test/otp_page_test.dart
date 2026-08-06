import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gas_tam_de/features/auth/_auth_widgets.dart';
import 'package:gas_tam_de/features/auth/auth_api.dart';
import 'package:gas_tam_de/features/auth/auth_models.dart';
import 'package:gas_tam_de/features/auth/auth_session.dart';
import 'package:gas_tam_de/features/auth/otp_page.dart';
import 'package:shared_preferences/shared_preferences.dart';

class _FakeAuthApi extends AuthApi {
  _FakeAuthApi() : super(Dio());

  int verifyCalls = 0;
  String? lastCode;

  @override
  Future<OtpVerifyResult> verifyOtp({
    required String phone,
    required String code,
  }) async {
    verifyCalls += 1;
    lastCode = code;
    return const AuthTokenResult(
      accessToken: 'access',
      refreshToken: 'refresh',
      tokenType: 'Bearer',
      expiresIn: 900,
      user: AuthUser(id: 'u1', role: 'customer'),
    );
  }
}

const _args = OtpNavArgs(
  phone: '0901234567',
  phoneMasked: '090****567',
  resendAfterSec: 60,
  expiresInSec: 300,
);

Future<void> _pumpOtpPage(
  WidgetTester tester, {
  required AuthApi api,
  VoidCallback? onVerified,
}) async {
  SharedPreferences.setMockInitialValues({});
  final prefs = await SharedPreferences.getInstance();
  await tester.pumpWidget(
    ProviderScope(
      overrides: [
        sharedPreferencesProvider.overrideWithValue(prefs),
        authApiProvider.overrideWithValue(api),
      ],
      child: MaterialApp(
        home: OtpPage(args: _args, onVerified: onVerified ?? () {}),
      ),
    ),
  );
  await tester.pump();
}

void main() {
  testWidgets('OTP field is tappable and sized (keyboard can open)',
      (tester) async {
    await _pumpOtpPage(tester, api: _FakeAuthApi());

    // Regression: the field used to live in a SizedBox(height: 0), which never
    // gets a browser keyboard on mobile web.
    final field = find.byType(TextField);
    expect(field, findsOneWidget);
    final size = tester.getSize(field);
    expect(size.height, kOtpBoxHeight);
    expect(size.width, greaterThan(100));

    // A tap anywhere on the digit boxes must reach the field.
    final boxes = tester.getCenter(find.byType(OtpBoxRow));
    await tester.tapAt(boxes);
    await tester.pump();
    expect(
      tester.widget<TextField>(field).focusNode?.hasFocus,
      isTrue,
    );
  });

  testWidgets('entering 6 digits verifies automatically', (tester) async {
    final api = _FakeAuthApi();
    var verified = 0;
    await _pumpOtpPage(tester, api: api, onVerified: () => verified += 1);

    await tester.enterText(find.byType(TextField), '123456');
    await tester.pumpAndSettle();

    expect(api.verifyCalls, 1);
    expect(api.lastCode, '123456');
    expect(verified, 1);
  });

  testWidgets('layout survives an open keyboard', (tester) async {
    tester.view.physicalSize = const Size(400, 700);
    tester.view.devicePixelRatio = 1.0;
    addTearDown(tester.view.reset);

    await _pumpOtpPage(tester, api: _FakeAuthApi());

    // Keyboard takes most of the viewport: the body must scroll, not overflow.
    tester.view.viewInsets = const FakeViewPadding(bottom: 420);
    await tester.pumpAndSettle();

    expect(tester.takeException(), isNull);
    expect(find.byType(AuthScrollBody), findsOneWidget);

    // Scrollable body: the field can be brought above the keyboard.
    final scrollable = find.byType(Scrollable).first;
    await tester.drag(scrollable, const Offset(0, -200));
    await tester.pumpAndSettle();
    final rect = tester.getRect(find.byType(TextField));
    expect(rect.bottom, lessThan(700 - 420));
  });
}
