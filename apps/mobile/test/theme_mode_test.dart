import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gas_tam_de/core/ui/ui.dart';
import 'package:gas_tam_de/features/auth/auth_session.dart';
import 'package:shared_preferences/shared_preferences.dart';

Future<ProviderContainer> _container(Map<String, Object> seed) async {
  SharedPreferences.setMockInitialValues(seed);
  final prefs = await SharedPreferences.getInstance();
  return ProviderContainer(
    overrides: [sharedPreferencesProvider.overrideWithValue(prefs)],
  );
}

void main() {
  testWidgets('theme mode defaults to light and survives a restart',
      (tester) async {
    final container = await _container({});
    addTearDown(container.dispose);

    // Light is the designed look; a machine sitting in dark mode used to drag
    // the whole app dark on first run.
    expect(container.read(themeModeProvider), ThemeMode.light);

    await container.read(themeModeProvider.notifier).set(ThemeMode.dark);
    expect(container.read(themeModeProvider), ThemeMode.dark);

    // A fresh container reads the same SharedPreferences the app would.
    final restarted = await _container(
      {'gas_tam_de.theme_mode.v1': 'dark'},
    );
    addTearDown(restarted.dispose);
    expect(restarted.read(themeModeProvider), ThemeMode.dark);

    // «Hệ thống» is still reachable, it is just no longer the default.
    final system = await _container(
      {'gas_tam_de.theme_mode.v1': 'system'},
    );
    addTearDown(system.dispose);
    expect(system.read(themeModeProvider), ThemeMode.system);
  });

  testWidgets('the settings section switches the app to light mode',
      (tester) async {
    final container = await _container({});
    addTearDown(container.dispose);

    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: container,
        child: Consumer(
          builder: (context, ref, _) => MaterialApp(
            theme: buildAppTheme(Brightness.light),
            darkTheme: buildAppTheme(Brightness.dark),
            themeMode: ref.watch(themeModeProvider),
            home: const Scaffold(
              body: SingleChildScrollView(child: AppThemeModeSection()),
            ),
          ),
        ),
      ),
    );

    expect(find.text('Giao diện'), findsOneWidget);
    for (final label in ['Hệ thống', 'Sáng', 'Tối']) {
      expect(find.text(label), findsOneWidget);
    }

    await tester.tap(find.text('Tối'));
    await tester.pumpAndSettle();
    expect(container.read(themeModeProvider), ThemeMode.dark);

    await tester.tap(find.text('Sáng'));
    await tester.pumpAndSettle();
    expect(container.read(themeModeProvider), ThemeMode.light);
  });
}
