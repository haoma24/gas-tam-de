package main

type stockLevelListResponse struct {
	Items []stockLevel `json:"items"`
	Count int          `json:"count"`
}
