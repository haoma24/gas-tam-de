import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'auth_api.dart';
import 'auth_models.dart';
import 'auth_session_store.dart';

/// In-memory + persisted session after OTP verify or admin login.
class AuthSession {
  const AuthSession({
    required this.accessToken,
    required this.refreshToken,
    required this.tokenType,
    required this.expiresIn,
    required this.user,
    this.accessExpiresAt,
  });

  final String accessToken;
  final String refreshToken;
  final String tokenType;
  final int expiresIn;
  final AuthUser user;

  /// Absolute expiry of [accessToken] (UTC). Used on restore to decide refresh.
  final DateTime? accessExpiresAt;

  bool get isAdmin => user.role == 'admin';
  bool get isCustomer => user.role == 'customer';

  bool get accessExpired {
    final at = accessExpiresAt;
    if (at == null) return false;
    // Refresh a bit early to avoid edge 401s.
    return DateTime.now()
        .toUtc()
        .isAfter(at.subtract(const Duration(seconds: 30)));
  }

  factory AuthSession.fromTokens(AuthTokenResult result) {
    final now = DateTime.now().toUtc();
    return AuthSession(
      accessToken: result.accessToken,
      refreshToken: result.refreshToken,
      tokenType: result.tokenType,
      expiresIn: result.expiresIn,
      user: result.user,
      accessExpiresAt: now.add(Duration(seconds: result.expiresIn)),
    );
  }

  /// Kept for OTP call sites (T1.1.4).
  factory AuthSession.fromVerify(OtpVerifyResult result) =>
      AuthSession.fromTokens(result);

  factory AuthSession.fromAdminLogin(AdminLoginResult result) =>
      AuthSession.fromTokens(result);

  factory AuthSession.fromJson(Map<String, dynamic> json) {
    final userJson = json['user'];
    final expiresRaw = json['access_expires_at'] as String?;
    return AuthSession(
      accessToken: json['access_token'] as String? ?? '',
      refreshToken: json['refresh_token'] as String? ?? '',
      tokenType: json['token_type'] as String? ?? 'Bearer',
      expiresIn: (json['expires_in'] as num?)?.toInt() ?? 900,
      user: userJson is Map<String, dynamic>
          ? AuthUser.fromJson(userJson)
          : const AuthUser(id: '', role: 'customer'),
      accessExpiresAt:
          expiresRaw != null ? DateTime.tryParse(expiresRaw)?.toUtc() : null,
    );
  }

  Map<String, dynamic> toJson() => {
        'access_token': accessToken,
        'refresh_token': refreshToken,
        'token_type': tokenType,
        'expires_in': expiresIn,
        'user': user.toJson(),
        if (accessExpiresAt != null)
          'access_expires_at': accessExpiresAt!.toUtc().toIso8601String(),
      };
}

final sharedPreferencesProvider = Provider<SharedPreferences>((ref) {
  throw UnimplementedError('override sharedPreferencesProvider in main()');
});

final authSessionStoreProvider = Provider<AuthSessionStore>((ref) {
  return AuthSessionStore(ref.watch(sharedPreferencesProvider));
});

class AuthSessionNotifier extends StateNotifier<AuthSession?> {
  AuthSessionNotifier(this._ref) : super(null);

  final Ref _ref;

  AuthSessionStore get _store => _ref.read(authSessionStoreProvider);

  /// How long startup waits for a token refresh before showing the UI anyway.
  ///
  /// Dio allows 15s connect + 15s receive, so an unreachable API used to hold
  /// the splash spinner for ~30s before the login screen appeared.
  static const bootstrapRefreshTimeout = Duration(seconds: 4);

  /// Load persisted session and refresh access token when expired / near expiry.
  Future<void> bootstrap() async {
    final saved = await _store.load();
    if (saved == null) {
      state = null;
      return;
    }
    if (saved.accessToken.isEmpty || saved.refreshToken.isEmpty) {
      await clear();
      return;
    }

    // Publish the stored session first: routing only needs the role, so the UI
    // renders immediately instead of waiting on the network.
    state = saved;
    if (!saved.accessExpired) return;

    // `timeout` does not cancel the request — a slow refresh keeps running and
    // still updates the session when it lands.
    await _refresh(saved).timeout(bootstrapRefreshTimeout, onTimeout: () {});
  }

  Future<void> _refresh(AuthSession saved) async {
    try {
      final result =
          await _ref.read(authApiProvider).refresh(saved.refreshToken);
      await setSession(AuthSession.fromTokens(result));
    } on AuthApiException catch (e) {
      if (e.code == 'INVALID_TOKEN' || e.statusCode == 401) {
        await clear();
      }
      // Network / transient — keep saved tokens so offline UI still works.
    } catch (_) {
      // Keep saved tokens.
    }
  }

  Future<void> setSession(AuthSession session) async {
    state = session;
    await _store.save(session);
  }

  Future<void> clear() async {
    state = null;
    await _store.clear();
  }
}

final authSessionProvider =
    StateNotifierProvider<AuthSessionNotifier, AuthSession?>((ref) {
  return AuthSessionNotifier(ref);
});

/// Completes after first [AuthSessionNotifier.bootstrap].
final authBootstrapProvider = FutureProvider<void>((ref) async {
  await ref.read(authSessionProvider.notifier).bootstrap();
});
