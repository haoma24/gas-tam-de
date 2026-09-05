import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:shared_preferences/shared_preferences.dart';

import '../../features/auth/auth_session.dart' show sharedPreferencesProvider;
import 'app_section.dart';
import 'app_tokens.dart';

const _kThemeModeKey = 'gas_tam_de.theme_mode.v1';

/// Light / dark preference, persisted locally.
///
/// [ThemeMode.system] is the default: the app used to be light-only, so a
/// phone or browser set to dark suddenly turned the whole UI dark with no way
/// back. This controller is that way back.
class ThemeModeController extends StateNotifier<ThemeMode> {
  ThemeModeController(this._prefs) : super(_restore(_prefs));

  final SharedPreferences _prefs;

  static ThemeMode _restore(SharedPreferences prefs) {
    switch (prefs.getString(_kThemeModeKey)) {
      case 'light':
        return ThemeMode.light;
      case 'dark':
        return ThemeMode.dark;
      default:
        return ThemeMode.system;
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
