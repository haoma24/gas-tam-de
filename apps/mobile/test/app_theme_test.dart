import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gas_tam_de/core/ui/ui.dart';

void main() {
  test('theme builds for both brightnesses and carries the palette', () {
    for (final brightness in Brightness.values) {
      final theme = buildAppTheme(brightness);
      final palette = theme.extension<AppPalette>();

      expect(palette, isNotNull, reason: '$brightness must carry AppPalette');
      expect(theme.brightness, brightness);
      // Primary is the ink block, not the accent — the core of the palette rule.
      expect(theme.colorScheme.primary, palette!.ink);
      expect(theme.colorScheme.onPrimary, palette.onInk);
      expect(theme.scaffoldBackgroundColor, palette.bg);
      // Elevation is expressed as a hairline, so cards stay flat.
      expect(theme.cardTheme.elevation, 0);
      expect(theme.appBarTheme.elevation, 0);
    }
  });

  test('light and dark differ on every surface token', () {
    final light = AppPalette.light;
    final dark = AppPalette.dark;
    expect(light.bg, isNot(dark.bg));
    expect(light.ink, isNot(dark.ink));
    expect(light.surface, isNot(dark.surface));
    expect(light.border, isNot(dark.border));
  });

  testWidgets('context.palette resolves through the theme', (tester) async {
    late AppPalette resolved;
    await tester.pumpWidget(
      MaterialApp(
        theme: buildAppTheme(Brightness.light),
        home: Builder(
          builder: (context) {
            resolved = context.palette;
            return const SizedBox.shrink();
          },
        ),
      ),
    );
    expect(resolved.ink, AppPalette.light.ink);
  });
}
