import 'package:flutter/foundation.dart' show TargetPlatform, defaultTargetPlatform, kIsWeb;
import 'package:url_launcher/url_launcher.dart';

/// Outcome of opening an external maps / navigation app.
class NavigationLaunchResult {
  const NavigationLaunchResult._({this.errorMessage});

  factory NavigationLaunchResult.ok() => const NavigationLaunchResult._();

  factory NavigationLaunchResult.fail(String message) =>
      NavigationLaunchResult._(errorMessage: message);

  final String? errorMessage;

  bool get isOk => errorMessage == null;
}

/// Google Maps directions (HTTPS) — Web primary + universal fallback.
///
/// Omits `origin` so the Maps app / browser uses the device's current
/// location as the start (PRD M5 / US-5.2).
Uri googleMapsDirectionsUri(double lat, double lng) {
  final dest = _coordPair(lat, lng);
  return Uri.parse(
    'https://www.google.com/maps/dir/?api=1&destination=$dest&travelmode=driving',
  );
}

/// Android `geo:` intent (opens any installed maps app / chooser).
Uri geoIntentUri(double lat, double lng) {
  final dest = _coordPair(lat, lng);
  return Uri(scheme: 'geo', path: dest, queryParameters: {'q': dest});
}

/// Android Google Maps turn-by-turn navigation scheme.
///
/// Opaque form `google.navigation:q=lat,lng&mode=d` (no `?`) — required by
/// the Google Maps app intent filter.
Uri googleNavigationUri(double lat, double lng) {
  final dest = _coordPair(lat, lng);
  return Uri(scheme: 'google.navigation', path: 'q=$dest&mode=d');
}

/// Google Maps app deep-link (iOS / Android when app installed).
Uri googleMapsAppDirectionsUri(double lat, double lng) {
  final dest = _coordPair(lat, lng);
  return Uri(
    scheme: 'comgooglemaps',
    host: '',
    queryParameters: {
      'daddr': dest,
      'directionsmode': 'driving',
    },
  );
}

/// Apple Maps directions (`maps://`).
Uri appleMapsDirectionsUri(double lat, double lng) {
  final dest = _coordPair(lat, lng);
  return Uri(
    scheme: 'maps',
    host: '',
    queryParameters: {
      'daddr': dest,
      'dirflg': 'd',
    },
  );
}

/// Candidate URIs to try, in priority order for the current platform.
///
/// Pure helper — useful for tests / debugging without launching.
List<Uri> navigationCandidateUris(double lat, double lng) {
  _assertFinite(lat, lng);
  final https = googleMapsDirectionsUri(lat, lng);
  if (kIsWeb) {
    return [https];
  }
  switch (defaultTargetPlatform) {
    case TargetPlatform.android:
      return [
        googleNavigationUri(lat, lng),
        geoIntentUri(lat, lng),
        googleMapsAppDirectionsUri(lat, lng),
        https,
      ];
    case TargetPlatform.iOS:
      return [
        googleMapsAppDirectionsUri(lat, lng),
        appleMapsDirectionsUri(lat, lng),
        https,
      ];
    default:
      // Desktop / others — HTTPS is the reliable path.
      return [https];
  }
}

/// Open turn-by-turn navigation to destination [lat]/[lng] (WGS84).
///
/// Tries platform-native schemes first (Google Maps / `geo:` / Apple Maps),
/// then falls back to the HTTPS Google Maps directions URL. Uses
/// [LaunchMode.externalApplication] so Web opens a new tab / external browser.
///
/// Used by Order Desk «Dẫn đường» on chi tiết đơn (T5.2.3).
Future<NavigationLaunchResult> openNavigationTo(double lat, double lng) async {
  if (!_isFiniteCoord(lat) || !_isFiniteCoord(lng)) {
    return NavigationLaunchResult.fail(
      'Toạ độ điểm giao không hợp lệ.',
    );
  }
  if (lat < -90 || lat > 90 || lng < -180 || lng > 180) {
    return NavigationLaunchResult.fail(
      'Toạ độ điểm giao nằm ngoài phạm vi WGS84.',
    );
  }

  final candidates = navigationCandidateUris(lat, lng);
  for (final uri in candidates) {
    try {
      final ok = await launchUrl(uri, mode: LaunchMode.externalApplication);
      if (ok) {
        return NavigationLaunchResult.ok();
      }
    } catch (_) {
      // Try next candidate (scheme unsupported / app missing).
    }
  }

  return NavigationLaunchResult.fail(
    'Không mở được ứng dụng bản đồ. Cài Google Maps / Apple Maps hoặc thử lại trên trình duyệt.',
  );
}

String _coordPair(double lat, double lng) =>
    '${lat.toStringAsFixed(6)},${lng.toStringAsFixed(6)}';

void _assertFinite(double lat, double lng) {
  if (!_isFiniteCoord(lat) || !_isFiniteCoord(lng)) {
    throw ArgumentError('lat/lng must be finite numbers');
  }
}

bool _isFiniteCoord(double v) => v.isFinite;
