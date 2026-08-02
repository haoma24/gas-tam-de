import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'auth_models.dart';

/// In-memory session after OTP verify or admin login (no secure storage yet).
class AuthSession {
  const AuthSession({
    required this.accessToken,
    required this.refreshToken,
    required this.tokenType,
    required this.expiresIn,
    required this.user,
  });

  final String accessToken;
  final String refreshToken;
  final String tokenType;
  final int expiresIn;
  final AuthUser user;

  factory AuthSession.fromTokens(AuthTokenResult result) {
    return AuthSession(
      accessToken: result.accessToken,
      refreshToken: result.refreshToken,
      tokenType: result.tokenType,
      expiresIn: result.expiresIn,
      user: result.user,
    );
  }

  /// Kept for OTP call sites (T1.1.4).
  factory AuthSession.fromVerify(OtpVerifyResult result) =>
      AuthSession.fromTokens(result);

  factory AuthSession.fromAdminLogin(AdminLoginResult result) =>
      AuthSession.fromTokens(result);
}

final authSessionProvider = StateProvider<AuthSession?>((ref) => null);
