package delivery

import (
	"encoding/json"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
	"net/http"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	descriptioninterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/description"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/delivery/responses"
)

type DescriptionHandler struct {
	descriptionUseCase descriptioninterfaces.DescriptionUsecases
}

func NewDescriptionHandler(descriptionUseCase descriptioninterfaces.DescriptionUsecases) *DescriptionHandler {
	return &DescriptionHandler{
		descriptionUseCase: descriptionUseCase,
	}
}

func (h *DescriptionHandler) GenerateDescription(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := logger.FromContext(ctx)

	var reqData models.DescriptionGenerationRequest

	err := json.NewDecoder(r.Body).Decode(&reqData)
	if err != nil {
		l.DeliveryError(ctx, responses.StatusBadRequest, responses.ErrBadJSON, nil, nil)
		responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrBadJSON)

		return
	}

	resp, err := h.descriptionUseCase.GenerateDescription(ctx, reqData)
	if err != nil {
		l.DeliveryError(ctx, responses.StatusInternalServerError, responses.ErrInternalServer, err, nil)
		responses.SendErrResponse(w, responses.StatusInternalServerError, responses.ErrInternalServer)

		return
	}

	responses.SendOkResponse(w, resp)
}
