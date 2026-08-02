package main

import (
	"regexp"
	"strings"
)

var rePIIDigits = regexp.MustCompile(`\D+`)

// maskPhoneDisplay masks a VN phone for customer-facing API responses.
// Matches auth-service maskPhoneE164 style: 090***4567 (prefix 3 + *** + last 4).
func maskPhoneDisplay(raw string) string {
	s := strings.TrimSpace(raw)
	digits := rePIIDigits.ReplaceAllString(s, "")

	// +84xxxxxxxxx → local 0xxxxxxxxx then mask.
	if (strings.HasPrefix(s, "+84") || strings.HasPrefix(digits, "84")) && len(digits) == 11 {
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

// ensurePhoneMasked returns a display-safe phone. Already-masked values
// (containing "***") are re-normalized so accidental full numbers from
// X-Phone-Masked never leak into persist or JSON responses.
func ensurePhoneMasked(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	return maskPhoneDisplay(s)
}

// customerOrderView builds the customer-facing order JSON: phone always masked,
// never includes phone_hash / phone_e164 / raw phone.
func customerOrderView(
	id, userID, customerName, phoneMasked, addressText string,
	lat, lng, distanceKm float64,
	deliveryFee, subtotal, total int64,
	status, createdAt string,
	items []orderItemView,
) orderView {
	if items == nil {
		items = []orderItemView{}
	}
	return orderView{
		ID:           id,
		UserID:       userID,
		CustomerName: customerName,
		PhoneMasked:  ensurePhoneMasked(phoneMasked),
		AddressText:  addressText,
		Lat:          lat,
		Lng:          lng,
		DistanceKm:   distanceKm,
		DeliveryFee:  deliveryFee,
		Subtotal:     subtotal,
		Total:        total,
		Status:       status,
		CreatedAt:    createdAt,
		Items:        items,
	}
}
