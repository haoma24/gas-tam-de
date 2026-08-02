import 'package:flutter/material.dart';

import 'desk_settings_models.dart';

/// Small wait-time badge for Order Desk rows.
class WaitTimeBadge extends StatelessWidget {
  const WaitTimeBadge({
    super.key,
    required this.createdAt,
    required this.settings,
    this.now,
  });

  final String createdAt;
  final DeskSettings settings;
  final DateTime? now;

  @override
  Widget build(BuildContext context) {
    final created = DateTime.tryParse(createdAt)?.toLocal();
    if (created == null) {
      return const SizedBox.shrink();
    }
    final waited = (now ?? DateTime.now()).difference(created);
    final urgency = waitUrgencyFor(waited, settings);
    final label = _fmtWait(waited);
    final colors = _colors(urgency);

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
      decoration: BoxDecoration(
        color: colors.$1,
        borderRadius: BorderRadius.circular(999),
      ),
      child: Text(
        label,
        style: Theme.of(context).textTheme.labelMedium?.copyWith(
              color: colors.$2,
              fontWeight: FontWeight.w800,
            ),
      ),
    );
  }

  static (Color, Color) _colors(WaitUrgency u) {
    switch (u) {
      case WaitUrgency.blue:
        return (const Color(0xFFDBEAFE), const Color(0xFF1D4ED8));
      case WaitUrgency.orange:
        return (const Color(0xFFFFEDD5), const Color(0xFFC2410C));
      case WaitUrgency.red:
        return (const Color(0xFFFEE2E2), const Color(0xFFB91C1C));
    }
  }

  static String _fmtWait(Duration d) {
    if (d.isNegative) return '0p';
    final m = d.inMinutes;
    if (m < 60) return '${m}p';
    final h = d.inHours;
    final rem = m % 60;
    return rem == 0 ? '${h}g' : '${h}g${rem}p';
  }
}
