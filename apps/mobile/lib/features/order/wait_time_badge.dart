import 'package:flutter/material.dart';

import '../../core/ui/ui.dart';
import 'desk_settings_models.dart';

/// How long an order has been waiting, escalating neutral → warning → danger.
///
/// The three tones come from the palette now; this widget used to carry six
/// hardcoded hex values of its own, which is why it never matched the rest of
/// the app in either light or dark mode.
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
    if (created == null) return const SizedBox.shrink();

    final waited = (now ?? DateTime.now()).difference(created);
    final tone = switch (waitUrgencyFor(waited, settings)) {
      WaitUrgency.blue => AppBadgeTone.neutral,
      WaitUrgency.orange => AppBadgeTone.warning,
      WaitUrgency.red => AppBadgeTone.danger,
    };

    return AppBadge(_fmtWait(waited), tone: tone);
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
