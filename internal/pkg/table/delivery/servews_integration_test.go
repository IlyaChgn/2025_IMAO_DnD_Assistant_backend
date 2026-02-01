//go:build integration
// +build integration

package delivery_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/table/delivery"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

const integrationCtxUserKey = "test-user-key"

// wsRecordingUsecases records AddNewConnection calls, then closes the conn.
type wsRecordingUsecases struct {
	mu           sync.Mutex
	connReceived bool
	sessionIDGot string
	userGot      *models.User
	done         chan struct{}
}

func (f *wsRecordingUsecases) CreateSession(_ context.Context, _ *models.User, _ string) (string, error) {
	return "", nil
}

func (f *wsRecordingUsecases) GetTableData(_ context.Context, _ string) (*models.TableData, error) {
	return nil, nil
}

func (f *wsRecordingUsecases) AddNewConnection(_ context.Context, user *models.User, sessionID string,
	conn *websocket.Conn) {
	f.mu.Lock()
	f.connReceived = true
	f.sessionIDGot = sessionID
	f.userGot = user
	f.mu.Unlock()

	close(f.done)
	conn.Close()
}

// TestServeWS_UpgradeAndConnect verifies that the websocket upgrade succeeds
// through a real HTTP server with gorilla/mux routing, and that the handler
// correctly extracts the session ID from the URL and the user from the context.
func TestServeWS_UpgradeAndConnect(t *testing.T) {
	testUser := &models.User{ID: 42, Name: "WS Tester"}
	fake := &wsRecordingUsecases{done: make(chan struct{})}
	handler := delivery.NewTableHandler(fake, integrationCtxUserKey)

	router := mux.NewRouter()
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), integrationCtxUserKey, testUser)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	router.HandleFunc("/table/session/{id}/connect", handler.ServeWS).Methods("GET")

	server := httptest.NewServer(router)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/table/session/test-session-42/connect"

	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	assert.NoError(t, err, "websocket dial should succeed")

	if conn != nil {
		defer conn.Close()
	}

	if resp != nil {
		assert.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)
		resp.Body.Close()
	}

	// Wait for the fake to record the call (channel-based, no sleep).
	select {
	case <-fake.done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for AddNewConnection to be called")
	}

	fake.mu.Lock()
	defer fake.mu.Unlock()

	assert.True(t, fake.connReceived, "AddNewConnection should have been called")
	assert.Equal(t, "test-session-42", fake.sessionIDGot)
	assert.Equal(t, testUser.ID, fake.userGot.ID)
	assert.Equal(t, testUser.Name, fake.userGot.Name)
}
