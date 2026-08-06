/// Client-side Vietnam mobile helpers (mirror auth-service `normalizePhoneVN`).
library;

final _digitsOnly = RegExp(r'\D+');

/// Accepts `09xxxxxxxx`, `+849xxxxxxxx`, `849xxxxxxxx`, spaces/dashes allowed.
bool isValidVnMobile(String raw) {
  final s = raw.trim().replaceAll(' ', '').replaceAll('-', '').replaceAll('.', '');
  final digits = s.replaceAll(_digitsOnly, '');

  if (RegExp(r'^0\d{9}$').hasMatch(digits)) return true;
  if (RegExp(r'^84\d{9}$').hasMatch(digits)) return true;
  if (s.startsWith('+84') && RegExp(r'^84\d{9}$').hasMatch(digits)) return true;
  return false;
}

/// Best-effort local mask for UI before API returns `phone_masked`.
String maskVnPhone(String raw) {
  final digits = raw.replaceAll(_digitsOnly, '');
  String local;
  if (digits.length == 10 && digits.startsWith('0')) {
    local = digits;
  } else if (digits.length == 11 && digits.startsWith('84')) {
    local = '0${digits.substring(2)}';
  } else {
    return '***';
  }
  if (local.length != 10) return '***';
  return '${local.substring(0, 3)}***${local.substring(local.length - 4)}';
}
