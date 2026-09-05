import 'package:flutter/widgets.dart';

/// Layout width classes. The admin shell switches between a bottom
/// NavigationBar and a left NavigationRail + two-pane layout on [expanded].
enum AppWidthClass { compact, medium, expanded }

abstract final class AppBreakpoints {
  /// Below this a phone layout is assumed.
  static const double medium = 600;

  /// At or above this the admin shell shows a rail and a detail pane.
  static const double expanded = 900;

  static AppWidthClass of(double width) {
    if (width >= expanded) return AppWidthClass.expanded;
    if (width >= medium) return AppWidthClass.medium;
    return AppWidthClass.compact;
  }
}

extension AppBreakpointsX on BuildContext {
  AppWidthClass get widthClass =>
      AppBreakpoints.of(MediaQuery.sizeOf(this).width);

  /// Rail + two-pane territory.
  bool get isExpanded => widthClass == AppWidthClass.expanded;

  /// Phone territory — bottom nav, single column, full-screen detail.
  bool get isCompact => widthClass == AppWidthClass.compact;

  /// Content column cap so text does not run edge-to-edge on desktop.
  double get contentMaxWidth => isExpanded ? 720 : double.infinity;
}
