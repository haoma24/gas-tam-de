package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
)

var (
	reVNLocal = regexp.MustCompile(`^0\d{9}$`)
	reE164VN  = regexp.MustCompile(`^\+84\d{9}$`)
	reDigits  = regexp.MustCompile(`\D+`)
)

// normalizePhoneVN accepts common VN forms and returns E.164 (+84…).
func normalizePhoneVN(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "-", "")
	s = strings.ReplaceAll(s, ".", "")

	if strings.HasPrefix(s, "84") && !strings.HasPrefix(s, "+") && len(reDigits.ReplaceAllString(s, "")) == 11 {
		s = "+" + reDigits.ReplaceAllString(s, "")
	}
	if strings.HasPrefix(s, "0084") {
		s = "+" + strings.TrimPrefix(reDigits.ReplaceAllString(s, ""), "00")
	}

	digits := reDigits.ReplaceAllString(s, "")
	switch {
	case reVNLocal.MatchString(digits):
		return "+84" + digits[1:], nil
	case strings.HasPrefix(s, "+84") && reE164VN.MatchString("+"+digits):
		return "+" + digits, nil
	case reE164VN.MatchString(s):
		return s, nil
	default:
		return "", fmt.Errorf("invalid vietnamese phone")
	}
}

func maskPhoneE164(e164 string) string {
	digits := reDigits.ReplaceAllString(e164, "")
	if strings.HasPrefix(e164, "+84") && len(digits) == 11 {
		local := "0" + digits[2:]
		if len(local) == 10 {
			return local[:3] + "***" + local[len(local)-4:]
		}
	}
	if len(digits) < 7 {
		return "***"
	}
	return digits[:3] + "***" + digits[len(digits)-4:]
}

func hashPhone(e164, pepper string) string {
	mac := hmac.New(sha256.New, []byte(pepper))
	_, _ = mac.Write([]byte(e164))
	return hex.EncodeToString(mac.Sum(nil))
}
