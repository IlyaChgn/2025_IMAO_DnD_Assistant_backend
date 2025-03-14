package delivery

import (
	"net/http"

	"github.com/gorilla/mux"

	creatureinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/creature"
	responses "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/delivery/responses"
)

// CreatureHandler обрабатывает HTTP-запросы, связанные с существами
type CreatureHandler struct {
	creatureUsecases creatureinterfaces.CreatureUsecases
}

// NewCreatureHandler создает новый экземпляр CreatureHandler
func NewCreatureHandler(creatureUsecases creatureinterfaces.CreatureUsecases) *CreatureHandler {
	return &CreatureHandler{creatureUsecases: creatureUsecases}
}

// GetCreatureByName обрабатывает HTTP-запрос для получения существа по имени
func (h *CreatureHandler) GetCreatureByName(w http.ResponseWriter, r *http.Request) {
	// Извлекаем параметр имени из URL
	vars := mux.Vars(r)
	creatureName := vars["name"]

	// Вызываем usecase для получения существа
	creature, err := h.creatureUsecases.GetCreatureByEngName(r.Context(), creatureName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	responses.SendOkResponse(w, creature)
}
