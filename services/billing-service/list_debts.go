package main

import (
	"log/slog"
	"net/http"

	"gas-tam-de/pkg/httpx"
)

// debtItem is one customer debt row for admin list (T6.2.1).
type debtItem struct {
	CustomerKey string `json:"customer_key"`
	PhoneMasked string `json:"phone_masked"`
	Balance     int64  `json:"balance"`
	UpdatedAt   string `json:"updated_at"`
}

// listDebtsResponse is GET /v1/admin/debts — per-customer balances + aggregate.
type listDebtsResponse struct {
	Items        []debtItem `json:"items"`
	TotalBalance int64      `json:"total_balance"`
	Count        int        `json:"count"`
}

// handleListDebts serves GET /v1/admin/debts — customers with outstanding balance
// (balance > 0), highest first. Aggregate total_balance = SUM(balance).
// Gateway mounts under /v1/admin/* with role=admin RBAC.
func (s *billingService) handleListDebts(w http.ResponseWriter, _ *http.Request) {
	out, err := s.listDebts()
	if err != nil {
		slog.Error("list debts", "err", err)
		httpx.Error(w, http.StatusInternalServerError, "INTERNAL", "could not list debts")
		return
	}
	httpx.JSON(w, http.StatusOK, out)
}

func (s *billingService) listDebts() (listDebtsResponse, error) {
	rows, err := s.db.Query(`
		SELECT customer_key, phone_masked, balance, updated_at
		FROM debts
		WHERE balance > 0
		ORDER BY balance DESC, updated_at DESC`)
	if err != nil {
		return listDebtsResponse{}, err
	}
	defer rows.Close()

	items := make([]debtItem, 0)
	var total int64
	for rows.Next() {
		var d debtItem
		if err := rows.Scan(&d.CustomerKey, &d.PhoneMasked, &d.Balance, &d.UpdatedAt); err != nil {
			return listDebtsResponse{}, err
		}
		items = append(items, d)
		total += d.Balance
	}
	if err := rows.Err(); err != nil {
		return listDebtsResponse{}, err
	}

	return listDebtsResponse{
		Items:        items,
		TotalBalance: total,
		Count:        len(items),
	}, nil
}
