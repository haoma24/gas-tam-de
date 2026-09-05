import 'dart:typed_data';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gas_tam_de/core/ui/ui.dart';
import 'package:gas_tam_de/features/order/new_order_alarm.dart';

String _tagAt(Uint8List b, int at) =>
    String.fromCharCodes(b.sublist(at, at + 4));

void main() {
  group('buildAlarmWav', () {
    final wav = buildAlarmWav(sampleRate: 8000);
    final view = ByteData.sublistView(wav);

    test('is a 16-bit mono PCM WAV of exactly one second', () {
      expect(_tagAt(wav, 0), 'RIFF');
      expect(_tagAt(wav, 8), 'WAVE');
      expect(_tagAt(wav, 12), 'fmt ');
      expect(_tagAt(wav, 36), 'data');
      expect(view.getUint16(20, Endian.little), 1, reason: 'PCM');
      expect(view.getUint16(22, Endian.little), 1, reason: 'mono');
      expect(view.getUint32(24, Endian.little), 8000);
      expect(view.getUint16(34, Endian.little), 16, reason: 'bits per sample');
      // 8000 samples x 2 bytes + a 44-byte header; the declared sizes must
      // agree with the buffer or players read past the end and click.
      expect(wav.length, 44 + 8000 * 2);
      expect(view.getUint32(40, Endian.little), 8000 * 2);
      expect(view.getUint32(4, Endian.little), wav.length - 8);
    });

    test('rings twice and leaves the loop seam silent', () {
      int peak(double fromSec, double toSec) {
        var max = 0;
        for (var i = (fromSec * 8000).round();
            i < (toSec * 8000).round();
            i++) {
          final v = view.getInt16(44 + i * 2, Endian.little).abs();
          if (v > max) max = v;
        }
        return max;
      }

      expect(peak(0.01, 0.12), greaterThan(8000), reason: 'first chime');
      expect(peak(0.21, 0.32), greaterThan(8000), reason: 'second chime');
      expect(peak(0.15, 0.19), 0, reason: 'gap between the two chimes');
      // The tail is what the loop repeats into — audio there would make the
      // seam pop on every cycle.
      expect(peak(0.40, 1.00), 0, reason: 'tail before the loop repeats');
    });
  });

  group('showNewOrderAlarmDialog', () {
    Future<Duration?> open(WidgetTester tester, String tapLabel) async {
      Duration? result;
      var returned = false;
      await tester.pumpWidget(
        MaterialApp(
          theme: buildAppTheme(Brightness.light),
          home: Builder(
            builder: (context) => TextButton(
              onPressed: () async {
                result = await showNewOrderAlarmDialog(context, pending: 3);
                returned = true;
              },
              child: const Text('open'),
            ),
          ),
        ),
      );
      await tester.tap(find.text('open'));
      await tester.pumpAndSettle();
      expect(find.text('Có 3 đơn chưa giao'), findsOneWidget);

      await tester.tap(find.text(tapLabel));
      await tester.pumpAndSettle();
      expect(returned, isTrue);
      return result;
    }

    testWidgets('a snooze choice returns its duration', (tester) async {
      expect(await open(tester, '10 phút'), const Duration(minutes: 10));
    });

    testWidgets('«Không hiển thị lại» returns null', (tester) async {
      expect(await open(tester, 'Không hiển thị lại'), isNull);
    });

    testWidgets('the barrier cannot dismiss it', (tester) async {
      await tester.pumpWidget(
        MaterialApp(
          theme: buildAppTheme(Brightness.light),
          home: Builder(
            builder: (context) => TextButton(
              onPressed: () => showNewOrderAlarmDialog(context, pending: 1),
              child: const Text('open'),
            ),
          ),
        ),
      );
      await tester.tap(find.text('open'));
      await tester.pumpAndSettle();
      // Tapping the scrim must not silence an alarm that is still ringing.
      await tester.tapAt(const Offset(10, 10));
      await tester.pumpAndSettle();
      expect(find.text('Có 1 đơn chưa giao'), findsOneWidget);
    });
  });
}
