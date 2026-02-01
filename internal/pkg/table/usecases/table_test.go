package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	tableinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/table"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

// --- fakes ---

type fakeEncounterRepo struct {
	encounter *models.Encounter
	getErr    error
	updateErr error
}

func (f *fakeEncounterRepo) GetEncountersListWithSearch(_ context.Context, _, _, _ int,
	_ *models.SearchParams) (*models.EncountersList, error) {
	return nil, nil
}
func (f *fakeEncounterRepo) GetEncountersList(_ context.Context, _, _, _ int) (*models.EncountersList, error) {
	return nil, nil
}
func (f *fakeEncounterRepo) GetEncounterByID(_ context.Context, _ string) (*models.Encounter, error) {
	return f.encounter, f.getErr
}
func (f *fakeEncounterRepo) SaveEncounter(_ context.Context, _ *models.SaveEncounterReq, _ string, _ int) error {
	return nil
}
func (f *fakeEncounterRepo) UpdateEncounter(_ context.Context, _ []byte, _ string) error {
	return f.updateErr
}
func (f *fakeEncounterRepo) RemoveEncounter(_ context.Context, _ string) error { return nil }
func (f *fakeEncounterRepo) CheckPermission(_ context.Context, _ string, _ int) bool {
	return false
}

type fakeTableManager struct {
	createCalled bool
	createdID    string
	tableData    *models.TableData
	tableErr     error
	encounterRaw []byte
	encounterErr error
	removeCalled bool
}

func (f *fakeTableManager) CreateSession(_ context.Context, _ *models.User, _ *models.Encounter,
	sessionID string, _ func(string)) {
	f.createCalled = true
	f.createdID = sessionID
}
func (f *fakeTableManager) RemoveSession(_ context.Context, _ string) {
	f.removeCalled = true
}
func (f *fakeTableManager) GetTableData(_ context.Context, _ string) (*models.TableData, error) {
	return f.tableData, f.tableErr
}
func (f *fakeTableManager) GetEncounterData(_ context.Context, _ string) ([]byte, error) {
	return f.encounterRaw, f.encounterErr
}
func (f *fakeTableManager) AddNewConnection(_ context.Context, _ *models.User, _ string,
	_ *websocket.Conn) {
}
func (f *fakeTableManager) HasActiveUsers(_ context.Context, _ string) bool { return false }

type fixedSessionIDGen struct {
	id string
}

func (g *fixedSessionIDGen) NewSessionID() string { return g.id }

type fakeTimer struct {
	stopCalled  bool
	resetCalled bool
}

func (t *fakeTimer) Stop() bool                 { t.stopCalled = true; return true }
func (t *fakeTimer) Reset(_ time.Duration) bool { t.resetCalled = true; return true }

type fakeTimerFactory struct {
	lastTimer *fakeTimer
}

func (f *fakeTimerFactory) AfterFunc(_ time.Duration, _ func()) tableinterfaces.SessionTimer {
	f.lastTimer = &fakeTimer{}
	return f.lastTimer
}

// --- tests ---

func TestCreateSession(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("db failure")

	tests := []struct {
		name    string
		admin   *models.User
		encID   string
		repo    *fakeEncounterRepo
		manager *fakeTableManager
		fixedID string
		wantErr error
		wantID  string
	}{
		{
			name:    "encounter repo error is propagated",
			admin:   &models.User{ID: 1, Name: "Admin"},
			encID:   "enc-1",
			repo:    &fakeEncounterRepo{getErr: repoErr},
			manager: &fakeTableManager{},
			fixedID: "session-1",
			wantErr: repoErr,
		},
		{
			name:  "wrong user returns PermissionDeniedError",
			admin: &models.User{ID: 1, Name: "Admin"},
			encID: "enc-1",
			repo: &fakeEncounterRepo{
				encounter: &models.Encounter{UserID: 999, UUID: "enc-1"},
			},
			manager: &fakeTableManager{},
			fixedID: "session-2",
			wantErr: apperrors.PermissionDeniedError,
		},
		{
			name:  "happy path returns session ID",
			admin: &models.User{ID: 1, Name: "Admin"},
			encID: "enc-1",
			repo: &fakeEncounterRepo{
				encounter: &models.Encounter{UserID: 1, UUID: "enc-1", Name: "Battle"},
			},
			manager: &fakeTableManager{},
			fixedID: "test-session-abc",
			wantID:  "test-session-abc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tf := &fakeTimerFactory{}
			uc := NewTableUsecases(tt.repo, tt.manager,
				&fixedSessionIDGen{id: tt.fixedID}, tf)

			id, err := uc.CreateSession(context.Background(), tt.admin, tt.encID)

			if tt.wantErr != nil {
				assert.True(t, errors.Is(err, tt.wantErr), "expected %v, got %v", tt.wantErr, err)
				assert.Empty(t, id)
				assert.False(t, tt.manager.createCalled)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.wantID, id)
			assert.True(t, tt.manager.createCalled)
			assert.Equal(t, tt.fixedID, tt.manager.createdID)
			assert.NotNil(t, tf.lastTimer, "timer should be created")
		})
	}
}

func TestCreateSession_TimerIsRegistered(t *testing.T) {
	t.Parallel()

	repo := &fakeEncounterRepo{
		encounter: &models.Encounter{UserID: 1, UUID: "enc-1", Name: "Battle"},
	}
	manager := &fakeTableManager{}
	tf := &fakeTimerFactory{}

	uc := NewTableUsecases(repo, manager, &fixedSessionIDGen{id: "sid-1"}, tf)

	id, err := uc.CreateSession(context.Background(), &models.User{ID: 1, Name: "Admin"}, "enc-1")
	assert.NoError(t, err)
	assert.Equal(t, "sid-1", id)
	assert.NotNil(t, tf.lastTimer)
}

func TestGetTableData(t *testing.T) {
	t.Parallel()

	expected := &models.TableData{
		AdminName:     "Admin",
		EncounterName: "Battle",
	}
	managerErr := errors.New("not found")

	tests := []struct {
		name    string
		manager *fakeTableManager
		wantErr bool
	}{
		{
			name:    "happy path returns table data",
			manager: &fakeTableManager{tableData: expected},
		},
		{
			name:    "manager error is propagated",
			manager: &fakeTableManager{tableErr: managerErr},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			uc := NewTableUsecases(&fakeEncounterRepo{}, tt.manager,
				&fixedSessionIDGen{id: "unused"}, &fakeTimerFactory{})

			result, err := uc.GetTableData(context.Background(), "session-1")

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, expected, result)
			}
		})
	}
}

func TestCreateSession_MultipleSessionsGetUniqueIDs(t *testing.T) {
	t.Parallel()

	repo := &fakeEncounterRepo{
		encounter: &models.Encounter{UserID: 1, UUID: "enc-1"},
	}
	manager := &fakeTableManager{}

	// First session
	uc1 := NewTableUsecases(repo, manager, &fixedSessionIDGen{id: "session-A"}, &fakeTimerFactory{})
	id1, err1 := uc1.CreateSession(context.Background(), &models.User{ID: 1, Name: "Admin"}, "enc-1")
	assert.NoError(t, err1)
	assert.Equal(t, "session-A", id1)

	// Second session with different ID
	manager2 := &fakeTableManager{}
	uc2 := NewTableUsecases(repo, manager2, &fixedSessionIDGen{id: "session-B"}, &fakeTimerFactory{})
	id2, err2 := uc2.CreateSession(context.Background(), &models.User{ID: 1, Name: "Admin"}, "enc-1")
	assert.NoError(t, err2)
	assert.Equal(t, "session-B", id2)
	assert.NotEqual(t, id1, id2)
}
