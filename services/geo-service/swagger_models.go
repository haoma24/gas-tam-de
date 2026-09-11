package main

type publicStoreResponse struct {
	Name        string  `json:"name"`
	Lat         float64 `json:"lat"`
	Lng         float64 `json:"lng"`
	MaxRadiusKm float64 `json:"max_radius_km"`
	AddressText string  `json:"address_text,omitempty"`
}

type geoSearchResponse struct {
	Items  []geoPlace `json:"items"`
	Cached bool       `json:"cached"`
}
