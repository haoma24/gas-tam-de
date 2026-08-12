package main

import "testing"

// Every carrier prefix in service must keep working: rejecting a real customer
// at login is worse than accepting a junk number.
func TestNormalizePhoneVNAcceptsEveryCarrierPrefix(t *testing.T) {
	prefixes := []string{
		// Viettel
		"032", "033", "034", "035", "036", "037", "038", "039", "086", "096", "097", "098",
		// Vietnamobile / Reddi / Gmobile
		"052", "055", "056", "058", "059", "092", "099",
		// MobiFone
		"070", "076", "077", "078", "079", "089", "090", "093",
		// VinaPhone / Itelecom
		"081", "082", "083", "084", "085", "087", "088", "091", "094",
	}
	for _, p := range prefixes {
		local := p + "1234567"
		got, err := normalizePhoneVN(local)
		if err != nil {
			t.Errorf("normalizePhoneVN(%q) rejected a real prefix: %v", local, err)
			continue
		}
		want := "+84" + local[1:]
		if got != want {
			t.Errorf("normalizePhoneVN(%q)=%q want %q", local, got, want)
		}
	}
}

func TestNormalizePhoneVNAcceptsWritingVariants(t *testing.T) {
	const want = "+84901234567"
	for _, in := range []string{
		"0901234567",
		"090 123 4567",
		"090-123-4567",
		"090.123.4567",
		"  0901234567  ",
		"+84901234567",
		"+84 90 123 4567",
		"84901234567",
		"0084901234567",
	} {
		got, err := normalizePhoneVN(in)
		if err != nil {
			t.Errorf("normalizePhoneVN(%q): %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("normalizePhoneVN(%q)=%q want %q", in, got, want)
		}
	}
}

// These all passed the old `^0\d{9}$` rule. Each one reached the SMS gateway
// and cost a message before failing to deliver.
func TestNormalizePhoneVNRejectsNonMobileNumbers(t *testing.T) {
	cases := map[string]string{
		"0123456789":   "012 was retired in the 2018 renumbering",
		"0169123456 7": "0169 was retired too",
		"0000000000":   "placeholder digits",
		"0212345678":   "02x is a landline prefix",
		"0412345678":   "04x is not a mobile prefix",
		"0612345678":   "06x is not a mobile prefix",
		"0801234567":   "080 is a special-service range, not public mobile",
		"0501234567":   "050 is not assigned to mobile",
		"0751234567":   "075 is not assigned to mobile",
		"+84012345678": "E.164 for Vietnam never has a 0 after +84",
		"+84123456789": "same retired range in E.164 form",
		"84012345678":  "same, written without the plus",
	}
	for in, why := range cases {
		if got, err := normalizePhoneVN(in); err == nil {
			t.Errorf("normalizePhoneVN(%q)=%q, want rejected — %s", in, got, why)
		}
	}
}

func TestNormalizePhoneVNRejectsMalformedInput(t *testing.T) {
	for _, in := range []string{
		"",
		"123",
		"090123456",      // 9 digits
		"09012345678",    // 11 digits
		"02839221234",    // landline, 11 digits
		"0abc123456",
		"+849012345678",  // one digit too many
		"+1901234567",    // not Vietnam
	} {
		if got, err := normalizePhoneVN(in); err == nil {
			t.Errorf("normalizePhoneVN(%q)=%q, want rejected", in, got)
		}
	}
}

// The seeded admin allow-list must survive the stricter rule, otherwise the
// shop owner's phone stops getting role=admin from the customer OTP flow.
func TestNormalizePhoneVNAcceptsSeededAdminPhone(t *testing.T) {
	got, err := normalizePhoneVN(defaultAdminPhones)
	if err != nil {
		t.Fatalf("default ADMIN_PHONES %q is rejected: %v", defaultAdminPhones, err)
	}
	if got != "+84909777020" {
		t.Fatalf("normalizePhoneVN(%q)=%q", defaultAdminPhones, got)
	}
}
