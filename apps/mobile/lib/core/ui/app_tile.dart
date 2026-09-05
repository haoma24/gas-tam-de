import 'package:flutter/material.dart';

import 'app_tokens.dart';

/// Tappable row that navigates somewhere. Replaces `_AdminNavTile` and
/// `_ActionTile`.
class AppNavTile extends StatelessWidget {
  const AppNavTile({
    super.key,
    required this.title,
    this.subtitle,
    this.icon,
    this.trailing,
    this.onTap,
    this.destructive = false,
  });

  final String title;
  final String? subtitle;
  final IconData? icon;

  /// Defaults to a chevron. Pass a badge or a value to override.
  final Widget? trailing;

  final VoidCallback? onTap;
  final bool destructive;

  @override
  Widget build(BuildContext context) {
    final p = context.palette;
    final fg = destructive ? p.danger : p.ink;

    return Material(
      color: p.surface,
      borderRadius: AppRadius.md,
      child: InkWell(
        onTap: onTap,
        borderRadius: AppRadius.md,
        child: Ink(
          decoration: BoxDecoration(
            borderRadius: AppRadius.md,
            border: Border.all(color: p.border),
          ),
          padding: const EdgeInsets.symmetric(
            horizontal: AppSpacing.lg,
            vertical: AppSpacing.lg - 2,
          ),
          child: Row(
            children: [
              if (icon != null) ...[
                Icon(icon,
                    size: 20, color: destructive ? p.danger : p.inkMuted),
                const HGap(AppSpacing.md),
              ],
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      title,
                      style: context.text.bodyLarge?.copyWith(
                        color: fg,
                        fontWeight: FontWeight.w500,
                      ),
                    ),
                    if (subtitle != null) ...[
                      const VGap(2),
                      Text(
                        subtitle!,
                        style: context.text.bodySmall?.copyWith(
                          color: p.inkMuted,
                        ),
                      ),
                    ],
                  ],
                ),
              ),
              const HGap(AppSpacing.sm),
              trailing ??
                  Icon(Icons.chevron_right_rounded,
                      size: 20, color: p.inkFaint),
            ],
          ),
        ),
      ),
    );
  }
}

/// Read-only `label / value` row. Replaces `_DetailField` and `_InfoRow`.
class AppDataRow extends StatelessWidget {
  const AppDataRow({
    super.key,
    required this.label,
    required this.value,
    this.valueWidget,
    this.stacked = false,
  });

  final String label;
  final String value;

  /// Rendered instead of [value] when set (a badge, a link).
  final Widget? valueWidget;

  /// Puts the value under the label — better for long addresses.
  final bool stacked;

  @override
  Widget build(BuildContext context) {
    final p = context.palette;
    final labelWidget = Text(
      label,
      style: context.text.bodyMedium?.copyWith(color: p.inkMuted),
    );
    final valueChild = valueWidget ??
        Text(
          value,
          style: context.text.bodyLarge?.copyWith(fontWeight: FontWeight.w500),
        );

    return Padding(
      padding: const EdgeInsets.symmetric(vertical: AppSpacing.sm),
      child: stacked
          ? Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [labelWidget, const VGap(AppSpacing.xs), valueChild],
            )
          : Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                SizedBox(width: 120, child: labelWidget),
                const HGap(AppSpacing.md),
                Expanded(child: valueChild),
              ],
            ),
    );
  }
}
