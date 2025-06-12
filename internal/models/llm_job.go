package models

import "time"

// LLMJob — асинхронная задача генерации Creature
type LLMJob struct {
	ID string `db:"id"`
	// одно из двух:
	Description *string   `db:"description,omitempty"` // если пришёл текст
	Image       []byte    `db:"image,omitempty"`       // если пришла картинка
	Status      string    `db:"status"`                // "pending"|"processing_step_1"|"processing_step_2"|"done"|"error"
	Result      *Creature `db:"result,omitempty"`      // готовый Creature
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}

type DescriptionGenPrompt struct {
	Description string `json:"description"`
}

type LLMJobResponse struct {
	JobID string `json:"job_id"`
}

type LLMJobStatusResponse struct {
	Status string    `json:"status"`
	Result *Creature `json:"result"`
}
