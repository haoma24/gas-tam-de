/// Client-side Vietnam mobile helpers (mirror auth-service `normalizePhoneVN`).

final _digitsOnly = RegExp(r'\D+');

/// Vietnamese mobile prefixes since the 2018 renumbering, without the leading 0.
/// Keep in sync with `vnMobilePrefix` in auth-service `phone.go` — the server
/// rejects anything else, so a looser rule here only produces a failed request.
///
///   3[2-9]    Viettel          032-039
///   5[25689]  Vietnamobile 052/056/058, Reddi 055, Gmobile 059
///   7[06-9]   MobiFone         070, 076-079
///   8[1-9]    VinaPhone 081-085/088, Viettel 086, Itelecom 087, MobiFone 089
///   9[0-9]    all carriers     090-099
const _vnMobilePrefix = r'(3[2-9]|5[25689]|7[06-9]|8[1-9]|9[0-9])';

final _reVnLocal = RegExp('^0$_vnMobilePrefix' r'\d{7}$');
final _reVnIntl = RegExp('^84$_vnMobilePrefix' r'\d{7}$');

/// Accepts `09xxxxxxxx`, `+849xxxxxxxx`, `849xxxxxxxx`, spaces/dashes allowed.
/// Only real mobile prefixes pass: retired ranges (`012…`), landline prefixes
/// (`02…`) and placeholders like `0000000000` are rejected before they cost an
/// SMS.
bool isValidVnMobile(String raw) {
  final s = raw.trim().replaceAll(' ', '').replaceAll('-', '').replaceAll('.', '');
  final digits = s.replaceAll(_digitsOnly, '');

  if (_reVnLocal.hasMatch(digits)) return true;
  if (_reVnIntl.hasMatch(digits)) return true;
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
