/// Pure formatting helpers — no Flutter imports, so both models and widgets
/// can depend on it.
library;

/// Formats VND integer with thousand separators (e.g. `450000` → `450.000 ₫`).
String formatVnd(int amount) {
  final negative = amount < 0;
  final digits = amount.abs().toString();
  final buf = StringBuffer();
  for (var i = 0; i < digits.length; i++) {
    if (i > 0 && (digits.length - i) % 3 == 0) {
      buf.write('.');
    }
    buf.write(digits[i]);
  }
  return '${negative ? '-' : ''}${buf.toString()} ₫';
}
