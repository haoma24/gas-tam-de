import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:shared_preferences/shared_preferences.dart';

import '../../features/auth/auth_session.dart' show sharedPreferencesProvider;
import 'app_section.dart';
import 'app_tokens.dart';

const _kThemeModeKey = 'gas_tam_de.theme_mode.v1';

/// Light / dark preference, persisted locally.
///
/// [ThemeMode.light] is the default. The shop opens the desk on whatever
/// machine is free and several of those sit in dark mode, which used to drag
/// the whole app dark on first run; light is the intended look and 「Hệ
/// thống」 is the opt-in.
class ThemeModeController extends StateNotifier<ThemeMode> {
  ThemeModeController(this._prefs) : super(_restore(_prefs));

  final SharedPreferences _prefs;

  static ThemeMode _restore(SharedPreferences prefs) {
    switch (prefs.getString(_kThemeModeKey)) {
      case 'light':
        return ThemeMode.light;
      case 'dark':
        return ThemeMode.dark;
      case 'system':
        return ThemeMode.system;
      default:
        return ThemeMode.light;
    }
  }

  Future<void> set(ThemeMode mode) async {
    if (mode == state) return;
    state = mode;
    await _prefs.setString(_kThemeModeKey, mode.name);
  }
}

final themeModeProvider =
    StateNotifierProvider<ThemeModeController, ThemeMode>((ref) {
  return ThemeModeController(ref.watch(sharedPreferencesProvider));
});

String _labelOf(ThemeMode mode) => switch (mode) {
      ThemeMode.system => 'Hệ thống',
      ThemeMode.light => 'Sáng',
      ThemeMode.dark => 'Tối',
    };

IconData _iconOf(ThemeMode mode) => switch (mode) {
      ThemeMode.system => Icons.brightness_auto_outlined,
      ThemeMode.light => Icons.light_mode_outlined,
      ThemeMode.dark => Icons.dark_mode_outlined,
    };

/// «Giao diện» block — the light / dark switch, shared by the customer profile
/// and the admin settings tab.
class AppThemeModeSection extends ConsumerWidget {
  const AppThemeModeSection({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final mode = ref.watch(themeModeProvider);

    return AppSection(
      title: 'Giao diện',
      icon: _iconOf(mode),
      children: [
        SizedBox(
          width: double.infinity,
          child: SegmentedButton<ThemeMode>(
            segments: [
              for (final m in ThemeMode.values)
                ButtonSegment(value: m, label: Text(_labelOf(m))),
            ],
            selected: {mode},
            showSelectedIcon: false,
            onSelectionChanged: (set) =>
                ref.read(themeModeProvider.notifier).set(set.first),
          ),
        ),
        const VGap(AppSpacing.sm),
        Text(
          'Chọn «Hệ thống» để đi theo cài đặt sáng / tối của máy.',
          style: context.text.bodySmall?.copyWith(
            color: context.palette.inkMuted,
          ),
        ),
      ],
    );
  }
}
