package delivery

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	authinterface "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/auth"
	bestiaryinterface "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/bestiary"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/delivery/responses"
)

type LLMHandler struct {
	usecases     bestiaryinterface.GenerationUsecases
	authUsecases authinterface.AuthUsecases
}

func NewLLMHandler(usecases bestiaryinterface.GenerationUsecases,
	authUsecases authinterface.AuthUsecases) *LLMHandler {
	return &LLMHandler{
		usecases:     usecases,
		authUsecases: authUsecases,
	}
}

// POST /api/llm/text
// body: { "description": "какое-то описание" }
// ответ: { "job_id": "<uuid>" }
func (h *LLMHandler) SubmitGenerationPrompt(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req struct {
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Println("SubmitGenerationPrompt: bad JSON:", err)
		responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrBadJSON)
		return
	}

	jobID, err := h.usecases.SubmitText(ctx, req.Description)
	if err != nil {
		log.Println("SubmitGenerationPrompt:", err)
		responses.SendErrResponse(w, responses.StatusInternalServerError, responses.ErrInternalServer)
		return
	}

	responses.SendOkResponse(w, map[string]string{"job_id": jobID})
}

// POST /api/llm/image
// raw-body или multipart/form-data field "image"
// ответ: { "job_id": "<uuid>" }
func (h *LLMHandler) SubmitGenerationImage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// поддержим и raw-body, и multipart
	var imgBytes []byte
	if r.Header.Get("Content-Type") == "application/octet-stream" {
		var err error
		imgBytes, err = io.ReadAll(r.Body)
		if err != nil {
			log.Println("SubmitGenerationImage: failed to read body:", err)
			responses.SendErrResponse(w, responses.StatusBadRequest, "Failed to read image")
			return
		}
	} else {
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			log.Println("SubmitGenerationImage: parse form error:", err)
			responses.SendErrResponse(w, responses.StatusBadRequest, "Invalid form data")
			return
		}
		file, _, err := r.FormFile("image")
		if err != nil {
			log.Println("SubmitGenerationImage: form file error:", err)
			responses.SendErrResponse(w, responses.StatusBadRequest, "Image not provided")
			return
		}
		defer file.Close()

		imgBytes, err = io.ReadAll(file)
		if err != nil {
			log.Println("SubmitGenerationImage: read file error:", err)
			responses.SendErrResponse(w, responses.StatusInternalServerError, "Failed to read image")
			return
		}
	}

	jobID, err := h.usecases.SubmitImage(ctx, imgBytes)
	if err != nil {
		log.Println("SubmitGenerationImage:", err)
		responses.SendErrResponse(w, responses.StatusInternalServerError, responses.ErrInternalServer)
		return
	}

	responses.SendOkResponse(w, map[string]string{"job_id": jobID})
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
			responses.SendErrResponse(w, responses.StatusBadRequest, "Job not found")
		default:
			log.Println("GetGenerationStatus:", err)
			responses.SendErrResponse(w, responses.StatusInternalServerError, responses.ErrInternalServer)
		}
		return
	}

	// собираем ответ
	resp := map[string]interface{}{
		"status": job.Status,
	}
	if job.Status == "done" && job.Result != nil {
		resp["result"] = job.Result
	}

	responses.SendOkResponse(w, resp)
}
