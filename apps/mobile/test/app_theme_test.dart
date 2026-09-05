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
      // Buttons and selected controls carry the brand colour; ink stays text.
      expect(theme.colorScheme.primary, palette!.primary);
      expect(theme.colorScheme.onPrimary, palette.onPrimary);
      expect(theme.colorScheme.secondary, palette.secondary);
      expect(
          theme.filledButtonTheme.style?.backgroundColor
              ?.resolve(const <WidgetState>{}),
          palette.primary);
      // A translucent container colour would let a card leak its background.
      expect(theme.colorScheme.primaryContainer.a, 1.0);
      expect(theme.colorScheme.secondaryContainer.a, 1.0);
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
    expect(light.primary, isNot(dark.primary));
    expect(light.secondary, isNot(dark.secondary));
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
    // Legacy `accent` call sites resolve to the brand colour.
    expect(resolved.accent, AppPalette.light.primary);
  });
}
