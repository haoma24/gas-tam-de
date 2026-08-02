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
      lat: _asDouble(json['lat']),
      lng: _asDouble(json['lng']),
      source: (json['source'] as String?)?.trim() ?? '',
    );
  }

  static double _asDouble(Object? v) {
    if (v is num) return v.toDouble();
    if (v is String) return double.tryParse(v) ?? 0;
    return 0;
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
      distanceKm: _asDouble(json['distance_km']),
      inRange: json['in_range'] == true,
      maxRadiusKm: _asDouble(json['max_radius_km']),
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

  static double _asDouble(Object? v) {
    if (v is num) return v.toDouble();
    if (v is String) return double.tryParse(v) ?? 0;
    return 0;
  }
}
