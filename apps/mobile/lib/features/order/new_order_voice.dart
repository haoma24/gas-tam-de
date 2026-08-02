import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart';
import 'package:flutter_tts/flutter_tts.dart';

/// Loud spoken alert for admin Order Desk (Vietnamese).
class NewOrderVoice {
  NewOrderVoice._();

  static final FlutterTts _tts = FlutterTts();
  static bool _ready = false;

  static Future<void> _ensureReady() async {
    if (_ready) return;
    // Prefer Vietnamese; fall back if voice pack missing on the host.
    for (final lang in ['vi-VN', 'vi_VN', 'vi']) {
      try {
        final ok = await _tts.setLanguage(lang);
        if (ok == 1 || ok == true) break;
      } catch (_) {}
    }
    try {
      final voices = await _tts.getVoices;
      if (voices is List) {
        for (final v in voices) {
          if (v is Map) {
            final locale = '${v['locale'] ?? ''}'.toLowerCase();
            final name = '${v['name'] ?? ''}'.toLowerCase();
            if (locale.startsWith('vi') || name.contains('vietnam')) {
              await _tts.setVoice({
                'name': '${v['name']}',
                'locale': '${v['locale']}',
              });
              break;
            }
          }
        }
      }
    } catch (_) {}
    await _tts.setSpeechRate(kIsWeb ? 0.9 : 0.45);
    await _tts.setVolume(1.0);
    await _tts.setPitch(1.0);
    _ready = true;
  }

  /// Speaks pending-order summary in Vietnamese.
  static Future<void> announcePending(int count) async {
    if (count <= 0) return;
    try {
      SystemSound.play(SystemSoundType.alert);
    } catch (_) {}
    try {
      await _ensureReady();
      final text = count == 1
          ? 'Bạn có một đơn chưa giao'
          : 'Bạn có $count đơn chưa giao';
      await _tts.stop();
      await _tts.speak(text);
    } catch (e, st) {
      debugPrint('NewOrderVoice.announcePending failed: $e\n$st');
    }
  }

  /// Backward-compatible alias used when new orders arrive.
  static Future<void> announce({int count = 1}) => announcePending(count);
}
