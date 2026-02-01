package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	encmocks "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/encounter/mocks"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/table/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestCreateSession(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("db failure")

	tests := []struct {
		name  string
		admin *models.User
		encID string
		setup func(repo *encmocks.MockEncounterRepository, mgr *mocks.MockTableManager,
			idGen *mocks.MockSessionIDGenerator, tf *mocks.MockTimerFactory, timer *mocks.MockSessionTimer)
		wantErr error
		wantID  string
	}{
		{
			name:  "encounter repo error is propagated",
			admin: &models.User{ID: 1, Name: "Admin"},
			encID: "enc-1",
			setup: func(repo *encmocks.MockEncounterRepository, _ *mocks.MockTableManager,
				_ *mocks.MockSessionIDGenerator, _ *mocks.MockTimerFactory, _ *mocks.MockSessionTimer) {
				repo.EXPECT().GetEncounterByID(gomock.Any(), "enc-1").Return(nil, repoErr)
			},
			wantErr: repoErr,
		},
		{
			name:  "wrong user returns PermissionDeniedError",
			admin: &models.User{ID: 1, Name: "Admin"},
			encID: "enc-1",
			setup: func(repo *encmocks.MockEncounterRepository, _ *mocks.MockTableManager,
				_ *mocks.MockSessionIDGenerator, _ *mocks.MockTimerFactory, _ *mocks.MockSessionTimer) {
				repo.EXPECT().GetEncounterByID(gomock.Any(), "enc-1").
					Return(&models.Encounter{UserID: 999, UUID: "enc-1"}, nil)
			},
			wantErr: apperrors.PermissionDeniedError,
		},
		{
			name:  "happy path returns session ID",
			admin: &models.User{ID: 1, Name: "Admin"},
			encID: "enc-1",
			setup: func(repo *encmocks.MockEncounterRepository, mgr *mocks.MockTableManager,
				idGen *mocks.MockSessionIDGenerator, tf *mocks.MockTimerFactory, timer *mocks.MockSessionTimer) {
				repo.EXPECT().GetEncounterByID(gomock.Any(), "enc-1").
					Return(&models.Encounter{UserID: 1, UUID: "enc-1", Name: "Battle"}, nil)
				idGen.EXPECT().NewSessionID().Return("test-session-abc")
				mgr.EXPECT().CreateSession(gomock.Any(), gomock.Any(), gomock.Any(), "test-session-abc", gomock.Any())
				tf.EXPECT().AfterFunc(gomock.Any(), gomock.Any()).Return(timer)
			},
			wantID: "test-session-abc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			repo := encmocks.NewMockEncounterRepository(ctrl)
			mgr := mocks.NewMockTableManager(ctrl)
			idGen := mocks.NewMockSessionIDGenerator(ctrl)
			tf := mocks.NewMockTimerFactory(ctrl)
			timer := mocks.NewMockSessionTimer(ctrl)
			tt.setup(repo, mgr, idGen, tf, timer)

			uc := NewTableUsecases(repo, mgr, idGen, tf)
			id, err := uc.CreateSession(context.Background(), tt.admin, tt.encID)

			if tt.wantErr != nil {
				assert.True(t, errors.Is(err, tt.wantErr), "expected %v, got %v", tt.wantErr, err)
				assert.Empty(t, id)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.wantID, id)
		})
	}
}

func TestCreateSession_TimerIsRegistered(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	repo := encmocks.NewMockEncounterRepository(ctrl)
	mgr := mocks.NewMockTableManager(ctrl)
	idGen := mocks.NewMockSessionIDGenerator(ctrl)
	tf := mocks.NewMockTimerFactory(ctrl)
	timer := mocks.NewMockSessionTimer(ctrl)

	repo.EXPECT().GetEncounterByID(gomock.Any(), "enc-1").
		Return(&models.Encounter{UserID: 1, UUID: "enc-1", Name: "Battle"}, nil)
	idGen.EXPECT().NewSessionID().Return("sid-1")
	mgr.EXPECT().CreateSession(gomock.Any(), gomock.Any(), gomock.Any(), "sid-1", gomock.Any())
	tf.EXPECT().AfterFunc(gomock.Any(), gomock.Any()).Return(timer)

	uc := NewTableUsecases(repo, mgr, idGen, tf)
	id, err := uc.CreateSession(context.Background(), &models.User{ID: 1, Name: "Admin"}, "enc-1")
	assert.NoError(t, err)
	assert.Equal(t, "sid-1", id)
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
		setup   func(mgr *mocks.MockTableManager)
		wantErr bool
	}{
		{
			name: "happy path returns table data",
			setup: func(mgr *mocks.MockTableManager) {
				mgr.EXPECT().GetTableData(gomock.Any(), "session-1").Return(expected, nil)
			},
		},
		{
			name: "manager error is propagated",
			setup: func(mgr *mocks.MockTableManager) {
				mgr.EXPECT().GetTableData(gomock.Any(), "session-1").Return(nil, managerErr)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			repo := encmocks.NewMockEncounterRepository(ctrl)
			mgr := mocks.NewMockTableManager(ctrl)
			idGen := mocks.NewMockSessionIDGenerator(ctrl)
			tf := mocks.NewMockTimerFactory(ctrl)
			tt.setup(mgr)

			uc := NewTableUsecases(repo, mgr, idGen, tf)
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

	encounter := &models.Encounter{UserID: 1, UUID: "enc-1"}
	admin := &models.User{ID: 1, Name: "Admin"}

	// First session
	ctrl1 := gomock.NewController(t)
	repo1 := encmocks.NewMockEncounterRepository(ctrl1)
	mgr1 := mocks.NewMockTableManager(ctrl1)
	idGen1 := mocks.NewMockSessionIDGenerator(ctrl1)
	tf1 := mocks.NewMockTimerFactory(ctrl1)
	timer1 := mocks.NewMockSessionTimer(ctrl1)

	repo1.EXPECT().GetEncounterByID(gomock.Any(), "enc-1").Return(encounter, nil)
	idGen1.EXPECT().NewSessionID().Return("session-A")
	mgr1.EXPECT().CreateSession(gomock.Any(), gomock.Any(), gomock.Any(), "session-A", gomock.Any())
	tf1.EXPECT().AfterFunc(gomock.Any(), gomock.Any()).Return(timer1)

	uc1 := NewTableUsecases(repo1, mgr1, idGen1, tf1)
	id1, err1 := uc1.CreateSession(context.Background(), admin, "enc-1")
	assert.NoError(t, err1)
	assert.Equal(t, "session-A", id1)

	// Second session with different ID
	ctrl2 := gomock.NewController(t)
	repo2 := encmocks.NewMockEncounterRepository(ctrl2)
	mgr2 := mocks.NewMockTableManager(ctrl2)
	idGen2 := mocks.NewMockSessionIDGenerator(ctrl2)
	tf2 := mocks.NewMockTimerFactory(ctrl2)
	timer2 := mocks.NewMockSessionTimer(ctrl2)

	repo2.EXPECT().GetEncounterByID(gomock.Any(), "enc-1").Return(encounter, nil)
	idGen2.EXPECT().NewSessionID().Return("session-B")
	mgr2.EXPECT().CreateSession(gomock.Any(), gomock.Any(), gomock.Any(), "session-B", gomock.Any())
	tf2.EXPECT().AfterFunc(gomock.Any(), gomock.Any()).Return(timer2)

	uc2 := NewTableUsecases(repo2, mgr2, idGen2, tf2)
	id2, err2 := uc2.CreateSession(context.Background(), admin, "enc-1")
	assert.NoError(t, err2)
	assert.Equal(t, "session-B", id2)
	assert.NotEqual(t, id1, id2)
}

// Verify that AfterFunc receives a reasonable duration (non-zero).
func TestCreateSession_TimerDuration(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	repo := encmocks.NewMockEncounterRepository(ctrl)
	mgr := mocks.NewMockTableManager(ctrl)
	idGen := mocks.NewMockSessionIDGenerator(ctrl)
	tf := mocks.NewMockTimerFactory(ctrl)
	timer := mocks.NewMockSessionTimer(ctrl)

	repo.EXPECT().GetEncounterByID(gomock.Any(), "enc-1").
		Return(&models.Encounter{UserID: 1, UUID: "enc-1"}, nil)
	idGen.EXPECT().NewSessionID().Return("sid-dur")
	mgr.EXPECT().CreateSession(gomock.Any(), gomock.Any(), gomock.Any(), "sid-dur", gomock.Any())

	var capturedDuration time.Duration
	tf.EXPECT().AfterFunc(gomock.Any(), gomock.Any()).DoAndReturn(
		func(d time.Duration, f func()) *mocks.MockSessionTimer {
			capturedDuration = d
			return timer
		})

	uc := NewTableUsecases(repo, mgr, idGen, tf)
	_, err := uc.CreateSession(context.Background(), &models.User{ID: 1}, "enc-1")
	assert.NoError(t, err)
	assert.True(t, capturedDuration > 0, "timer duration should be positive")
}
