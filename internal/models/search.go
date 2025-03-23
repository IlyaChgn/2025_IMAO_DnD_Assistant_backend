package models

type SearchParams struct {
	Value string `json:"value"`
	Exact bool   `json:"exact"`
}

type Order struct {
	Field     string `json:"field"`
	Direction string `json:"direction"`
}
