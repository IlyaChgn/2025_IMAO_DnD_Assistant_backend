package models

type DescriptionGenerationRequest struct {
	FirstCharID  string `json:"first_char_id"`
	SecondCharID string `json:"second_char_id"`
}

type DescriptionGenerationResponse struct {
	BattleDescription string `json:"battle_description"`
}
