import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/app_theme.dart';
import '_auth_widgets.dart';
import 'auth_session.dart';
import 'google_auth.dart';
import 'google_sign_in_button.dart';

class GoogleLoginPage extends ConsumerWidget {
  const GoogleLoginPage({
    super.key,
    required this.onBack,
    required this.onLoggedIn,
  });

  final VoidCallback onBack;
  final VoidCallback onLoggedIn;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final google = ref.watch(googleAuthProvider);
    ref.listen(authSessionProvider, (previous, next) {
      if (previous == null && next != null) onLoggedIn();
    });

    return AnnotatedRegion<SystemUiOverlayStyle>(
      value: SystemUiOverlayStyle.light,
      child: Scaffold(
        backgroundColor: AppColors.obsidian,
        body: Stack(
          children: [
            const Positioned.fill(
              child: CustomPaint(painter: FlameAmbientPainter()),
            ),
            SafeArea(
              child: AuthScrollBody(
                top: Column(
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: [
                    Padding(
                      padding: const EdgeInsets.fromLTRB(8, 8, 16, 0),
                      child: Row(
                        children: [
                          IconButton(
                            icon: const Icon(
                              Icons.arrow_back_ios_new_rounded,
                              color: AppColors.onDark,
                              size: 20,
                            ),
                            onPressed: google.busy ? null : onBack,
                          ),
                          const Spacer(),
                        ],
                      ),
                    ),
                    const SizedBox(height: 40),
                    Padding(
                      padding: const EdgeInsets.symmetric(horizontal: 28),
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(
                            'Đăng nhập\nvới Google',
                            style: Theme.of(context)
                                .textTheme
                                .displaySmall
                                ?.copyWith(
                                  color: AppColors.onDark,
                                  fontWeight: FontWeight.w900,
                                  letterSpacing: -1.5,
                                  height: 1,
                                ),
                          ),
                          const SizedBox(height: 14),
                          Text(
                            'Phiên đăng nhập được ghi nhớ trên thiết bị cho đến khi bạn đăng xuất.',
                            style: Theme.of(context)
                                .textTheme
                                .bodyLarge
                                ?.copyWith(
                                  color: AppColors.onDark.withValues(alpha: .6),
                                  height: 1.5,
                                ),
                          ),
                        ],
                      ),
                    ),
                  ],
                ),
                bottom: Padding(
                  padding: const EdgeInsets.fromLTRB(16, 24, 16, 0),
                  child: AuthCard(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.stretch,
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        buildGoogleSignInButton(
                          enabled: google.ready && !google.busy,
                          onPressed: () => ref
                              .read(googleAuthProvider.notifier)
                              .authenticate(),
                        ),
                        if (google.busy) ...[
                          const SizedBox(height: 16),
                          const Center(
                            child: CircularProgressIndicator(strokeWidth: 2),
                          ),
                        ],
                        if (google.error != null) ...[
                          const SizedBox(height: 14),
                          AuthErrorText(google.error!),
                        ],
                        const SizedBox(height: 14),
                        Text(
                          'Google chỉ được dùng để xác minh danh tính. Gas Tam Đệ không nhận mật khẩu Google của bạn.',
                          textAlign: TextAlign.center,
                          style: Theme.of(context)
                              .textTheme
                              .bodySmall
                              ?.copyWith(
                                color: AppColors.onDark.withValues(alpha: .5),
                                height: 1.4,
                              ),
                        ),
                      ],
                    ),
                  ),
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}
