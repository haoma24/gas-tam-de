package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"gas-tam-de/pkg/httpx"
)

type quoteOrderItemBody struct {
	ProductID string `json:"product_id"`
	Qty       int    `json:"qty"`
}

type quoteOrderBody struct {
	Lat   float64              `json:"lat"`
	Lng   float64              `json:"lng"`
	Items []quoteOrderItemBody `json:"items"`
}

type quoteOrderView struct {
	DistanceKm  float64 `json:"distance_km"`
	InRange     bool    `json:"in_range"`
	MaxRadiusKm float64 `json:"max_radius_km"`
	DeliveryFee int64   `json:"delivery_fee"`
	Subtotal    int64   `json:"subtotal"`
	Total       int64   `json:"total"`
}

// handleQuoteOrder serves POST /v1/orders/quote — preview distance, fee, and totals
// without persisting. Returns in_range so clients can block place when out of radius.
func (s *orderService) handleQuoteOrder(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := requireCustomerIdentity(w, r); !ok {
		return
	}

	var body quoteOrderBody
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_JSON", "invalid request body")
		return
	}

	if err := validateCoords(body.Lat, body.Lng); err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_COORDS", err.Error())
		return
	}
	if len(body.Items) == 0 {
		httpx.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "items must not be empty")
		return
	}

	qtyByProduct := make(map[string]int, len(body.Items))
	orderedIDs := make([]string, 0, len(body.Items))
	for _, it := range body.Items {
		pid := strings.TrimSpace(it.ProductID)
		if pid == "" {
			httpx.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "items[].product_id is required")
			return
		}
		if it.Qty < 1 {
			httpx.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "items[].qty must be >= 1")
			return
		}
		if _, seen := qtyByProduct[pid]; !seen {
			orderedIDs = append(orderedIDs, pid)
		}
		qtyByProduct[pid] += it.Qty
	}

	geo, err := s.geo.Check(r.Context(), body.Lat, body.Lng)
	if err != nil {
		slog.Error("geo check", "err", err)
		httpx.Error(w, http.StatusBadGateway, "GEO_UNAVAILABLE", "could not verify delivery range")
		return
	}

	products, err := s.catalog.ListActive(r.Context())
	if err != nil {
		slog.Error("catalog list", "err", err)
		httpx.Error(w, http.StatusBadGateway, "CATALOG_UNAVAILABLE", "could not load products")
		return
	}
	byID := make(map[string]catalogProduct, len(products))
	for _, p := range products {
		byID[p.ID] = p
	}

	var subtotal int64
	for _, pid := range orderedIDs {
		qty := qtyByProduct[pid]
		p, found := byID[pid]
		if !found || !p.Active {
			httpx.Error(w, http.StatusBadRequest, "PRODUCT_NOT_FOUND", "product not found or inactive: "+pid)
			return
		}
		if p.SalePrice < 0 {
			httpx.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid product price for "+pid)
			return
		}
		subtotal += p.SalePrice * int64(qty)
	}

	deliveryFee, err := computeDeliveryFee(s.db, geo.DistanceKm)
	if err != nil {
		slog.Error("compute delivery fee", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not compute delivery fee")
		return
	}

	httpx.JSON(w, http.StatusOK, quoteOrderView{
		DistanceKm:  geo.DistanceKm,
		InRange:     geo.InRange,
		MaxRadiusKm: geo.MaxRadiusKm,
		DeliveryFee: deliveryFee,
		Subtotal:    subtotal,
		Total:       subtotal + deliveryFee,
	})
}
