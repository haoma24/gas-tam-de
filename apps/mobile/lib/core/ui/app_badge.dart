import 'package:flutter/material.dart';

import 'app_tokens.dart';

enum AppBadgeTone { neutral, accent, success, warning, danger, ink }

/// A small pill label.
///
/// Replaces `_ActiveChip`, `_LowStockChip`, `_SttBadge`, `_HeroChip` and the
/// six hardcoded colours that used to live inside `wait_time_badge.dart`.
class AppBadge extends StatelessWidget {
  const AppBadge(
    this.label, {
    super.key,
    this.tone = AppBadgeTone.neutral,
    this.icon,
  });

  final String label;
  final AppBadgeTone tone;
  final IconData? icon;

  @override
  Widget build(BuildContext context) {
    final p = context.palette;
    final (Color fg, Color bg, Color border) = switch (tone) {
      AppBadgeTone.neutral => (p.inkMuted, p.surfaceSubtle, p.border),
      AppBadgeTone.accent => (
          p.accent,
          p.accent.withValues(alpha: 0.10),
          p.accent.withValues(alpha: 0.30),
        ),
      AppBadgeTone.success => (
          p.success,
          p.success.withValues(alpha: 0.10),
          p.success.withValues(alpha: 0.30),
        ),
      AppBadgeTone.warning => (
          p.warning,
          p.warning.withValues(alpha: 0.10),
          p.warning.withValues(alpha: 0.30),
        ),
      AppBadgeTone.danger => (
          p.danger,
          p.danger.withValues(alpha: 0.10),
          p.danger.withValues(alpha: 0.30),
        ),
      AppBadgeTone.ink => (p.onInk, p.ink, p.ink),
    };

    return Container(
      padding: const EdgeInsets.symmetric(
        horizontal: AppSpacing.sm,
        vertical: 3,
      ),
      decoration: BoxDecoration(
        color: bg,
        borderRadius: AppRadius.full,
        border: Border.all(color: border),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          if (icon != null) ...[
            Icon(icon, size: 12, color: fg),
            const HGap(AppSpacing.xs),
          ],
          Text(
            label,
            style: context.text.labelSmall?.copyWith(
              color: fg,
              fontWeight: FontWeight.w600,
            ),
          ),
        ],
      ),
    );
  }
}

/// Square queue-number badge for the FIFO order desk.
class AppNumberBadge extends StatelessWidget {
  const AppNumberBadge(this.value, {super.key});

  final int value;

  @override
  Widget build(BuildContext context) {
    final p = context.palette;
    return Container(
      width: 32,
      height: 32,
      alignment: Alignment.center,
      decoration: BoxDecoration(
        color: p.surfaceSubtle,
        borderRadius: AppRadius.sm,
        border: Border.all(color: p.border),
      ),
      child: Text(
        '$value',
        style: context.text.labelLarge?.copyWith(
          color: p.ink,
          fontFeatures: kTabularFigures,
        ),
      ),
    );
  }
}
