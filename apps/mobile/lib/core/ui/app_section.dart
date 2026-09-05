import 'package:flutter/material.dart';

import 'app_tokens.dart';

/// A titled hairline card.
///
/// Replaces `_SectionCard` (order review), `_ProfileCard` (profile) and
/// `_MessageCard` (address) — three names for the same shape.
class AppSection extends StatelessWidget {
  const AppSection({
    super.key,
    this.title,
    this.trailing,
    this.icon,
    required this.children,
    this.padding = const EdgeInsets.all(AppSpacing.lg),
    this.tone = AppSectionTone.surface,
  });

  final String? title;

  /// Right-aligned widget on the title row (a spinner, an "Đổi" text button).
  final Widget? trailing;

  final IconData? icon;
  final List<Widget> children;
  final EdgeInsets padding;
  final AppSectionTone tone;

  @override
  Widget build(BuildContext context) {
    final p = context.palette;
    final borderColor = switch (tone) {
      AppSectionTone.surface => p.border,
      AppSectionTone.selected => p.primary,
      AppSectionTone.danger => p.danger.withValues(alpha: 0.4),
    };

    return Container(
      width: double.infinity,
      decoration: BoxDecoration(
        color: p.surface,
        borderRadius: AppRadius.md,
        border: Border.all(color: borderColor),
      ),
      padding: padding,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          if (title != null) ...[
            Row(
              children: [
                if (icon != null) ...[
                  // The header icon is the section's label, not a control:
                  // secondary keeps it findable without pulling the eye off
                  // the primary-coloured action inside the card.
                  Icon(icon, size: 18, color: p.secondary),
                  const HGap(AppSpacing.sm),
                ],
                Expanded(child: Text(title!, style: context.text.titleSmall)),
                if (trailing != null) trailing!,
              ],
            ),
            const VGap(AppSpacing.md),
          ],
          ...children,
        ],
      ),
    );
  }
}

enum AppSectionTone { surface, selected, danger }

/// Standalone section heading for content that is not inside a card.
class AppSectionTitle extends StatelessWidget {
  const AppSectionTitle(this.title, {super.key, this.trailing});

  final String title;
  final Widget? trailing;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.only(bottom: AppSpacing.md),
      child: Row(
        children: [
          Expanded(child: Text(title, style: context.text.titleMedium)),
          if (trailing != null) trailing!,
        ],
      ),
    );
  }
}
