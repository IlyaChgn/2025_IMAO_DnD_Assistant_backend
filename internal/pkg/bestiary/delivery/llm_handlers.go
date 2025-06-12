package delivery

import (
	"encoding/json"
	"errors"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"io"
	"log"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	bestiaryinterface "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/bestiary"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/delivery/responses"
)

type LLMHandler struct {
	usecases bestiaryinterface.GenerationUsecases
}

func NewLLMHandler(usecases bestiaryinterface.GenerationUsecases) *LLMHandler {
	return &LLMHandler{
		usecases: usecases,
	}
}

// POST /api/llm/text
// body: { "description": "какое-то описание" }
// ответ: { "job_id": "<uuid>" }
func (h *LLMHandler) SubmitGenerationPrompt(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req models.DescriptionGenPrompt

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrBadJSON)

		return
	}

	jobID, err := h.usecases.SubmitText(ctx, req.Description)
	if err != nil {
		responses.SendErrResponse(w, responses.StatusInternalServerError, responses.ErrInternalServer)

		return
	}

	responses.SendOkResponse(w, &models.LLMJobResponse{JobID: jobID})
}

// POST /api/llm/image
// raw-body или multipart/form-data field "image"
// ответ: { "job_id": "<uuid>" }
func (h *LLMHandler) SubmitGenerationImage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var imgBytes []byte

	if r.Header.Get("Content-Type") == "application/octet-stream" {
		var err error

		imgBytes, err = io.ReadAll(r.Body)
		if err != nil {
			responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrWrongImage)

			return
		}
	} else {
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrWrongFileSize)

			return
		}
		file, _, err := r.FormFile("image")
		if err != nil {
			responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrEmptyImage)

			return
		}
		defer file.Close()

		imgBytes, err = io.ReadAll(file)
		if err != nil {
			log.Println("SubmitGenerationImage: read file error:", err)
			responses.SendErrResponse(w, responses.StatusInternalServerError, responses.ErrInternalServer)

			return
		}
	}

	jobID, err := h.usecases.SubmitImage(ctx, imgBytes)
	if err != nil {
		responses.SendErrResponse(w, responses.StatusInternalServerError, responses.ErrInternalServer)

		return
	}

	responses.SendOkResponse(w, models.LLMJobResponse{JobID: jobID})
}

// GET /api/llm/{id}
// ответ до готовности: { "status": "processing_step_1" } or { "status": "processing_step_2" }
// когда done:             { "status": "done", "result": <models.Creature> }
func (h *LLMHandler) GetGenerationStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	job, err := h.usecases.GetJob(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, apperrors.NotFoundError):
			responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrWrongJobID)
		default:
			log.Println("GetGenerationStatus:", err)
			responses.SendErrResponse(w, responses.StatusInternalServerError, responses.ErrInternalServer)
		}

		return
	}

	var resp models.LLMJobStatusResponse

	if job.Status == "done" && job.Result != nil {
		resp.Result = job.Result
	}

	responses.SendOkResponse(w, resp)
}
