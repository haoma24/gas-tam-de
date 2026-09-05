import 'package:flutter/material.dart';

import 'app_states.dart';
import 'app_tokens.dart';

enum AppButtonVariant {
  /// Ink block. The default primary action.
  primary,

  /// Hairline outline.
  secondary,

  /// No chrome.
  text,

  /// Destructive — outlined in danger, filled only on confirm dialogs.
  danger,

  /// Accent fill. Reserved for the order CTA; at most one per viewport.
  accent,
}

/// One button for the whole app.
///
/// Replaces `GradientCTAButton`, `_LoginButton`, `_OrderFAB` and `_FireButton`
/// — four separate hand-rolled `AnimationController` + `ScaleTransition` press
/// effects. Material's own ink response is the press feedback now.
class AppButton extends StatelessWidget {
  const AppButton({
    super.key,
    required this.label,
    this.onPressed,
    this.variant = AppButtonVariant.primary,
    this.icon,
    this.loading = false,
    this.expand = false,
  });

  const AppButton.primary({
    super.key,
    required this.label,
    this.onPressed,
    this.icon,
    this.loading = false,
    this.expand = false,
  }) : variant = AppButtonVariant.primary;

  const AppButton.secondary({
    super.key,
    required this.label,
    this.onPressed,
    this.icon,
    this.loading = false,
    this.expand = false,
  }) : variant = AppButtonVariant.secondary;

  const AppButton.text({
    super.key,
    required this.label,
    this.onPressed,
    this.icon,
    this.loading = false,
    this.expand = false,
  }) : variant = AppButtonVariant.text;

  const AppButton.danger({
    super.key,
    required this.label,
    this.onPressed,
    this.icon,
    this.loading = false,
    this.expand = false,
  }) : variant = AppButtonVariant.danger;

  const AppButton.accent({
    super.key,
    required this.label,
    this.onPressed,
    this.icon,
    this.loading = false,
    this.expand = false,
  }) : variant = AppButtonVariant.accent;

  final String label;
  final VoidCallback? onPressed;
  final AppButtonVariant variant;
  final IconData? icon;

  /// Swaps the leading icon for a spinner and blocks presses.
  final bool loading;

  /// Stretches to the full width of the parent.
  final bool expand;

  @override
  Widget build(BuildContext context) {
    final p = context.palette;
    final enabled = onPressed != null && !loading;
    final child = _Content(label: label, icon: icon, loading: loading);

    final Widget button = switch (variant) {
      AppButtonVariant.primary => FilledButton(
          onPressed: enabled ? onPressed : null,
          child: child,
        ),
      AppButtonVariant.accent => FilledButton(
          onPressed: enabled ? onPressed : null,
          style: FilledButton.styleFrom(
            backgroundColor: p.accent,
            foregroundColor: p.onAccent,
          ),
          child: child,
        ),
      AppButtonVariant.secondary => OutlinedButton(
          onPressed: enabled ? onPressed : null,
          child: child,
        ),
      AppButtonVariant.danger => OutlinedButton(
          onPressed: enabled ? onPressed : null,
          style: OutlinedButton.styleFrom(
            foregroundColor: p.danger,
            side: BorderSide(color: p.danger.withValues(alpha: 0.5)),
          ),
          child: child,
        ),
      AppButtonVariant.text => TextButton(
          onPressed: enabled ? onPressed : null,
          child: child,
        ),
    };

    if (!expand) return button;
    return SizedBox(width: double.infinity, child: button);
  }
}

class _Content extends StatelessWidget {
  const _Content({required this.label, this.icon, required this.loading});

  final String label;
  final IconData? icon;
  final bool loading;

  @override
  Widget build(BuildContext context) {
    if (!loading && icon == null) return Text(label);

    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        if (loading)
          AppInlineSpinner(
              size: 16, color: DefaultTextStyle.of(context).style.color)
        else
          Icon(icon, size: 18),
        const HGap(AppSpacing.sm),
        Flexible(child: Text(label, overflow: TextOverflow.ellipsis)),
      ],
    );
  }
}
