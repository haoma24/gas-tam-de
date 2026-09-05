import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../core/ui/ui.dart';

/// Guest landing. One job: sign in.
///
/// PRD §3.1 is explicit that a signed-out visitor sees a single «Đăng nhập»
/// action — no order CTA, no admin entry point (admin arrives via the
/// `/#/admin/login` deep link).
class WelcomePage extends StatelessWidget {
  const WelcomePage({super.key});

  @override
  Widget build(BuildContext context) {
    final p = context.palette;

    return Scaffold(
      backgroundColor: p.bg,
      body: SafeArea(
        child: Center(
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 400),
            child: Padding(
              padding: const EdgeInsets.all(AppSpacing.xl),
              child: Column(
                mainAxisAlignment: MainAxisAlignment.center,
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Container(
                    width: 44,
                    height: 44,
                    alignment: Alignment.center,
                    decoration: BoxDecoration(
                      color: p.ink,
                      borderRadius: AppRadius.md,
                    ),
                    child: Icon(
                      Icons.local_fire_department_rounded,
                      color: p.onInk,
                      size: 24,
                    ),
                  ),
                  const VGap(AppSpacing.xl),
                  Text('Gas Tâm Đệ', style: context.text.headlineLarge),
                  const VGap(AppSpacing.sm),
                  Text(
                    'Đặt giao gas tận nơi. Biết trước phí giao và tổng tiền '
                    'trước khi đặt.',
                    style: context.text.bodyLarge?.copyWith(color: p.inkMuted),
                  ),
                  const VGap(AppSpacing.xxl),
                  AppButton.primary(
                    label: 'Đăng nhập',
                    expand: true,
                    onPressed: () => context.go('/auth/login'),
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}
