import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/api_client.dart';
import 'desk_settings_models.dart';

final deskSettingsApiProvider = Provider<DeskSettingsApi>((ref) {
  return DeskSettingsApi(ref.watch(dioProvider));
});

class DeskSettingsApi {
  DeskSettingsApi(this._dio);
  final Dio _dio;

  Future<DeskSettings> get() async {
    final res = await _dio.get<Map<String, dynamic>>('/v1/admin/desk-settings');
    final data = res.data;
    if (data == null) throw StateError('empty desk settings');
    return DeskSettings.fromJson(data);
  }

  Future<DeskSettings> put(DeskSettings settings) async {
    final res = await _dio.put<Map<String, dynamic>>(
      '/v1/admin/desk-settings',
      data: settings.toPutJson(),
    );
    final data = res.data;
    if (data == null) throw StateError('empty desk settings');
    return DeskSettings.fromJson(data);
  }
}

final deskSettingsProvider = FutureProvider<DeskSettings>((ref) async {
  try {
    return await ref.read(deskSettingsApiProvider).get();
  } catch (_) {
    return DeskSettings.defaults;
  }
});
