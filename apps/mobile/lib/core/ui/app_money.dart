import 'package:flutter/material.dart';

import '../format.dart';
import 'app_tokens.dart';

export '../format.dart' show formatVnd;

/// A money amount rendered with tabular figures so columns line up.
class MoneyText extends StatelessWidget {
  const MoneyText(
    this.amount, {
    super.key,
    this.emphasis = MoneyEmphasis.normal,
    this.color,
  });

  final int amount;
  final MoneyEmphasis emphasis;
  final Color? color;

  @override
  Widget build(BuildContext context) {
    final p = context.palette;
    final base = switch (emphasis) {
      MoneyEmphasis.total => context.text.titleMedium,
      MoneyEmphasis.normal => context.text.bodyLarge,
      MoneyEmphasis.muted => context.text.bodyMedium,
    };
    return Text(
      formatVnd(amount),
      style: base?.copyWith(
        color: color ?? (emphasis == MoneyEmphasis.muted ? p.inkMuted : p.ink),
        fontFeatures: kTabularFigures,
        fontWeight: emphasis == MoneyEmphasis.normal ? FontWeight.w500 : null,
      ),
    );
  }
}

enum MoneyEmphasis { normal, total, muted }

/// One `label ........ value` line in a price breakdown.
///
/// Replaces the two private `_MoneyRow` copies that lived in the order review
/// and admin order detail screens.
class MoneyRow extends StatelessWidget {
  const MoneyRow({
    super.key,
    required this.label,
    this.amount,
    this.valueText,
    this.emphasis = MoneyEmphasis.normal,
    this.valueColor,
  }) : assert(
          amount != null || valueText != null,
          'MoneyRow needs either an amount or a valueText',
        );

  final String label;

  /// VND amount; formatted with [formatVnd]. Ignored when [valueText] is set.
  final int? amount;

  /// Free-form value (e.g. "3,2 km") for rows that are not money.
  final String? valueText;

  final MoneyEmphasis emphasis;
  final Color? valueColor;

  @override
  Widget build(BuildContext context) {
    final p = context.palette;
    final isTotal = emphasis == MoneyEmphasis.total;

    return Padding(
      padding: const EdgeInsets.symmetric(vertical: AppSpacing.sm),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Expanded(
            child: Text(
              label,
              style:
                  (isTotal ? context.text.titleSmall : context.text.bodyLarge)
                      ?.copyWith(color: isTotal ? p.ink : p.inkMuted),
            ),
          ),
          const HGap(AppSpacing.md),
          if (valueText != null)
            Text(
              valueText!,
              style:
                  (isTotal ? context.text.titleMedium : context.text.bodyLarge)
                      ?.copyWith(
                color: valueColor ?? p.ink,
                fontFeatures: kTabularFigures,
                fontWeight: isTotal ? null : FontWeight.w500,
              ),
            )
          else
            MoneyText(amount!, emphasis: emphasis, color: valueColor),
        ],
      ),
    );
  }
}
