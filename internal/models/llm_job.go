package models

import "time"

// LLMJob — асинхронная задача генерации Creature
type LLMJob struct {
	ID string `db:"id"`
	// одно из двух:
	Description *string   `db:"description,omitempty"` // если пришёл текст
	Image       []byte    `db:"image,omitempty"`       // если пришла картинка
	Status      string    `db:"status"`                // "pending"|"processing"|"done"|"error"
	Result      *Creature `db:"result,omitempty"`      // готовый Creature
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}
