import 'package:flutter/material.dart';

/// Semantic design tokens for Gas Tam Đệ.
///
/// Tokens are named by ROLE, not by colour, so light and dark resolve through
/// one call site: `context.palette.ink`. No hex literal belongs outside this
/// file — if a screen needs a colour that is not here, add a token instead.
@immutable
class AppPalette extends ThemeExtension<AppPalette> {
  const AppPalette({
    required this.bg,
    required this.bgSubtle,
    required this.surface,
    required this.surfaceSubtle,
    required this.border,
    required this.borderStrong,
    required this.ink,
    required this.inkMuted,
    required this.inkFaint,
    required this.onInk,
    required this.primary,
    required this.onPrimary,
    required this.secondary,
    required this.onSecondary,
    required this.success,
    required this.warning,
    required this.danger,
  });

  /// Page background.
  final Color bg;

  /// Recessed background for grouped blocks.
  final Color bgSubtle;

  /// Card / sheet fill.
  final Color surface;

  /// Input fill, alternating rows, image placeholders.
  final Color surfaceSubtle;

  /// 1px hairline — this is how elevation is expressed, not shadow.
  final Color border;

  /// Stronger hairline for emphasis (focused input, selected card).
  final Color borderStrong;

  /// Primary text, headings and icons.
  final Color ink;

  /// Secondary text.
  final Color inkMuted;

  /// Placeholder text, disabled icons.
  final Color inkFaint;

  /// Text/icon on top of an [ink] fill.
  final Color onInk;

  /// Brand colour: fills every primary button, the FAB and the selected state
  /// of a control. A screen built only from [ink] and greys reads as
  /// unfinished — this is the colour that tells the eye where the action is.
  final Color primary;

  final Color onPrimary;

  /// Informational colour: counts, links, neutral badges — what the eye should
  /// find quickly without it competing with [primary] for the tap.
  final Color secondary;

  final Color onSecondary;

  final Color success;
  final Color warning;
  final Color danger;

  /// The accent and the primary action are the same brand colour, so the
  /// existing `accent` call sites resolve here instead of being renamed.
  Color get accent => primary;
  Color get onAccent => onPrimary;

  static const light = AppPalette(
    bg: Color(0xFFFFFFFF),
    bgSubtle: Color(0xFFFAFAFA),
    surface: Color(0xFFFFFFFF),
    surfaceSubtle: Color(0xFFF5F5F5),
    border: Color(0xFFE5E5E5),
    borderStrong: Color(0xFFD4D4D4),
    ink: Color(0xFF171717),
    inkMuted: Color(0xFF737373),
    inkFaint: Color(0xFFA3A3A3),
    onInk: Color(0xFFFFFFFF),
    primary: Color(0xFFEA580C),
    onPrimary: Color(0xFFFFFFFF),
    secondary: Color(0xFF0284C7),
    onSecondary: Color(0xFFFFFFFF),
    success: Color(0xFF16A34A),
    warning: Color(0xFFD97706),
    danger: Color(0xFFDC2626),
  );

  static const dark = AppPalette(
    bg: Color(0xFF0A0A0A),
    bgSubtle: Color(0xFF141414),
    surface: Color(0xFF171717),
    surfaceSubtle: Color(0xFF1F1F1F),
    border: Color(0xFF2A2A2A),
    borderStrong: Color(0xFF3A3A3A),
    ink: Color(0xFFFAFAFA),
    inkMuted: Color(0xFFA3A3A3),
    inkFaint: Color(0xFF737373),
    onInk: Color(0xFF171717),
    primary: Color(0xFFFB923C),
    onPrimary: Color(0xFF1A1A1A),
    secondary: Color(0xFF38BDF8),
    onSecondary: Color(0xFF0A1A22),
    success: Color(0xFF4ADE80),
    warning: Color(0xFFFBBF24),
    danger: Color(0xFFF87171),
  );

  @override
  AppPalette copyWith({
    Color? bg,
    Color? bgSubtle,
    Color? surface,
    Color? surfaceSubtle,
    Color? border,
    Color? borderStrong,
    Color? ink,
    Color? inkMuted,
    Color? inkFaint,
    Color? onInk,
    Color? primary,
    Color? onPrimary,
    Color? secondary,
    Color? onSecondary,
    Color? success,
    Color? warning,
    Color? danger,
  }) {
    return AppPalette(
      bg: bg ?? this.bg,
      bgSubtle: bgSubtle ?? this.bgSubtle,
      surface: surface ?? this.surface,
      surfaceSubtle: surfaceSubtle ?? this.surfaceSubtle,
      border: border ?? this.border,
      borderStrong: borderStrong ?? this.borderStrong,
      ink: ink ?? this.ink,
      inkMuted: inkMuted ?? this.inkMuted,
      inkFaint: inkFaint ?? this.inkFaint,
      onInk: onInk ?? this.onInk,
      primary: primary ?? this.primary,
      onPrimary: onPrimary ?? this.onPrimary,
      secondary: secondary ?? this.secondary,
      onSecondary: onSecondary ?? this.onSecondary,
      success: success ?? this.success,
      warning: warning ?? this.warning,
      danger: danger ?? this.danger,
    );
  }

  @override
  AppPalette lerp(covariant ThemeExtension<AppPalette>? other, double t) {
    if (other is! AppPalette) return this;
    return AppPalette(
      bg: Color.lerp(bg, other.bg, t)!,
      bgSubtle: Color.lerp(bgSubtle, other.bgSubtle, t)!,
      surface: Color.lerp(surface, other.surface, t)!,
      surfaceSubtle: Color.lerp(surfaceSubtle, other.surfaceSubtle, t)!,
      border: Color.lerp(border, other.border, t)!,
      borderStrong: Color.lerp(borderStrong, other.borderStrong, t)!,
      ink: Color.lerp(ink, other.ink, t)!,
      inkMuted: Color.lerp(inkMuted, other.inkMuted, t)!,
      inkFaint: Color.lerp(inkFaint, other.inkFaint, t)!,
      onInk: Color.lerp(onInk, other.onInk, t)!,
      primary: Color.lerp(primary, other.primary, t)!,
      onPrimary: Color.lerp(onPrimary, other.onPrimary, t)!,
      secondary: Color.lerp(secondary, other.secondary, t)!,
      onSecondary: Color.lerp(onSecondary, other.onSecondary, t)!,
      success: Color.lerp(success, other.success, t)!,
      warning: Color.lerp(warning, other.warning, t)!,
      danger: Color.lerp(danger, other.danger, t)!,
    );
  }
}

/// Spacing scale. Every gap in the app comes from here.
abstract final class AppSpacing {
  static const double xs = 4;
  static const double sm = 8;
  static const double md = 12;
  static const double lg = 16;
  static const double xl = 24;
  static const double xxl = 32;
  static const double huge = 48;

  /// Horizontal page gutter.
  static const EdgeInsets pageH = EdgeInsets.symmetric(horizontal: lg);
}

/// Vertical gap — `const VGap(AppSpacing.lg)` instead of a bare SizedBox.
class VGap extends StatelessWidget {
  const VGap(this.size, {super.key});
  final double size;

  @override
  Widget build(BuildContext context) => SizedBox(height: size);
}

/// Horizontal gap.
class HGap extends StatelessWidget {
  const HGap(this.size, {super.key});
  final double size;

  @override
  Widget build(BuildContext context) => SizedBox(width: size);
}

/// Three radii only. Larger radii read as playful, not minimal.
abstract final class AppRadius {
  static const double smValue = 8;
  static const double mdValue = 12;

  static const sm = BorderRadius.all(Radius.circular(smValue));
  static const md = BorderRadius.all(Radius.circular(mdValue));
  static const full = BorderRadius.all(Radius.circular(999));
}

abstract final class AppDuration {
  static const fast = Duration(milliseconds: 150);
  static const base = Duration(milliseconds: 200);
}

/// Money and other numerals that must line up in a column.
const kTabularFigures = <FontFeature>[FontFeature.tabularFigures()];

extension AppThemeX on BuildContext {
  /// Semantic colours. Falls back to light so a bare `MaterialApp` in a widget
  /// test still renders instead of throwing.
  AppPalette get palette =>
      Theme.of(this).extension<AppPalette>() ?? AppPalette.light;

  TextTheme get text => Theme.of(this).textTheme;
}
