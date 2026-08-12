import 'package:flutter_test/flutter_test.dart';
import 'package:gas_tam_de/features/auth/phone_utils.dart';

/// Mirrors `services/auth-service/phone_test.go`. The client rule must not be
/// looser than the server's, or the user gets a rejected request instead of an
/// inline hint; and not stricter, or a real customer cannot log in at all.

void main() {
  group('isValidVnMobile accepts', () {
    test('every carrier prefix in service', () {
      const prefixes = [
        // Viettel
        '032', '033', '034', '035', '036', '037', '038', '039',
        '086', '096', '097', '098',
        // Vietnamobile / Reddi / Gmobile
        '052', '055', '056', '058', '059', '092', '099',
        // MobiFone
        '070', '076', '077', '078', '079', '089', '090', '093',
        // VinaPhone / Itelecom
        '081', '082', '083', '084', '085', '087', '088', '091', '094',
      ];
      for (final p in prefixes) {
        expect(isValidVnMobile('${p}1234567'), isTrue, reason: 'prefix $p');
      }
    });

    test('the ways people write a number', () {
      for (final input in [
        '0901234567',
        '090 123 4567',
        '090-123-4567',
        '090.123.4567',
        '  0901234567  ',
        '+84901234567',
        '+84 90 123 4567',
        '84901234567',
      ]) {
        expect(isValidVnMobile(input), isTrue, reason: input);
      }
    });
  });

  group('isValidVnMobile rejects', () {
    test('numbers that are not mobile lines', () {
      const cases = {
        '0123456789': '012 retired in the 2018 renumbering',
        '0000000000': 'placeholder digits',
        '0212345678': '02x is a landline prefix',
        '0412345678': '04x is not a mobile prefix',
        '0612345678': '06x is not a mobile prefix',
        '0801234567': '080 is a special-service range',
        '+84012345678': 'E.164 for Vietnam never has a 0 after +84',
        '84012345678': 'same, without the plus',
      };
      cases.forEach((input, why) {
        expect(isValidVnMobile(input), isFalse, reason: '$input — $why');
      });
    });

    test('malformed input', () {
      for (final input in [
        '',
        '123',
        '090123456', // 9 digits
        '09012345678', // 11 digits
        '02839221234', // landline
        '0abc123456',
        '+1901234567', // not Vietnam
      ]) {
        expect(isValidVnMobile(input), isFalse, reason: input);
      }
    });
  });

  group('maskVnPhone', () {
    test('keeps the first three and last four digits', () {
      expect(maskVnPhone('0901234567'), '090***4567');
      expect(maskVnPhone('+84321234567'), '032***4567');
    });

    test('refuses to guess at anything else', () {
      expect(maskVnPhone('123'), '***');
    });
  });
}
