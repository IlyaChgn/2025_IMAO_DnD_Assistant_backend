package delivery

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
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

	var reqData models.DescriptionGenerationRequest
	err := json.NewDecoder(r.Body).Decode(&reqData)
	if err != nil {
		responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrBadJSON)
		return
	}

	resp, err := h.descriptionUseCase.GenerateDescription(ctx, reqData)

	if err != nil {
		switch {
		case errors.Is(err, apperrors.NoDocsErr):
			responses.SendOkResponse(w, nil)
		case errors.Is(err, apperrors.StartPosSizeError):
			responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrSizeOrPosition)
		default:
			responses.SendErrResponse(w, responses.StatusInternalServerError, responses.ErrInternalServer)
		}
		return
	}

	responses.SendOkResponse(w, resp)
}
