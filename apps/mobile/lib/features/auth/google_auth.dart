import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:google_sign_in/google_sign_in.dart';

import 'auth_api.dart';
import 'auth_models.dart';
import 'auth_session.dart';

const _googleWebClientId = String.fromEnvironment('GOOGLE_WEB_CLIENT_ID');
const _googleIosClientId = String.fromEnvironment('GOOGLE_IOS_CLIENT_ID');

class GoogleAuthState {
  const GoogleAuthState({
    this.ready = false,
    this.busy = false,
    this.error,
  });

  final bool ready;
  final bool busy;
  final String? error;

  GoogleAuthState copyWith({bool? ready, bool? busy, String? error}) =>
      GoogleAuthState(
        ready: ready ?? this.ready,
        busy: busy ?? this.busy,
        error: error,
      );
}

class GoogleAuthNotifier extends StateNotifier<GoogleAuthState> {
  GoogleAuthNotifier(this._ref) : super(const GoogleAuthState()) {
    unawaited(_initialize());
  }

  final Ref _ref;
  final GoogleSignIn _google = GoogleSignIn.instance;
  StreamSubscription<GoogleSignInAuthenticationEvent>? _subscription;
  String? _exchangeInProgress;
  String? _lastExchangedToken;

  Future<void> _initialize() async {
    if (_googleWebClientId.isEmpty) {
      state = const GoogleAuthState(
        error: 'Ứng dụng chưa cấu hình GOOGLE_WEB_CLIENT_ID.',
      );
      return;
    }
    try {
      await _google.initialize(
        clientId: kIsWeb
            ? _googleWebClientId
            : defaultTargetPlatform == TargetPlatform.iOS &&
                    _googleIosClientId.isNotEmpty
                ? _googleIosClientId
                : null,
        serverClientId: kIsWeb ? null : _googleWebClientId,
      );
      _subscription = _google.authenticationEvents.listen(
        (event) {
          if (event is GoogleSignInAuthenticationEventSignIn) {
            unawaited(_exchange(event.user));
          }
        },
        onError: (Object error) {
          state = state.copyWith(busy: false, error: _messageFor(error));
        },
      );
      state = const GoogleAuthState(ready: true);
    } catch (error) {
      state = GoogleAuthState(error: _messageFor(error));
    }
  }

  Future<void> authenticate() async {
    if (!state.ready || state.busy) return;
    if (!_google.supportsAuthenticate()) {
      state = state.copyWith(
        error: 'Hãy dùng nút Google để đăng nhập trên trình duyệt.',
      );
      return;
    }
    state = state.copyWith(busy: true);
    try {
      final account = await _google.authenticate();
      await _exchange(account);
    } catch (error) {
      state = state.copyWith(busy: false, error: _messageFor(error));
    }
  }

  Future<void> _exchange(GoogleSignInAccount account) async {
    final idToken = account.authentication.idToken;
    if (idToken == null || idToken.isEmpty) {
      state = state.copyWith(
        busy: false,
        error: 'Google không trả về mã định danh. Vui lòng thử lại.',
      );
      return;
    }
    if (_exchangeInProgress == idToken || _lastExchangedToken == idToken) return;
    _exchangeInProgress = idToken;
    state = state.copyWith(busy: true);
    try {
      final result = await _ref.read(authApiProvider).googleLogin(idToken);
      await _ref
          .read(authSessionProvider.notifier)
          .setSession(AuthSession.fromTokens(result));
      _lastExchangedToken = idToken;
      state = state.copyWith(busy: false);
    } on AuthApiException catch (error) {
      state = state.copyWith(busy: false, error: error.displayMessage);
    } catch (_) {
      state = state.copyWith(
        busy: false,
        error: 'Không thể đăng nhập với Google. Vui lòng thử lại.',
      );
    } finally {
      _exchangeInProgress = null;
    }
  }

  Future<void> logout() async {
    try {
      await _ref.read(authSessionProvider.notifier).logout();
    } finally {
      try {
        await _google.signOut();
      } catch (_) {
        // The app session is already cleared; provider cleanup is best-effort.
      }
      _lastExchangedToken = null;
      state = GoogleAuthState(ready: state.ready);
    }
  }

  String _messageFor(Object error) {
    if (error is GoogleSignInException &&
        error.code == GoogleSignInExceptionCode.canceled) {
      return 'Bạn đã hủy đăng nhập Google.';
    }
    return 'Không thể mở đăng nhập Google. Kiểm tra cấu hình OAuth.';
  }

  @override
  void dispose() {
    unawaited(_subscription?.cancel());
    super.dispose();
  }
}

final googleAuthProvider =
    StateNotifierProvider<GoogleAuthNotifier, GoogleAuthState>((ref) {
  return GoogleAuthNotifier(ref);
});
