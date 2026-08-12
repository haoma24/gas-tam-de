import 'package:flutter_test/flutter_test.dart';
import 'package:gas_tam_de/features/order/new_order_voice.dart';

/// Voice selection is what decided whether the Order Desk spoke Vietnamese or
/// fell back to the default English voice, so the ranking is pinned here.
/// Engines report locales inconsistently: `vi-VN` on Android, `vi_VN` on some
/// iOS builds, and browsers sometimes only hint the language in the name.

void main() {
  group('pickVietnameseVoice', () {
    test('prefers an exact vi-VN locale over other Vietnamese entries', () {
      final picked = NewOrderVoice.pickVietnameseVoice(const [
        TtsVoiceOption(name: 'Google US English', locale: 'en-US'),
        TtsVoiceOption(name: 'Some Voice', locale: 'vi'),
        TtsVoiceOption(name: 'Google Tiếng Việt', locale: 'vi-VN'),
      ]);
      expect(picked?.name, 'Google Tiếng Việt');
    });

    test('accepts an underscore locale (vi_VN)', () {
      final picked = NewOrderVoice.pickVietnameseVoice(const [
        TtsVoiceOption(name: 'Daniel', locale: 'en-GB'),
        TtsVoiceOption(name: 'Linh', locale: 'vi_VN'),
      ]);
      expect(picked?.name, 'Linh');
    });

    test('falls back to a voice whose name mentions Vietnamese', () {
      final picked = NewOrderVoice.pickVietnameseVoice(const [
        TtsVoiceOption(name: 'Microsoft An - Vietnamese (Vietnam)', locale: ''),
        TtsVoiceOption(name: 'Microsoft David', locale: 'en-US'),
      ]);
      expect(picked?.name, 'Microsoft An - Vietnamese (Vietnam)');
    });

    test('returns null when nothing is Vietnamese', () {
      final picked = NewOrderVoice.pickVietnameseVoice(const [
        TtsVoiceOption(name: 'Google US English', locale: 'en-US'),
        TtsVoiceOption(name: 'Kyoko', locale: 'ja-JP'),
      ]);
      expect(picked, isNull);
    });

    test('returns null for the empty list a browser reports before load', () {
      // The first speechSynthesis.getVoices() after page load is empty; that
      // must stay a "not found yet", never a silent English fallback.
      expect(NewOrderVoice.pickVietnameseVoice(const []), isNull);
      expect(NewOrderVoice.hasVietnameseVoice, isFalse);
    });
  });

  group('pendingText', () {
    test('reads one order in words', () {
      expect(NewOrderVoice.pendingText(1), 'Bạn có một đơn chưa giao');
    });

    test('reads the count for several orders', () {
      expect(NewOrderVoice.pendingText(4), 'Bạn có 4 đơn chưa giao');
    });
  });
}
