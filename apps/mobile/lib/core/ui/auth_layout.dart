import 'dart:math' as math;

import 'package:flutter/material.dart';

import 'app_tokens.dart';

/// Keyboard-safe auth layout: [top] block pinned high, [bottom] block pinned
/// low, both scrollable together when the keyboard shrinks the viewport.
///
/// Moved out of `features/auth/_auth_widgets.dart` — it is layout, not auth.
class AuthScrollBody extends StatelessWidget {
  const AuthScrollBody({
    super.key,
    required this.top,
    required this.bottom,
    this.bottomPadding = AppSpacing.xxl,
  });

  final Widget top;
  final Widget bottom;
  final double bottomPadding;

  @override
  Widget build(BuildContext context) {
    return LayoutBuilder(
      builder: (context, constraints) {
        final minHeight = constraints.maxHeight.isFinite
            ? math.max(0.0, constraints.maxHeight - bottomPadding)
            : 0.0;
        return SingleChildScrollView(
          padding: EdgeInsets.only(bottom: bottomPadding),
          child: ConstrainedBox(
            constraints: BoxConstraints(minHeight: minHeight),
            // spaceBetween (not Spacer) — Expanded cannot resolve inside the
            // unbounded height of a scroll view.
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              mainAxisAlignment: MainAxisAlignment.spaceBetween,
              children: [top, bottom],
            ),
          ),
        );
      },
    );
  }
}

/// Hairline card holding the auth form itself.
class AuthCard extends StatelessWidget {
  const AuthCard({super.key, required this.child});

  final Widget child;

  @override
  Widget build(BuildContext context) {
    final p = context.palette;
    return Container(
      decoration: BoxDecoration(
        color: p.surface,
        borderRadius: AppRadius.md,
        border: Border.all(color: p.border),
      ),
      padding: const EdgeInsets.all(AppSpacing.xl),
      child: child,
    );
  }
}

/// Inline error line under an auth form.
class AuthErrorText extends StatelessWidget {
  const AuthErrorText(this.message, {super.key});

  final String? message;

  @override
  Widget build(BuildContext context) {
    final text = message?.trim();
    if (text == null || text.isEmpty) return const SizedBox.shrink();
    final p = context.palette;
    return Padding(
      padding: const EdgeInsets.only(top: AppSpacing.md),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Icon(Icons.error_outline_rounded, size: 16, color: p.danger),
          const HGap(AppSpacing.sm),
          Expanded(
            child: Text(
              text,
              style: context.text.bodyMedium?.copyWith(color: p.danger),
            ),
          ),
        ],
      ),
    );
  }
}
