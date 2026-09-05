import 'package:flutter/material.dart';
import 'package:google_fonts/google_fonts.dart';

import 'app_tokens.dart';

/// Builds the single ThemeData used by both customer and admin surfaces.
///
/// Every component the app actually renders is themed here on purpose: an
/// unthemed component is exactly what forces call sites to hardcode colours
/// and radii, which is the debt this refactor removes.
ThemeData buildAppTheme(Brightness brightness) {
  final p = brightness == Brightness.light ? AppPalette.light : AppPalette.dark;
  final scheme = _schemeOf(p, brightness);
  final textTheme = _textThemeOf(p, brightness);

  return ThemeData(
    useMaterial3: true,
    brightness: brightness,
    colorScheme: scheme,
    textTheme: textTheme,
    scaffoldBackgroundColor: p.bg,
    canvasColor: p.bg,
    extensions: <ThemeExtension<dynamic>>[p],
    splashFactory: InkRipple.splashFactory,
    visualDensity: VisualDensity.standard,
    appBarTheme: AppBarTheme(
      backgroundColor: p.bg,
      foregroundColor: p.ink,
      surfaceTintColor: Colors.transparent,
      elevation: 0,
      scrolledUnderElevation: 0,
      centerTitle: false,
      titleSpacing: AppSpacing.lg,
      iconTheme: IconThemeData(color: p.ink, size: 22),
      actionsIconTheme: IconThemeData(color: p.inkMuted, size: 22),
      titleTextStyle: textTheme.titleMedium,
    ),
    cardTheme: CardThemeData(
      color: p.surface,
      surfaceTintColor: Colors.transparent,
      elevation: 0,
      margin: EdgeInsets.zero,
      shape: RoundedRectangleBorder(
        borderRadius: AppRadius.md,
        side: BorderSide(color: p.border),
      ),
    ),
    dividerTheme: DividerThemeData(color: p.border, thickness: 1, space: 1),
    inputDecorationTheme: InputDecorationTheme(
      filled: true,
      fillColor: p.surfaceSubtle,
      hintStyle: textTheme.bodyLarge?.copyWith(color: p.inkFaint),
      labelStyle: textTheme.bodyLarge?.copyWith(color: p.inkMuted),
      floatingLabelStyle: textTheme.labelLarge?.copyWith(color: p.ink),
      prefixIconColor: p.inkFaint,
      suffixIconColor: p.inkFaint,
      contentPadding: const EdgeInsets.symmetric(
        horizontal: AppSpacing.lg,
        vertical: 14,
      ),
      border: OutlineInputBorder(
        borderRadius: AppRadius.md,
        borderSide: BorderSide(color: p.border),
      ),
      enabledBorder: OutlineInputBorder(
        borderRadius: AppRadius.md,
        borderSide: BorderSide(color: p.border),
      ),
      focusedBorder: OutlineInputBorder(
        borderRadius: AppRadius.md,
        borderSide: BorderSide(color: p.ink, width: 1.5),
      ),
      errorBorder: OutlineInputBorder(
        borderRadius: AppRadius.md,
        borderSide: BorderSide(color: p.danger),
      ),
      focusedErrorBorder: OutlineInputBorder(
        borderRadius: AppRadius.md,
        borderSide: BorderSide(color: p.danger, width: 1.5),
      ),
    ),
    // Primary action is an ink block, never the accent.
    filledButtonTheme: FilledButtonThemeData(
      style: FilledButton.styleFrom(
        backgroundColor: p.ink,
        foregroundColor: p.onInk,
        disabledBackgroundColor: p.surfaceSubtle,
        disabledForegroundColor: p.inkFaint,
        minimumSize: const Size(0, 48),
        padding: const EdgeInsets.symmetric(horizontal: AppSpacing.xl),
        elevation: 0,
        shape: const RoundedRectangleBorder(borderRadius: AppRadius.md),
        textStyle: textTheme.labelLarge?.copyWith(fontSize: 15),
      ),
    ),
    outlinedButtonTheme: OutlinedButtonThemeData(
      style: OutlinedButton.styleFrom(
        foregroundColor: p.ink,
        disabledForegroundColor: p.inkFaint,
        side: BorderSide(color: p.borderStrong),
        minimumSize: const Size(0, 48),
        padding: const EdgeInsets.symmetric(horizontal: AppSpacing.xl),
        shape: const RoundedRectangleBorder(borderRadius: AppRadius.md),
        textStyle: textTheme.labelLarge?.copyWith(fontSize: 15),
      ),
    ),
    textButtonTheme: TextButtonThemeData(
      style: TextButton.styleFrom(
        foregroundColor: p.ink,
        disabledForegroundColor: p.inkFaint,
        minimumSize: const Size(0, 40),
        padding: const EdgeInsets.symmetric(horizontal: AppSpacing.md),
        shape: const RoundedRectangleBorder(borderRadius: AppRadius.sm),
        textStyle: textTheme.labelLarge?.copyWith(fontSize: 14),
      ),
    ),
    iconButtonTheme: IconButtonThemeData(
      style: IconButton.styleFrom(
        foregroundColor: p.inkMuted,
        highlightColor: p.surfaceSubtle,
      ),
    ),
    // The one place accent is allowed to fill a shape.
    floatingActionButtonTheme: FloatingActionButtonThemeData(
      backgroundColor: p.accent,
      foregroundColor: p.onAccent,
      elevation: 2,
      focusElevation: 2,
      hoverElevation: 3,
      highlightElevation: 2,
      extendedTextStyle: textTheme.labelLarge?.copyWith(fontSize: 15),
      shape: const RoundedRectangleBorder(borderRadius: AppRadius.md),
    ),
    navigationBarTheme: NavigationBarThemeData(
      backgroundColor: p.surface,
      surfaceTintColor: Colors.transparent,
      indicatorColor: p.surfaceSubtle,
      indicatorShape: const RoundedRectangleBorder(
        borderRadius: AppRadius.full,
      ),
      elevation: 0,
      height: 64,
      labelBehavior: NavigationDestinationLabelBehavior.alwaysShow,
      iconTheme: WidgetStateProperty.resolveWith(
        (states) => IconThemeData(
          size: 22,
          color: states.contains(WidgetState.selected) ? p.ink : p.inkFaint,
        ),
      ),
      labelTextStyle: WidgetStateProperty.resolveWith(
        (states) => textTheme.labelMedium?.copyWith(
          color: states.contains(WidgetState.selected) ? p.ink : p.inkMuted,
          fontWeight: states.contains(WidgetState.selected)
              ? FontWeight.w600
              : FontWeight.w500,
        ),
      ),
    ),
    navigationRailTheme: NavigationRailThemeData(
      backgroundColor: p.bg,
      elevation: 0,
      indicatorColor: p.surfaceSubtle,
      indicatorShape: const RoundedRectangleBorder(borderRadius: AppRadius.md),
      selectedIconTheme: IconThemeData(color: p.ink, size: 22),
      unselectedIconTheme: IconThemeData(color: p.inkFaint, size: 22),
      selectedLabelTextStyle: textTheme.labelMedium?.copyWith(
        color: p.ink,
        fontWeight: FontWeight.w600,
      ),
      unselectedLabelTextStyle: textTheme.labelMedium?.copyWith(
        color: p.inkMuted,
      ),
    ),
    listTileTheme: ListTileThemeData(
      iconColor: p.inkMuted,
      textColor: p.ink,
      titleTextStyle:
          textTheme.bodyLarge?.copyWith(fontWeight: FontWeight.w500),
      subtitleTextStyle: textTheme.bodySmall?.copyWith(color: p.inkMuted),
      contentPadding: const EdgeInsets.symmetric(
        horizontal: AppSpacing.lg,
        vertical: AppSpacing.xs,
      ),
      shape: const RoundedRectangleBorder(borderRadius: AppRadius.md),
    ),
    chipTheme: ChipThemeData(
      backgroundColor: p.surfaceSubtle,
      selectedColor: p.ink,
      disabledColor: p.surfaceSubtle,
      checkmarkColor: p.onInk,
      side: BorderSide(color: p.border),
      shape: const RoundedRectangleBorder(borderRadius: AppRadius.full),
      labelStyle: textTheme.labelLarge?.copyWith(color: p.ink),
      secondaryLabelStyle: textTheme.labelLarge?.copyWith(color: p.onInk),
      labelPadding: const EdgeInsets.symmetric(horizontal: AppSpacing.sm),
      padding: const EdgeInsets.symmetric(
        horizontal: AppSpacing.sm,
        vertical: AppSpacing.sm,
      ),
    ),
    segmentedButtonTheme: SegmentedButtonThemeData(
      style: ButtonStyle(
        backgroundColor: WidgetStateProperty.resolveWith(
          (states) => states.contains(WidgetState.selected)
              ? p.ink
              : Colors.transparent,
        ),
        foregroundColor: WidgetStateProperty.resolveWith(
          (states) =>
              states.contains(WidgetState.selected) ? p.onInk : p.inkMuted,
        ),
        side: WidgetStatePropertyAll(BorderSide(color: p.border)),
        textStyle: WidgetStatePropertyAll(textTheme.labelLarge),
        shape: const WidgetStatePropertyAll(
          RoundedRectangleBorder(borderRadius: AppRadius.sm),
        ),
      ),
    ),
    switchTheme: SwitchThemeData(
      thumbColor: WidgetStateProperty.resolveWith(
        (states) => states.contains(WidgetState.selected) ? p.onInk : p.surface,
      ),
      trackColor: WidgetStateProperty.resolveWith(
        (states) =>
            states.contains(WidgetState.selected) ? p.ink : p.surfaceSubtle,
      ),
      trackOutlineColor: WidgetStateProperty.resolveWith(
        (states) =>
            states.contains(WidgetState.selected) ? p.ink : p.borderStrong,
      ),
    ),
    checkboxTheme: CheckboxThemeData(
      fillColor: WidgetStateProperty.resolveWith(
        (states) =>
            states.contains(WidgetState.selected) ? p.ink : Colors.transparent,
      ),
      checkColor: WidgetStatePropertyAll(p.onInk),
      side: BorderSide(color: p.borderStrong, width: 1.5),
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.all(Radius.circular(4)),
      ),
    ),
    radioTheme: RadioThemeData(
      fillColor: WidgetStateProperty.resolveWith(
        (states) =>
            states.contains(WidgetState.selected) ? p.ink : p.borderStrong,
      ),
    ),
    dialogTheme: DialogThemeData(
      backgroundColor: p.surface,
      surfaceTintColor: Colors.transparent,
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: AppRadius.md,
        side: BorderSide(color: p.border),
      ),
      titleTextStyle: textTheme.titleMedium,
      contentTextStyle: textTheme.bodyLarge?.copyWith(color: p.inkMuted),
    ),
    bottomSheetTheme: BottomSheetThemeData(
      backgroundColor: p.surface,
      surfaceTintColor: Colors.transparent,
      elevation: 0,
      dragHandleColor: p.borderStrong,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(
          top: Radius.circular(AppRadius.mdValue),
        ),
      ),
    ),
    popupMenuTheme: PopupMenuThemeData(
      color: p.surface,
      surfaceTintColor: Colors.transparent,
      elevation: 2,
      textStyle: textTheme.bodyLarge,
      shape: RoundedRectangleBorder(
        borderRadius: AppRadius.md,
        side: BorderSide(color: p.border),
      ),
    ),
    snackBarTheme: SnackBarThemeData(
      backgroundColor: p.ink,
      contentTextStyle: textTheme.bodyMedium?.copyWith(color: p.onInk),
      actionTextColor: p.accent,
      behavior: SnackBarBehavior.floating,
      elevation: 0,
      shape: const RoundedRectangleBorder(borderRadius: AppRadius.sm),
    ),
    tabBarTheme: TabBarThemeData(
      labelColor: p.ink,
      unselectedLabelColor: p.inkMuted,
      indicatorColor: p.ink,
      dividerColor: p.border,
      labelStyle: textTheme.labelLarge,
      unselectedLabelStyle: textTheme.labelLarge,
    ),
    progressIndicatorTheme: ProgressIndicatorThemeData(
      color: p.ink,
      linearTrackColor: p.surfaceSubtle,
      circularTrackColor: Colors.transparent,
    ),
    tooltipTheme: TooltipThemeData(
      decoration: BoxDecoration(color: p.ink, borderRadius: AppRadius.sm),
      textStyle: textTheme.bodySmall?.copyWith(color: p.onInk),
    ),
  );
}

ColorScheme _schemeOf(AppPalette p, Brightness brightness) {
  return ColorScheme(
    brightness: brightness,
    primary: p.ink,
    onPrimary: p.onInk,
    primaryContainer: p.surfaceSubtle,
    onPrimaryContainer: p.ink,
    secondary: p.accent,
    onSecondary: p.onAccent,
    secondaryContainer: p.surfaceSubtle,
    onSecondaryContainer: p.ink,
    tertiary: p.accent,
    onTertiary: p.onAccent,
    error: p.danger,
    onError: p.onAccent,
    errorContainer: p.surfaceSubtle,
    onErrorContainer: p.danger,
    surface: p.bg,
    onSurface: p.ink,
    onSurfaceVariant: p.inkMuted,
    surfaceContainerLowest: p.surface,
    surfaceContainerLow: p.bgSubtle,
    surfaceContainer: p.surfaceSubtle,
    surfaceContainerHigh: p.surfaceSubtle,
    surfaceContainerHighest: p.surfaceSubtle,
    outline: p.border,
    outlineVariant: p.border,
    inverseSurface: p.ink,
    onInverseSurface: p.onInk,
    shadow: const Color(0x1A000000),
    scrim: const Color(0x66000000),
  );
}

TextTheme _textThemeOf(AppPalette p, Brightness brightness) {
  final base = GoogleFonts.beVietnamProTextTheme(
    brightness == Brightness.light
        ? ThemeData.light().textTheme
        : ThemeData.dark().textTheme,
  );

  // Weight tops out at w600 — w800/w900 is not minimalism.
  return base
      .copyWith(
        headlineLarge: base.headlineLarge?.copyWith(
          fontSize: 32,
          fontWeight: FontWeight.w600,
          letterSpacing: -0.4,
          height: 1.15,
        ),
        headlineMedium: base.headlineMedium?.copyWith(
          fontSize: 26,
          fontWeight: FontWeight.w600,
          letterSpacing: -0.3,
          height: 1.2,
        ),
        headlineSmall: base.headlineSmall?.copyWith(
          fontSize: 22,
          fontWeight: FontWeight.w600,
          letterSpacing: -0.2,
          height: 1.25,
        ),
        titleLarge: base.titleLarge?.copyWith(
          fontSize: 22,
          fontWeight: FontWeight.w600,
          letterSpacing: -0.2,
          height: 1.25,
        ),
        titleMedium: base.titleMedium?.copyWith(
          fontSize: 17,
          fontWeight: FontWeight.w600,
          letterSpacing: -0.1,
          height: 1.3,
        ),
        titleSmall: base.titleSmall?.copyWith(
          fontSize: 15,
          fontWeight: FontWeight.w600,
          height: 1.3,
        ),
        bodyLarge: base.bodyLarge?.copyWith(
          fontSize: 15,
          fontWeight: FontWeight.w400,
          height: 1.45,
        ),
        bodyMedium: base.bodyMedium?.copyWith(
          fontSize: 14,
          fontWeight: FontWeight.w400,
          height: 1.45,
        ),
        bodySmall: base.bodySmall?.copyWith(
          fontSize: 12,
          fontWeight: FontWeight.w400,
          height: 1.4,
        ),
        labelLarge: base.labelLarge?.copyWith(
          fontSize: 13,
          fontWeight: FontWeight.w600,
          letterSpacing: 0,
        ),
        labelMedium: base.labelMedium?.copyWith(
          fontSize: 12,
          fontWeight: FontWeight.w500,
          letterSpacing: 0.1,
        ),
        labelSmall: base.labelSmall?.copyWith(
          fontSize: 11,
          fontWeight: FontWeight.w500,
          letterSpacing: 0.2,
        ),
      )
      .apply(bodyColor: p.ink, displayColor: p.ink, decorationColor: p.ink);
}
