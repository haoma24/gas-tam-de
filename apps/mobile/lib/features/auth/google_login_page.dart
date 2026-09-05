import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/ui/ui.dart';
import 'google_auth.dart';
import 'google_sign_in_button.dart';

/// Customer sign-in. The router redirects away once a session exists, so this
/// page only has to start the Google flow and report its errors.
class GoogleLoginPage extends ConsumerWidget {
  const GoogleLoginPage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final google = ref.watch(googleAuthProvider);
    final p = context.palette;

    return AppScaffold(
      backFallback: '/welcome',
      padBody: false,
      body: AuthScrollBody(
        top: Padding(
          padding: const EdgeInsets.fromLTRB(
            AppSpacing.xl,
            AppSpacing.xxl,
            AppSpacing.xl,
            0,
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text('Đăng nhập với Google', style: context.text.headlineSmall),
              const VGap(AppSpacing.md),
              Text(
                'Phiên đăng nhập được ghi nhớ trên thiết bị cho đến khi bạn '
                'đăng xuất.',
                style: context.text.bodyLarge?.copyWith(color: p.inkMuted),
              ),
            ],
          ),
        ),
        bottom: Padding(
          padding: const EdgeInsets.all(AppSpacing.lg),
          child: AuthCard(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              mainAxisSize: MainAxisSize.min,
              children: [
                buildGoogleSignInButton(
                  enabled: google.ready && !google.busy,
                  onPressed: () =>
                      ref.read(googleAuthProvider.notifier).authenticate(),
                ),
                if (google.busy) ...[
                  const VGap(AppSpacing.lg),
                  const Center(child: AppInlineSpinner()),
                ],
                if (google.error != null) AuthErrorText(google.error!),
                const VGap(AppSpacing.lg),
                Text(
                  'Google chỉ được dùng để xác minh danh tính. Gas Tâm Đệ '
                  'không nhận mật khẩu Google của bạn.',
                  textAlign: TextAlign.center,
                  style: context.text.bodySmall?.copyWith(color: p.inkFaint),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
