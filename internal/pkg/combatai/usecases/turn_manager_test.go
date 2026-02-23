package usecases

import (
	"testing"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

func TestProcessStartOfTurn_BonusActionReset(t *testing.T) {
	t.Parallel()

	npc := &models.ParticipantFull{
		InstanceID: "npc1",
		RuntimeState: models.CreatureRuntimeState{
			Resources: models.ResourceState{
				BonusActionUsed: true,
			},
		},
	}
	creature := &models.Creature{}

	changed := processStartOfTurn(npc, creature)

	if !changed {
		t.Error("processStartOfTurn should return true when BonusActionUsed is reset")
	}
	if npc.RuntimeState.Resources.BonusActionUsed {
		t.Error("BonusActionUsed should be false after processStartOfTurn")
	}
}

func TestProcessStartOfTurn_BonusActionAlreadyFalse(t *testing.T) {
	t.Parallel()

	npc := &models.ParticipantFull{
		InstanceID: "npc1",
		RuntimeState: models.CreatureRuntimeState{
			Resources: models.ResourceState{
				BonusActionUsed: false,
			},
		},
	}
	creature := &models.Creature{}

	changed := processStartOfTurn(npc, creature)

	if changed {
		t.Error("processStartOfTurn should return false when nothing to reset")
	}
}

func TestProcessStartOfTurn_LegendaryRestore(t *testing.T) {
	t.Parallel()

	npc := &models.ParticipantFull{
		InstanceID: "npc1",
		RuntimeState: models.CreatureRuntimeState{
			Resources: models.ResourceState{
				LegendaryActions: 1,
			},
		},
	}
	creature := &models.Creature{
		Legendary: models.Legendary{Count: 3},
	}

	changed := processStartOfTurn(npc, creature)

	if !changed {
		t.Error("processStartOfTurn should return true when legendary actions restored")
	}
	if npc.RuntimeState.Resources.LegendaryActions != 3 {
		t.Errorf("LegendaryActions = %d, want 3", npc.RuntimeState.Resources.LegendaryActions)
	}
}
