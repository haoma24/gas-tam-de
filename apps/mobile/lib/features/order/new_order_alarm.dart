import 'dart:math';
import 'dart:typed_data';

import 'package:audioplayers/audioplayers.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';

import '../../core/ui/ui.dart';

/// Alarm-clock tone + the modal that answers it, for pending orders on the
/// admin Order Desk.
///
/// This replaces the spoken announcement. Reading «Bạn có N đơn chưa giao»
/// needed a Vietnamese voice pack the shop's machines usually do not have, it
/// disappears under shop noise, and it says its piece exactly once — nobody who
/// stepped away ever learns an order arrived. An alarm keeps ringing until a
/// human answers it.

/// Snooze choices offered on the modal, in minutes.
const kAlarmSnoozeMinutes = <int>[5, 10, 15, 30];

/// One second of audio: two short chimes then silence, so the loop seam falls
/// in the quiet part and the tone reads as «ting ting … ting ting …».
@visibleForTesting
Uint8List buildAlarmWav({int sampleRate = 22050}) {
  final pcm = Int16List(sampleRate);

  void chime(double atSec, double lenSec, double freq) {
    final start = (atSec * sampleRate).round();
    final len = (lenSec * sampleRate).round();
    // 6 ms fade at both ends: a square-edged sine pops on every repeat.
    final fade = sampleRate * 0.006;
    for (var i = 0; i < len; i++) {
      final env = min(1.0, min(i, len - i) / fade);
      pcm[start + i] =
          (sin(2 * pi * freq * i / sampleRate) * 0.55 * env * 32767).round();
    }
  }

  chime(0.00, 0.13, 1046.5); // C6
  chime(0.20, 0.13, 1318.5); // E6
  return _wrapWav(pcm, sampleRate);
}

/// 16-bit mono PCM in a 44-byte WAV header — generating the tone beats
/// committing a binary asset nobody can review or retune.
Uint8List _wrapWav(Int16List pcm, int sampleRate) {
  final dataLen = pcm.lengthInBytes;
  final out = ByteData(44 + dataLen);

  void tag(int at, String s) {
    for (var i = 0; i < s.length; i++) {
      out.setUint8(at + i, s.codeUnitAt(i));
    }
  }

  tag(0, 'RIFF');
  out.setUint32(4, 36 + dataLen, Endian.little);
  tag(8, 'WAVE');
  tag(12, 'fmt ');
  out.setUint32(16, 16, Endian.little); // PCM header size
  out.setUint16(20, 1, Endian.little); // format = PCM
  out.setUint16(22, 1, Endian.little); // mono
  out.setUint32(24, sampleRate, Endian.little);
  out.setUint32(28, sampleRate * 2, Endian.little); // byte rate
  out.setUint16(32, 2, Endian.little); // block align
  out.setUint16(34, 16, Endian.little); // bits per sample
  tag(36, 'data');
  out.setUint32(40, dataLen, Endian.little);
  for (var i = 0; i < pcm.length; i++) {
    out.setInt16(44 + i * 2, pcm[i], Endian.little);
  }
  return out.buffer.asUint8List();
}

/// Looping alarm sound. Web browsers block audio until the page has been
/// interacted with; the desk is reached by tapping through a login, so by the
/// time an order lands the gesture requirement is already satisfied.
class NewOrderAlarm {
  NewOrderAlarm._();

  static AudioPlayer? _player;
  static Uint8List? _tone;

  static Future<void> start() async {
    try {
      final player = _player ??= AudioPlayer();
      await player.setReleaseMode(ReleaseMode.loop);
      await player.stop();
      await player.play(BytesSource(_tone ??= buildAlarmWav()));
    } catch (e, st) {
      debugPrint('NewOrderAlarm.start failed: $e\n$st');
    }
  }

  static Future<void> stop() async {
    try {
      await _player?.stop();
    } catch (e, st) {
      debugPrint('NewOrderAlarm.stop failed: $e\n$st');
    }
  }

  /// Short preview for the settings screen.
  static Future<void> preview() async {
    await start();
    await Future<void>.delayed(const Duration(milliseconds: 2500));
    await stop();
  }
}

/// The modal that answers the alarm.
///
/// Returns the chosen snooze, or `null` for «Không hiển thị lại». There is no
/// third outcome: the barrier and the back button are both blocked, so the
/// alarm cannot be dismissed by accident while orders are still waiting.
Future<Duration?> showNewOrderAlarmDialog(
  BuildContext context, {
  required int pending,
}) {
  return showDialog<Duration>(
    context: context,
    barrierDismissible: false,
    builder: (ctx) => PopScope(
      canPop: false,
      child: AlertDialog(
        icon: Icon(
          Icons.notifications_active_rounded,
          color: ctx.palette.primary,
          size: 32,
        ),
        title: Text(
          pending == 1 ? 'Có 1 đơn chưa giao' : 'Có $pending đơn chưa giao',
        ),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const Text('Nhấn «Báo lại» để tạm tắt chuông, hoặc tắt hẳn '
                'chuông cho phiên làm việc này.'),
            const VGap(AppSpacing.lg),
            Text(
              'BÁO LẠI SAU',
              style: ctx.text.labelSmall?.copyWith(
                color: ctx.palette.inkMuted,
                letterSpacing: 0.6,
              ),
            ),
            const VGap(AppSpacing.sm),
            Wrap(
              spacing: AppSpacing.sm,
              runSpacing: AppSpacing.sm,
              children: [
                for (final m in kAlarmSnoozeMinutes)
                  OutlinedButton(
                    onPressed: () =>
                        Navigator.of(ctx).pop(Duration(minutes: m)),
                    child: Text('$m phút'),
                  ),
              ],
            ),
          ],
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(),
            child: const Text('Không hiển thị lại'),
          ),
        ],
      ),
    ),
  );
}
