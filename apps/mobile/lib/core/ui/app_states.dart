import 'package:flutter/material.dart';

import 'app_tokens.dart';

/// Full-page loading state. Replaces the ~15 hand-rolled
/// `Center(child: CircularProgressIndicator())` copies.
class AppLoading extends StatelessWidget {
  const AppLoading({super.key, this.message});

  final String? message;

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          const SizedBox(
            width: 22,
            height: 22,
            child: CircularProgressIndicator(strokeWidth: 2),
          ),
          if (message != null) ...[
            const VGap(AppSpacing.md),
            Text(
              message!,
              style: context.text.bodyMedium?.copyWith(
                color: context.palette.inkMuted,
              ),
            ),
          ],
        ],
      ),
    );
  }
}

/// Small spinner for inside a button or a section header.
class AppInlineSpinner extends StatelessWidget {
  const AppInlineSpinner({super.key, this.size = 18, this.color});

  final double size;
  final Color? color;

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      width: size,
      height: size,
      child: CircularProgressIndicator(strokeWidth: 2, color: color),
    );
  }
}

/// Page-level error with a retry affordance. Replaces the ~10 near-identical
/// error blocks that each rendered a slightly different markup.
class AppErrorView extends StatelessWidget {
  const AppErrorView({
    super.key,
    required this.message,
    this.onRetry,
    this.retryLabel = 'Thử lại',
    this.icon = Icons.cloud_off_rounded,
  });

  final String message;
  final VoidCallback? onRetry;
  final String retryLabel;
  final IconData icon;

  @override
  Widget build(BuildContext context) {
    final p = context.palette;
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(AppSpacing.xl),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(icon, size: 32, color: p.inkFaint),
            const VGap(AppSpacing.md),
            Text(
              message,
              textAlign: TextAlign.center,
              style: context.text.bodyLarge?.copyWith(color: p.inkMuted),
            ),
            if (onRetry != null) ...[
              const VGap(AppSpacing.lg),
              OutlinedButton(onPressed: onRetry, child: Text(retryLabel)),
            ],
          ],
        ),
      ),
    );
  }
}

/// Inline error banner for a form or a section (does not take the whole page).
class AppErrorBanner extends StatelessWidget {
  const AppErrorBanner({super.key, required this.message});

  final String message;

  @override
  Widget build(BuildContext context) {
    final p = context.palette;
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.symmetric(
        horizontal: AppSpacing.md,
        vertical: AppSpacing.md,
      ),
      decoration: BoxDecoration(
        color: p.surfaceSubtle,
        borderRadius: AppRadius.sm,
        border: Border.all(color: p.danger.withValues(alpha: 0.35)),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Icon(Icons.error_outline_rounded, size: 18, color: p.danger),
          const HGap(AppSpacing.sm),
          Expanded(
            child: Text(
              message,
              style: context.text.bodyMedium?.copyWith(color: p.danger),
            ),
          ),
        ],
      ),
    );
  }
}

/// Empty state. Replaces the ~8 bespoke "nothing here yet" blocks.
class AppEmpty extends StatelessWidget {
  const AppEmpty({
    super.key,
    required this.title,
    this.body,
    this.icon,
    this.action,
  });

  final String title;
  final String? body;
  final IconData? icon;
  final Widget? action;

  @override
  Widget build(BuildContext context) {
    final p = context.palette;
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(AppSpacing.xl),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            if (icon != null) ...[
              Icon(icon, size: 32, color: p.inkFaint),
              const VGap(AppSpacing.md),
            ],
            Text(
              title,
              textAlign: TextAlign.center,
              style: context.text.titleSmall,
            ),
            if (body != null) ...[
              const VGap(AppSpacing.xs),
              Text(
                body!,
                textAlign: TextAlign.center,
                style: context.text.bodyMedium?.copyWith(color: p.inkMuted),
              ),
            ],
            if (action != null) ...[const VGap(AppSpacing.lg), action!],
          ],
        ),
      ),
    );
  }
}
