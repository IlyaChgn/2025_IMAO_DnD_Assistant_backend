package actions

import (
	"context"
	"time"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

// EncounterDataReader provides read-only access to encounter data for reaction evaluation.
type EncounterDataReader interface {
	GetParticipants() []models.ParticipantFull
	FindParticipantByInstanceID(id string) (*models.ParticipantFull, int, error)
}

// ReactionEvaluator evaluates NPC reactions during the action pipeline.
// Implemented by combatai/usecases, injected into actionsUsecases via setter.
type ReactionEvaluator interface {
	EvaluateShield(ctx context.Context, ed EncounterDataReader,
		targetID string, attackTotal int) (*ShieldReactionResult, error)
	EvaluateCounterspell(ctx context.Context, ed EncounterDataReader,
		casterID string, spellLevel int) (*CounterspellReactionResult, error)
	EvaluateParry(ctx context.Context, ed EncounterDataReader,
		targetID string, incomingDamage int) (*ParryReactionResult, error)
}

// ReactionEvaluatorSetter allows injecting the ReactionEvaluator after construction.
type ReactionEvaluatorSetter interface {
	SetReactionEvaluator(re ReactionEvaluator)
}

// ShieldReactionResult describes a Shield reaction outcome.
type ShieldReactionResult struct {
	ReactorID      string // NPC instance ID that cast Shield
	ReactorName    string
	SpellID        string // "shield"
	SlotLevel      int    // slot spent (0 for innate)
	InnateKey      string // non-empty for per-day innate casts (e.g. "innate:shield")
	ACBonus        int    // always +5
	NewEffectiveAC int    // AC after Shield applied
}

// CounterspellReactionResult describes a Counterspell reaction outcome.
type CounterspellReactionResult struct {
	ReactorID    string
	ReactorName  string
	SpellID      string // "counterspell"
	SlotLevel    int    // slot spent (0 for innate)
	InnateKey    string // non-empty for per-day innate casts
	Success      bool   // true if spell is countered
	AbilityCheck *int   // non-nil if ability check was needed (higher-level spell)
	CheckDC      int
}

// ParryReactionResult describes a Parry reaction outcome.
type ParryReactionResult struct {
	ReactorID       string
	ReactorName     string
	ActionID        string // parry StructuredAction ID
	DamageReduction int    // proficiency + DEX mod
}

type ActionsUsecases interface {
	ExecuteAction(ctx context.Context, encounterID string,
		req *models.ActionRequest, userID int) (*models.ActionResponse, error)
	GetActionLog(ctx context.Context, encounterID string, userID int,
		limit int, before time.Time) ([]*models.AuditLogEntry, error)
}

// AuditLogRepository handles persistence for the action_log MongoDB collection.
type AuditLogRepository interface {
	Insert(ctx context.Context, entry *models.AuditLogEntry) error
	GetByEncounterID(ctx context.Context, encounterID string, limit int, before time.Time) ([]*models.AuditLogEntry, error)
	EnsureIndexes(ctx context.Context) error
}
