package delivery_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/delivery/responses"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/table/delivery"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/testhelpers"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

// --- fake usecase ---

type fakeTableUsecases struct {
	sessionID string
	createErr error
	tableData *models.TableData
	tableErr  error
}

func (f *fakeTableUsecases) CreateSession(_ context.Context, _ *models.User, _ string) (string, error) {
	return f.sessionID, f.createErr
}
func (f *fakeTableUsecases) GetTableData(_ context.Context, _ string) (*models.TableData, error) {
	return f.tableData, f.tableErr
}
func (f *fakeTableUsecases) AddNewConnection(_ context.Context, _ *models.User, _ string,
	_ *websocket.Conn) {
}

// --- helpers ---

const ctxUserKey = "test-user-key"

func withUser(r *http.Request, key string, user *models.User) *http.Request {
	ctx := context.WithValue(r.Context(), key, user)
	return r.WithContext(ctx)
}

// --- tests ---

func TestCreateSession_BadJSON_Returns400(t *testing.T) {
	t.Parallel()

	handler := delivery.NewTableHandler(&fakeTableUsecases{}, ctxUserKey)

	req := httptest.NewRequest(http.MethodPost, "/api/table/create", nil)
	req.Body = io.NopCloser(bytes.NewReader([]byte(`{invalid json`)))
	req = withUser(req, ctxUserKey, &models.User{ID: 1, Name: "Tester"})

	rr := httptest.NewRecorder()
	handler.CreateSession(rr, req)

	assert.Equal(t, responses.StatusBadRequest, rr.Code)
	assert.Equal(t, responses.ErrBadJSON, testhelpers.DecodeErrorResponse(t, rr.Body))
}

func TestCreateSession_PermissionDenied_Returns400(t *testing.T) {
	t.Parallel()

	handler := delivery.NewTableHandler(
		&fakeTableUsecases{createErr: apperrors.PermissionDeniedError},
		ctxUserKey,
	)

	body := testhelpers.MustJSON(t, models.CreateTableRequest{EncounterID: "enc-1"})
	req := httptest.NewRequest(http.MethodPost, "/api/table/create", nil)
	req.Body = io.NopCloser(bytes.NewReader(body))
	req = withUser(req, ctxUserKey, &models.User{ID: 1, Name: "Tester"})

	rr := httptest.NewRecorder()
	handler.CreateSession(rr, req)

	assert.Equal(t, responses.StatusBadRequest, rr.Code)
	assert.Equal(t, responses.ErrInvalidID, testhelpers.DecodeErrorResponse(t, rr.Body))
}

func TestCreateSession_ScanError_Returns400(t *testing.T) {
	t.Parallel()

	handler := delivery.NewTableHandler(
		&fakeTableUsecases{createErr: apperrors.ScanError},
		ctxUserKey,
	)

	body := testhelpers.MustJSON(t, models.CreateTableRequest{EncounterID: "enc-1"})
	req := httptest.NewRequest(http.MethodPost, "/api/table/create", nil)
	req.Body = io.NopCloser(bytes.NewReader(body))
	req = withUser(req, ctxUserKey, &models.User{ID: 1, Name: "Tester"})

	rr := httptest.NewRecorder()
	handler.CreateSession(rr, req)

	assert.Equal(t, responses.StatusBadRequest, rr.Code)
	assert.Equal(t, responses.ErrInvalidID, testhelpers.DecodeErrorResponse(t, rr.Body))
}

func TestCreateSession_GenericError_Returns500(t *testing.T) {
	t.Parallel()

	handler := delivery.NewTableHandler(
		&fakeTableUsecases{createErr: errors.New("unexpected")},
		ctxUserKey,
	)

	body := testhelpers.MustJSON(t, models.CreateTableRequest{EncounterID: "enc-1"})
	req := httptest.NewRequest(http.MethodPost, "/api/table/create", nil)
	req.Body = io.NopCloser(bytes.NewReader(body))
	req = withUser(req, ctxUserKey, &models.User{ID: 1, Name: "Tester"})

	rr := httptest.NewRecorder()
	handler.CreateSession(rr, req)

	assert.Equal(t, responses.StatusInternalServerError, rr.Code)
	assert.Equal(t, responses.ErrInternalServer, testhelpers.DecodeErrorResponse(t, rr.Body))
}

// --- GetTableData tests ---

func TestGetTableData_MissingID_Returns400(t *testing.T) {
	t.Parallel()

	handler := delivery.NewTableHandler(&fakeTableUsecases{}, ctxUserKey)

	req := httptest.NewRequest(http.MethodGet, "/api/table/", nil)
	// No mux vars set — simulates missing "id" parameter
	rr := httptest.NewRecorder()
	handler.GetTableData(rr, req)

	assert.Equal(t, responses.StatusBadRequest, rr.Code)
	assert.Equal(t, responses.ErrInvalidID, testhelpers.DecodeErrorResponse(t, rr.Body))
}

func TestGetTableData_UsecaseError_Returns400(t *testing.T) {
	t.Parallel()

	handler := delivery.NewTableHandler(
		&fakeTableUsecases{tableErr: errors.New("session not found")},
		ctxUserKey,
	)

	req := httptest.NewRequest(http.MethodGet, "/api/table/session-1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "session-1"})

	rr := httptest.NewRecorder()
	handler.GetTableData(rr, req)

	assert.Equal(t, responses.StatusBadRequest, rr.Code)
	assert.Equal(t, responses.ErrWrongTableID, testhelpers.DecodeErrorResponse(t, rr.Body))
}

func TestGetTableData_HappyPath_Returns200(t *testing.T) {
	t.Parallel()

	expected := &models.TableData{
		AdminName:     "Admin",
		EncounterName: "Battle",
	}

	handler := delivery.NewTableHandler(
		&fakeTableUsecases{tableData: expected},
		ctxUserKey,
	)

	req := httptest.NewRequest(http.MethodGet, "/api/table/session-1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "session-1"})

	rr := httptest.NewRecorder()
	handler.GetTableData(rr, req)

	assert.Equal(t, responses.StatusOk, rr.Code)

	var got models.TableData
	testhelpers.DecodeJSON(t, rr.Body, &got)
	assert.Equal(t, "Admin", got.AdminName)
	assert.Equal(t, "Battle", got.EncounterName)
}
