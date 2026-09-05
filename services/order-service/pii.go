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

// adminOrderView builds the admin-facing order JSON. Same shape as
// [customerOrderView] plus the full contact number and the settlement fields —
// an admin who cannot phone the customer cannot deliver the order, and one who
// cannot reopen a finished order cannot answer a question about it.
func adminOrderView(o orderRow, items []orderItemView) orderView {
	v := customerOrderView(
		o.id, o.userID, o.customerName, o.phoneMasked, o.addressText,
		o.lat, o.lng, o.distanceKm,
		o.deliveryFee, o.subtotal, o.total,
		o.status, o.createdAt, items,
	)
	v.CustomerPhone = displayPhone(o.customerPhone)
	v.CompletedAt = o.completedAt
	v.CancelledAt = o.cancelledAt
	v.PaymentType = o.paymentType
	v.AmountPaid = o.amountPaid
	return v
}

// displayPhone renders a stored E.164 number the way a Vietnamese shop dials
// it: +84901234567 → 0901234567. Anything else is passed through untouched.
func displayPhone(raw string) string {
	s := strings.TrimSpace(raw)
	if strings.HasPrefix(s, "+84") && len(s) == 12 {
		return "0" + s[3:]
	}
	return s
}
