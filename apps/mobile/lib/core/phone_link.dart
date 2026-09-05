import 'package:url_launcher/url_launcher.dart';

/// Kết quả mở trình quay số — cùng khuôn với `openNavigationTo`.
class PhoneLaunchResult {
  const PhoneLaunchResult._({this.errorMessage});

  factory PhoneLaunchResult.ok() => const PhoneLaunchResult._();

  factory PhoneLaunchResult.fail(String message) =>
      PhoneLaunchResult._(errorMessage: message);

  final String? errorMessage;

  bool get isOk => errorMessage == null;
}

/// Chuẩn hoá số để đưa vào `tel:` — bỏ mọi ký tự không phải chữ số / dấu `+`.
///
/// Trả về chuỗi rỗng cho số đã bị che (`090***4567`) hoặc số quá ngắn: không
/// có gì để gọi thì thà tắt nút còn hơn mở trình quay số với số sai.
String telDigits(String raw) {
  final s = raw.trim();
  if (s.contains('*')) return '';
  final buf = StringBuffer();
  for (var i = 0; i < s.length; i++) {
    final c = s[i];
    if (c == '+' && i == 0) {
      buf.write(c);
    } else if (c.codeUnitAt(0) >= 0x30 && c.codeUnitAt(0) <= 0x39) {
      buf.write(c);
    }
  }
  final digits = buf.toString();
  return digits.replaceAll('+', '').length < 8 ? '' : digits;
}

/// URI `tel:` cho [phone]; `null` khi số không gọi được.
Uri? telUri(String phone) {
  final digits = telDigits(phone);
  if (digits.isEmpty) return null;
  return Uri(scheme: 'tel', path: digits);
}

/// Mở trình quay số của máy cho [phone].
///
/// Dùng ở Order Desk: admin phải gọi được cho khách ngay từ đơn hàng.
Future<PhoneLaunchResult> dialPhone(String phone) async {
  final uri = telUri(phone);
  if (uri == null) {
    return PhoneLaunchResult.fail('Đơn này chưa có số điện thoại để gọi.');
  }
  try {
    final ok = await launchUrl(uri, mode: LaunchMode.externalApplication);
    if (ok) return PhoneLaunchResult.ok();
  } catch (_) {
    // Rơi xuống thông báo bên dưới.
  }
  return PhoneLaunchResult.fail('Không mở được trình gọi. Hãy bấm giữ để chép số.');
}
