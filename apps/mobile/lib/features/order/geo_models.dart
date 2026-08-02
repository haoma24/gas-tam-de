/// One place suggestion from geo-service `GET /v1/geo/search`.
class GeoPlace {
  const GeoPlace({
    required this.label,
    required this.lat,
    required this.lng,
    this.source = '',
  });

  final String label;
  final double lat;
  final double lng;
  final String source;

  factory GeoPlace.fromJson(Map<String, dynamic> json) {
    return GeoPlace(
      label: (json['label'] as String?)?.trim() ?? '',
      lat: geoAsDouble(json['lat']),
      lng: geoAsDouble(json['lng']),
      source: (json['source'] as String?)?.trim() ?? '',
    );
  }
}

/// Selected delivery pin for the order flow (lat/lng + human label).
class SelectedAddress {
  const SelectedAddress({
    required this.lat,
    required this.lng,
    required this.label,
  });

  final double lat;
  final double lng;
  final String label;

  SelectedAddress copyWith({double? lat, double? lng, String? label}) {
    return SelectedAddress(
      lat: lat ?? this.lat,
      lng: lng ?? this.lng,
      label: label ?? this.label,
    );
  }
}

/// Result of `POST /v1/geo/check` — Haversine distance vs store radius (T3.2.2 / T3.2.3).
class GeoCheckResult {
  const GeoCheckResult({
    required this.distanceKm,
    required this.inRange,
    required this.maxRadiusKm,
  });

  final double distanceKm;
  final bool inRange;
  final double maxRadiusKm;

  factory GeoCheckResult.fromJson(Map<String, dynamic> json) {
    return GeoCheckResult(
      distanceKm: geoAsDouble(json['distance_km']),
      inRange: json['in_range'] == true,
      maxRadiusKm: geoAsDouble(json['max_radius_km']),
    );
  }

  /// Clear VN copy when [inRange] is false (T3.2.3).
  String get outOfRangeMessage {
    final dist = _fmtKm(distanceKm);
    final max = _fmtKm(maxRadiusKm);
    return 'Địa chỉ ngoài phạm vi giao hàng '
        '(khoảng $dist km, tối đa $max km). '
        'Vui lòng chọn vị trí gần cửa hàng hơn.';
  }

  String get inRangeHint {
    final dist = _fmtKm(distanceKm);
    final max = _fmtKm(maxRadiusKm);
    return 'Trong phạm vi giao · khoảng $dist km (tối đa $max km).';
  }

  static String _fmtKm(double km) {
    if (km == km.roundToDouble()) return km.toStringAsFixed(0);
    return km.toStringAsFixed(2);
  }
}

/// Shop geo fence from `GET /v1/geo/store` / `PUT /v1/admin/geo/store`.
class StoreSettings {
  const StoreSettings({
    required this.name,
    required this.lat,
    required this.lng,
    required this.maxRadiusKm,
    this.id,
    this.addressText,
    this.updatedAt,
  });

  final String? id;
  final String name;
  final double lat;
  final double lng;
  final double maxRadiusKm;
  final String? addressText;
  final String? updatedAt;

  factory StoreSettings.fromJson(Map<String, dynamic> json) {
    final address = (json['address_text'] as String?)?.trim();
    return StoreSettings(
      id: json['id'] as String?,
      name: (json['name'] as String?)?.trim() ?? '',
      lat: geoAsDouble(json['lat']),
      lng: geoAsDouble(json['lng']),
      maxRadiusKm: geoAsDouble(json['max_radius_km']),
      addressText: (address == null || address.isEmpty) ? null : address,
      updatedAt: json['updated_at'] as String?,
    );
  }
}

double geoAsDouble(Object? v) {
  if (v is num) return v.toDouble();
  if (v is String) return double.tryParse(v) ?? 0;
  return 0;
}
