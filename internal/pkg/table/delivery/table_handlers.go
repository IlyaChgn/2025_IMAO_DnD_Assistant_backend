package delivery

import (
	"encoding/json"
	"errors"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/delivery/responses"
	tableinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/table"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
)

type TableHandler struct {
	usecases   tableinterfaces.TableUsecases
	ctxUserKey string
	upgrader   *websocket.Upgrader
}

func NewTableHandler(usecases tableinterfaces.TableUsecases, ctxUserKey string) *TableHandler {
	return &TableHandler{
		usecases:   usecases,
		ctxUserKey: ctxUserKey,
		upgrader: &websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}
}

func (h *TableHandler) CreateSession(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var reqData models.CreateTableRequest

	err := json.NewDecoder(r.Body).Decode(&reqData)
	if err != nil {
		responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrBadJSON)

		return
	}

	user := ctx.Value(h.ctxUserKey).(*models.User)

	id, err := h.usecases.CreateSession(ctx, user, reqData.EncounterID)
	if err != nil {
		switch {
		case errors.Is(err, apperrors.PermissionDeniedError) || errors.Is(err, apperrors.ScanError):
			responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrInvalidID)
		default:
			log.Println(err)

			responses.SendErrResponse(w, responses.StatusInternalServerError, responses.ErrInternalServer)
		}

		return
	}

	responses.SendOkResponse(w, models.CreateTableResponse{SessionID: id})
}

func (h *TableHandler) GetTableData(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	sessionID, ok := vars["id"]
	if !ok || sessionID == "" {
		responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrInvalidID)

		return
	}

	data, err := h.usecases.GetTableData(sessionID)
	if err != nil {
		responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrWrongTableID)

		return
	}

	responses.SendOkResponse(w, data)
}

func (h *TableHandler) ServeWS(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	vars := mux.Vars(r)

	sessionID, ok := vars["id"]
	if !ok || sessionID == "" {
		responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrInvalidID)

		return
	}

	user := ctx.Value(h.ctxUserKey).(*models.User)

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		responses.SendErrResponse(w, responses.StatusBadRequest, responses.ErrWSUpgrade)

		return
	}

	h.usecases.AddNewConnection(user, sessionID, conn)
}
