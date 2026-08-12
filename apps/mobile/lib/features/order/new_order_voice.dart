import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart';
import 'package:flutter_tts/flutter_tts.dart';

/// One candidate voice reported by the TTS engine.
@immutable
class TtsVoiceOption {
  const TtsVoiceOption({required this.name, required this.locale});

  final String name;
  final String locale;

  String get label => locale.isEmpty ? name : '$name ($locale)';
}

/// Loud spoken alert for admin Order Desk (Vietnamese).
///
/// Picking the voice is the whole problem here. Browsers populate
/// `speechSynthesis.getVoices()` **asynchronously**: the first call right after
/// page load returns an empty list, so a one-shot lookup finds no Vietnamese
/// voice, and the desk then reads Vietnamese text with the default English
/// voice. Discovery therefore polls, and a failed lookup is never cached as
/// final — it is retried on the next announcement.
class NewOrderVoice {
  NewOrderVoice._();

  static final FlutterTts _tts = FlutterTts();

  /// Rate/volume/pitch only need applying once.
  static bool _baseApplied = false;

  /// Set once a genuinely Vietnamese voice is active.
  static bool _vietnameseVoiceSet = false;

  static TtsVoiceOption? _selectedVoice;

  /// Polling budget for the first lookup (~2s), enough for Chrome to publish
  /// its network voices without making the first alert feel late.
  static const int _discoveryAttempts = 10;
  static const Duration _discoveryDelay = Duration(milliseconds: 200);

  /// True when the alert is really speaking Vietnamese. False means the host
  /// has no Vietnamese voice installed — no app-side setting can fix that.
  static bool get hasVietnameseVoice => _vietnameseVoiceSet;

  /// Voice currently in use, for showing in settings.
  static TtsVoiceOption? get selectedVoice => _selectedVoice;

  /// Warms up voice discovery so the first real alert already speaks
  /// Vietnamese. Call when the Order Desk opens; safe to call repeatedly.
  static Future<void> prewarm() async {
    try {
      await _applyBaseSettings();
      await _applyVietnameseLanguage();
      await _selectVietnameseVoice(attempts: _discoveryAttempts);
    } catch (e, st) {
      debugPrint('NewOrderVoice.prewarm failed: $e\n$st');
    }
  }

  static Future<void> _applyBaseSettings() async {
    if (_baseApplied) return;
    await _tts.setSpeechRate(kIsWeb ? 0.9 : 0.45);
    await _tts.setVolume(1.0);
    await _tts.setPitch(1.0);
    _baseApplied = true;
  }

  /// Sets the utterance language even when no Vietnamese voice is listed —
  /// some engines still pick a matching voice from the language alone.
  static Future<void> _applyVietnameseLanguage() async {
    for (final lang in const ['vi-VN', 'vi_VN', 'vi']) {
      try {
        final ok = await _tts.setLanguage(lang);
        if (ok == 1 || ok == true) return;
      } catch (_) {}
    }
  }

  /// Looks for a Vietnamese voice and activates the best match.
  /// [attempts] > 1 polls, because the browser voice list arrives late.
  static Future<void> _selectVietnameseVoice({int attempts = 1}) async {
    if (_vietnameseVoiceSet) return;

    for (var i = 0; i < attempts; i++) {
      final voice = _pickVietnamese(await _listVoices());
      if (voice != null) {
        try {
          await _tts.setVoice({'name': voice.name, 'locale': voice.locale});
          _vietnameseVoiceSet = true;
          _selectedVoice = voice;
          debugPrint('NewOrderVoice: using ${voice.label}');
        } catch (e) {
          debugPrint('NewOrderVoice: setVoice ${voice.label} failed: $e');
        }
        return;
      }
      if (i < attempts - 1) {
        await Future<void>.delayed(_discoveryDelay);
      }
    }
  }

  static Future<List<TtsVoiceOption>> _listVoices() async {
    try {
      final raw = await _tts.getVoices;
      if (raw is! List) return const [];
      final out = <TtsVoiceOption>[];
      for (final v in raw) {
        if (v is Map) {
          out.add(TtsVoiceOption(
            name: '${v['name'] ?? ''}',
            locale: '${v['locale'] ?? ''}',
          ));
        }
      }
      return out;
    } catch (_) {
      return const [];
    }
  }

  /// Exact `vi-VN` wins over any `vi*`, which wins over a name that merely
  /// mentions Vietnamese — engines label voices inconsistently across
  /// Android, iOS and the browsers.
  @visibleForTesting
  static TtsVoiceOption? pickVietnameseVoice(List<TtsVoiceOption> voices) =>
      _pickVietnamese(voices);

  static TtsVoiceOption? _pickVietnamese(List<TtsVoiceOption> voices) {
    TtsVoiceOption? byLocalePrefix;
    TtsVoiceOption? byName;

    for (final v in voices) {
      final locale = v.locale.toLowerCase().replaceAll('_', '-');
      final name = v.name.toLowerCase();

      if (locale == 'vi-vn') return v;
      if (byLocalePrefix == null && (locale == 'vi' || locale.startsWith('vi-'))) {
        byLocalePrefix = v;
      }
      if (byName == null &&
          (name.contains('vietnam') ||
              name.contains('việt') ||
              name.contains('viet'))) {
        byName = v;
      }
    }
    return byLocalePrefix ?? byName;
  }

  /// Speaks pending-order summary in Vietnamese.
  static Future<void> announcePending(int count) async {
    if (count <= 0) return;
    try {
      SystemSound.play(SystemSoundType.alert);
    } catch (_) {}
    try {
      await _applyBaseSettings();
      await _applyVietnameseLanguage();
      // Cheap re-check: a voice pack (or a browser's network voices) can show
      // up long after the first attempt, so never give up permanently.
      await _selectVietnameseVoice();
      await _tts.stop();
      await _tts.speak(pendingText(count));
    } catch (e, st) {
      debugPrint('NewOrderVoice.announcePending failed: $e\n$st');
    }
  }

  @visibleForTesting
  static String pendingText(int count) =>
      count == 1 ? 'Bạn có một đơn chưa giao' : 'Bạn có $count đơn chưa giao';

  /// Speaks a sample so the shop can check the voice from settings.
  /// Returns the active voice, or null when no Vietnamese voice exists.
  static Future<TtsVoiceOption?> speakSample() async {
    try {
      await _applyBaseSettings();
      await _applyVietnameseLanguage();
      await _selectVietnameseVoice(attempts: _discoveryAttempts);
      await _tts.stop();
      await _tts.speak(pendingText(2));
    } catch (e, st) {
      debugPrint('NewOrderVoice.speakSample failed: $e\n$st');
    }
    return _vietnameseVoiceSet ? _selectedVoice : null;
  }

  /// Backward-compatible alias used when new orders arrive.
  static Future<void> announce({int count = 1}) => announcePending(count);
}
