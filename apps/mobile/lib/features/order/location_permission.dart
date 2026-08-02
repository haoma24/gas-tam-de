import 'dart:async';

import 'package:flutter/foundation.dart' show kIsWeb;
import 'package:geolocator/geolocator.dart';

/// Outcome of requesting location permission and reading the current position.
class LocationResult {
  const LocationResult._({this.position, this.errorMessage});

  factory LocationResult.ok(Position position) =>
      LocationResult._(position: position);

  factory LocationResult.fail(String message) =>
      LocationResult._(errorMessage: message);

  final Position? position;
  final String? errorMessage;

  bool get isOk => position != null;
}

/// Xin quyền vị trí + lấy tọa độ hiện tại (Web / Android / iOS).
///
/// Dùng `geolocator` (architecture §8.3). Không geocode — chỉ lat/lng (T3.1.1).
///
/// Web: nếu Permissions API thiếu, [Geolocator.checkPermission] có thể báo
/// denied — vẫn gọi [getCurrentPosition] để hiện dialog trình duyệt.
Future<LocationResult> requestLocationAndGetPosition() async {
  final serviceEnabled = await Geolocator.isLocationServiceEnabled();
  if (!serviceEnabled) {
    return LocationResult.fail(
      'Dịch vụ định vị đang tắt. Hãy bật GPS / Location Services rồi thử lại.',
    );
  }

  var permission = await Geolocator.checkPermission();
  if (permission == LocationPermission.denied) {
    permission = await Geolocator.requestPermission();
  }

  if (permission == LocationPermission.deniedForever) {
    return LocationResult.fail(
      'Quyền vị trí đã bị tắt vĩnh viễn. Mở Cài đặt ứng dụng / trình duyệt để bật lại quyền vị trí.',
    );
  }

  if (permission == LocationPermission.denied) {
    // Web browsers without Permissions API often report denied here;
    // still attempt getCurrentPosition to surface the browser prompt.
    if (!kIsWeb) {
      return LocationResult.fail(
        'Bạn đã từ chối quyền vị trí. Cho phép quyền vị trí để dùng vị trí hiện tại.',
      );
    }
  }

  try {
    final position = await Geolocator.getCurrentPosition(
      locationSettings: const LocationSettings(
        accuracy: LocationAccuracy.high,
        timeLimit: Duration(seconds: 20),
      ),
    );
    return LocationResult.ok(position);
  } on LocationServiceDisabledException {
    return LocationResult.fail(
      'Dịch vụ định vị đang tắt. Hãy bật GPS / Location Services rồi thử lại.',
    );
  } on PermissionDeniedException {
    return LocationResult.fail(
      'Bạn đã từ chối quyền vị trí. Cho phép quyền vị trí để dùng vị trí hiện tại.',
    );
  } on TimeoutException {
    return LocationResult.fail(
      'Không lấy được vị trí kịp thời. Thử lại khi tín hiệu GPS / mạng ổn định hơn.',
    );
  } catch (_) {
    return LocationResult.fail(
      'Không lấy được vị trí hiện tại. Thử lại hoặc chọn địa chỉ bằng cách khác.',
    );
  }
}
