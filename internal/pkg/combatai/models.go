package combatai

import "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"

// TurnInput contains everything the AI needs to decide a single NPC turn.
// Fully self-contained — no DB access required.
type TurnInput struct {
	// Who is taking the turn.
	ActiveNPC        models.ParticipantFull
	CreatureTemplate models.Creature

	// Full combat state.
	Participants []models.ParticipantFull
	CurrentRound int

	// Combat stats for all participants (key = InstanceID).
	// Abstracts the NPC/PC data model difference.
	CombatantStats map[string]CombatantStats

	// Intelligence (0.05–1.0), pre-computed from INT score + aiDifficultyMod.
	Intelligence float64

	// Previous target for sticky targeting (empty string = first round or target dead).
	PreviousTargetID string

	// Grid dimensions (for movement, Phase 3).
	MapWidth  int
	MapHeight int

	// Walkability grid (nil = movement disabled in Phase 1).
	WalkabilityGrid [][]bool
}

// CombatantStats is a unified slice of combat characteristics for a participant.
// Abstracts the difference between NPC (Creature) and PC (CharacterBase + DerivedStats).
// Built by the usecases layer when constructing TurnInput.
type CombatantStats struct {
	MaxHP           int            // maximum hit points (for HP% calculations, 0 = unknown)
	AC              int            // ArmorClass (including StatModifiers)
	SaveBonuses     map[string]int // ability type ("STR","DEX",...) → total save bonus
	Resistances     []string       // damage types: "fire", "cold", etc.
	Immunities      []string       // damage type immunities
	Vulnerabilities []string       // damage type vulnerabilities
	IsPC            bool           // true for player characters (affects targeting priority)
}

// TurnDecision is the AI's output for a single NPC turn.
type TurnDecision struct {
	Movement    *MovementDecision // nil = don't move
	Action      *ActionDecision   // nil = skip turn (e.g. incapacitated)
	BonusAction *ActionDecision   // nil = no suitable bonus action
	Reaction    *ReactionRule     // Phase 2+
	Reasoning   string            // human-readable explanation for DM log
}

// ActionDecision describes a concrete action or multiattack sequence,
// ready to be converted into one or more ActionRequest calls.
type ActionDecision struct {
	// Single action fields (used when MultiattackSteps is nil).
	ActionType models.ActionType // weapon_attack, spell_cast, use_feature, "" for universal
	ActionID   string            // StructuredAction.ID or SpellID
	TargetIDs  []string          // target instance IDs
	SlotLevel  int               // for spell_cast: slot level (upcast)

	// Multiattack fields (when set, single action fields above are ignored).
	MultiattackGroupID string            // MultiattackGroup.ID (for logging)
	MultiattackSteps   []MultiattackStep // nil = single action

	// Metadata for logging.
	ActionName     string
	ExpectedDamage float64 // estimated damage (sum for multiattack)
	Reasoning      string  // why this action was chosen
}

// MultiattackStep is one attack within a multiattack sequence.
type MultiattackStep struct {
	ActionType models.ActionType // weapon_attack
	ActionID   string            // specific attack ID (e.g. "bite", "claw")
	TargetIDs  []string          // target for this attack (may differ between steps)
}

// MovementDecision describes where to move.
type MovementDecision struct {
	TargetX   int
	TargetY   int
	Path      []models.CellsCoordinates // cell path for animation
	Reasoning string
}

// ReactionRule defines a condition for automatic reaction usage.
type ReactionRule struct {
	ActionID  string // reaction StructuredAction.ID
	Trigger   string // "opportunity_attack", "shield_spell", etc.
	Condition string // human-readable condition
}

// AITurnResult is the response from ExecuteAITurn — the AI's decision plus
// the action pipeline results from executing that decision.
type AITurnResult struct {
	NpcInstanceID string                   `json:"npcInstanceId"`
	Decision      *TurnDecision            `json:"decision"`
	ActionResults []*models.ActionResponse `json:"actionResults,omitempty"`
}

// AIRoundResult is the response from ExecuteAIRound — all NPC turns in one round.
type AIRoundResult struct {
	Round        int           `json:"round"`
	Turns        []AIRoundTurn `json:"turns"`
	CombatEnded  bool          `json:"combatEnded"`
	CombatResult string        `json:"combatResult,omitempty"`
}

// AIRoundTurn is one NPC's turn within an ai-round execution.
type AIRoundTurn struct {
	NpcID         string                  `json:"npcID"`
	NpcName       string                  `json:"npcName"`
	Decision      *TurnDecision           `json:"decision"`
	ActionResults []*models.ActionResponse `json:"actionResults,omitempty"`
	Skipped       bool                    `json:"skipped"`
	SkipReason    string                  `json:"skipReason,omitempty"`
}

// CreatureRole classifies a creature's combat behavior archetype.
type CreatureRole string

const (
	RoleBrute      CreatureRole = "brute"      // melee-oriented, high STR/HP
	RoleRanged     CreatureRole = "ranged"      // ranged attacks, keeps distance
	RoleCaster     CreatureRole = "caster"      // has Spellcasting, prioritizes spells
	RoleSkirmisher CreatureRole = "skirmisher"  // high DEX, hit-and-run
	RoleController CreatureRole = "controller"  // AoE/conditions, battlefield control
	RoleTank       CreatureRole = "tank"        // high AC/HP, protects allies
)
