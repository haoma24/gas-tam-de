import 'dart:convert';

import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:shared_preferences/shared_preferences.dart';

import '../auth/auth_session.dart';
import 'geo_models.dart';

class SavedAddress {
  const SavedAddress({
    required this.id,
    required this.label,
    required this.lat,
    required this.lng,
    required this.name,
    this.isDefault = false,
  });

  final String id;
  final String label;
  final double lat;
  final double lng;
  final String name;
  final bool isDefault;

  SelectedAddress get selection =>
      SelectedAddress(lat: lat, lng: lng, label: label);

  SavedAddress copyWith({bool? isDefault}) => SavedAddress(
        id: id,
        label: label,
        lat: lat,
        lng: lng,
        name: name,
        isDefault: isDefault ?? this.isDefault,
      );

  factory SavedAddress.fromJson(Map<String, dynamic> json) => SavedAddress(
        id: json['id'] as String? ?? '',
        label: json['label'] as String? ?? '',
        lat: geoAsDouble(json['lat']),
        lng: geoAsDouble(json['lng']),
        name: json['name'] as String? ?? 'Địa chỉ giao hàng',
        isDefault: json['is_default'] == true,
      );

  Map<String, dynamic> toJson() => {
        'id': id,
        'label': label,
        'lat': lat,
        'lng': lng,
        'name': name,
        'is_default': isDefault,
      };
}

class SavedAddressStore {
  SavedAddressStore(this._prefs);

  final SharedPreferences _prefs;

  String _key(String userId) => 'gas_tam_de.addresses.v1.$userId';

  List<SavedAddress> load(String userId) {
    final raw = _prefs.getString(_key(userId));
    if (raw == null || raw.isEmpty) return const [];
    try {
      final decoded = jsonDecode(raw);
      if (decoded is! List) return const [];
      return decoded
          .whereType<Map>()
          .map((item) => SavedAddress.fromJson(Map<String, dynamic>.from(item)))
          .where((item) => item.id.isNotEmpty && item.label.isNotEmpty)
          .toList(growable: false);
    } catch (_) {
      return const [];
    }
  }

  Future<void> save(String userId, List<SavedAddress> items) async {
    await _prefs.setString(
      _key(userId),
      jsonEncode(items.map((item) => item.toJson()).toList()),
    );
  }
}

final savedAddressStoreProvider = Provider<SavedAddressStore>((ref) {
  return SavedAddressStore(ref.watch(sharedPreferencesProvider));
});
