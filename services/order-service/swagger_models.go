package main

// Named envelopes for responses that handlers assemble with maps.
type orderListResponse struct {
	Orders []orderView `json:"orders"`
}

type adminOrderListResponse struct {
	Orders []orderView `json:"orders"`
	Status string      `json:"status"`
	Count  int         `json:"count"`
}

type cancelOrderResponse struct {
	OK          bool   `json:"ok"`
	OrderID     string `json:"order_id"`
	Status      string `json:"status"`
	CancelledAt string `json:"cancelled_at"`
}

type orderDefaultsResponse struct {
	HasDefaults  bool    `json:"has_defaults"`
	CustomerName string  `json:"customer_name,omitempty"`
	AddressText  string  `json:"address_text,omitempty"`
	Lat          float64 `json:"lat,omitempty"`
	Lng          float64 `json:"lng,omitempty"`
	OrderedAt    string  `json:"ordered_at,omitempty"`
}

type customerStatsResponse struct {
	From      string         `json:"from"`
	To        string         `json:"to"`
	Timezone  string         `json:"timezone"`
	Count     int            `json:"count"`
	Total     int            `json:"total"`
	Customers []customerStat `json:"customers"`
}
