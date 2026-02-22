package usecases

import (
	"context"
	"fmt"
	"strings"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/dice"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
)

// resolveNpcSpellCast handles the "spell_cast" action type for NPC participants.
// Supports regular spellcasting (slot-based), innate spellcasting (at-will / per-day), and cantrips.
func resolveNpcSpellCast(
	ctx context.Context,
	uc *actionsUsecases,
	cmd *models.ActionCommand,
	encounterID string,
	creature *models.Creature,
	actorName string,
	participant *models.ParticipantFull,
	ed *EncounterData,
	userID int,
) (*models.ActionResponse, error) {
	l := logger.FromContext(ctx)

	if cmd.SpellID == "" {
		return nil, apperrors.MissingSpellIDErr
	}

	// Find spell on the creature
	source, err := findNpcSpell(creature, cmd.SpellID)
	if err != nil {
		return nil, err
	}

	resources := &participant.RuntimeState.Resources

	var stateChanges []models.StateChange
	mutated := false
	slotLevel := cmd.SlotLevel

	// Determine slot level from spell if not specified
	if slotLevel == 0 && source.Spell.Level > 0 {
		slotLevel = source.Spell.Level
	}

	// Slot/use deduction
	if source.IsInnate {
		if !source.IsAtWill && source.PerDayLimit > 0 {
			// Innate per-day spell — track in AbilityUses
			resourceKey := fmt.Sprintf("innate:%s", cmd.SpellID)
			if resources.AbilityUses == nil {
				resources.AbilityUses = make(map[string]int)
			}
			remaining, exists := resources.AbilityUses[resourceKey]
			if !exists {
				remaining = source.PerDayLimit
				resources.AbilityUses[resourceKey] = remaining
			}
			if remaining <= 0 {
				return nil, apperrors.InnateUsesExhaustedErr
			}
			resources.AbilityUses[resourceKey] = remaining - 1
			mutated = true
			stateChanges = append(stateChanges, models.StateChange{
				Description: fmt.Sprintf("Used innate %s (%d/%d uses remaining)",
					source.Spell.Name, remaining-1, source.PerDayLimit),
			})
		}
		// At-will: no deduction needed
	} else if slotLevel > 0 {
		// Regular slot-based spellcasting
		if resources.SpellSlots == nil {
			resources.SpellSlots = make(map[int]int)
		}

		// Initialize from template if key missing
		remaining, exists := resources.SpellSlots[slotLevel]
		if !exists {
			if creature.Spellcasting != nil && creature.Spellcasting.SpellSlots != nil {
				remaining = creature.Spellcasting.SpellSlots[slotLevel]
			}
			resources.SpellSlots[slotLevel] = remaining
		}

		if remaining <= 0 {
			return nil, apperrors.NpcSpellSlotsExhaustedErr
		}
		resources.SpellSlots[slotLevel] = remaining - 1
		mutated = true
		stateChanges = append(stateChanges, models.StateChange{
			SlotSpent: slotLevel,
			Description: fmt.Sprintf("Spent a level %d spell slot (%d/%d remaining)",
				slotLevel, remaining-1, creature.Spellcasting.SpellSlots[slotLevel]),
		})
	}
	// Cantrips (level 0): no slot deduction

	// Load spell definition for rich resolution
	var spellDef *models.SpellDefinition
	if uc.spellsRepo != nil {
		var sErr error
		spellDef, sErr = uc.spellsRepo.GetSpellByID(ctx, cmd.SpellID)
		if sErr != nil {
			l.UsecasesWarn(sErr, userID, map[string]any{"spellID": cmd.SpellID})
			// Continue without definition
		}
	}

	// Concentration handling
	if spellDef != nil && spellDef.Concentration {
		if participant.RuntimeState.Concentration != nil {
			stateChanges = append(stateChanges, models.StateChange{
				Description: fmt.Sprintf("Dropped concentration on %s",
					participant.RuntimeState.Concentration.EffectName),
			})
		}
		participant.RuntimeState.Concentration = &models.ConcentrationState{
			EffectName: source.Spell.Name,
			EffectID:   cmd.SpellID,
		}
		mutated = true
		stateChanges = append(stateChanges, models.StateChange{
			Description: fmt.Sprintf("Concentrating on %s", source.Spell.Name),
		})
	}

	// Build response
	resp := &models.ActionResponse{
		StateChanges: stateChanges,
		Summary:      fmt.Sprintf("%s casts %s", actorName, source.Spell.Name),
	}

	if slotLevel > 0 && !source.IsInnate {
		resp.Summary += fmt.Sprintf(" (level %d slot)", slotLevel)
	} else if source.IsAtWill {
		resp.Summary += " (at will)"
	}

	// Resolve spell mechanics if we have a definition
	if spellDef != nil {
		resolveNpcSpellMechanics(ctx, uc, cmd, source, spellDef, slotLevel, ed, resp, userID)
	}

	// Apply damage/healing/conditions to targets
	targetIDs := resolveTargetIDs(cmd)

	skipDamage := false
	if resp.Hit != nil && !*resp.Hit {
		skipDamage = true
	}
	if spellDef != nil && spellDef.Resolution.Type == "attack" && resp.Hit == nil {
		skipDamage = true
	}

	// Apply damage to each target
	if !skipDamage && len(targetIDs) > 0 && len(resp.DamageRolls) > 0 {
		totalDamage := sumDamageRolls(resp.DamageRolls)

		for _, targetID := range targetIDs {
			target, _, tErr := ed.FindParticipantByInstanceID(targetID)
			if tErr != nil {
				l.UsecasesWarn(tErr, userID, map[string]any{"targetID": targetID, "phase": "npc_spell_damage"})
				continue
			}

			targetDamage := totalDamage

			// Multi-target save spells: roll individual save per target
			if len(targetIDs) > 1 && spellDef != nil && spellDef.Resolution.Type == "save" {
				ts, tsErr := loadTargetStats(ctx, uc, target)
				if tsErr == nil {
					perTargetSave := resolveNpcSpellSave(source.SpellSaveDC, spellDef, ts, resp)

					adjustedTotal := 0
					for _, dr := range resp.DamageRolls {
						rollDmg := dr.Total
						if dr.FinalDamage != nil {
							rollDmg = *dr.FinalDamage
						}
						if perTargetSave != nil {
							if perTargetSave.noDamage {
								rollDmg = 0
							} else if perTargetSave.halfDamage {
								rollDmg = rollDmg / 2
							}
						}
						if dr.DamageType != "" && ts != nil {
							adjusted, _ := applyResistance(rollDmg, dr.DamageType, ts)
							rollDmg = adjusted
						}
						adjustedTotal += rollDmg
					}
					targetDamage = adjustedTotal

					// Multi-target conditions on failed save
					if perTargetSave != nil && !perTargetSave.saved && spellDef != nil {
						for _, eff := range spellDef.Effects {
							if eff.Condition != nil {
								durationStr := eff.Condition.Duration
								if durationStr == "" {
									durationStr = "until removed"
								}
								resp.ConditionApplied = append(resp.ConditionApplied, models.ConditionApplied{
									TargetID:  targetID,
									Condition: string(eff.Condition.Condition),
									Duration:  durationStr,
									SaveEnds:  eff.Condition.SaveEnds,
								})
							}
						}
					}
				}
			}

			if targetDamage > 0 {
				applyDamageToTarget(target, targetDamage)
				targetName := participantName(target)
				resp.StateChanges = append(resp.StateChanges, models.StateChange{
					TargetID:    targetID,
					HPDelta:     -targetDamage,
					Description: fmt.Sprintf("%s takes %d damage from %s", targetName, targetDamage, source.Spell.Name),
				})
				mutated = true
			}
		}
	}

	// Apply healing
	if len(resp.HealingRolls) > 0 {
		totalHealing := 0
		for _, hr := range resp.HealingRolls {
			totalHealing += hr.Total
		}

		if totalHealing > 0 {
			healTargets := targetIDs
			if cmd.TargetSelf || len(healTargets) == 0 {
				healTargets = []string{participant.InstanceID}
			}
			for _, targetID := range healTargets {
				target, _, tErr := ed.FindParticipantByInstanceID(targetID)
				if tErr != nil {
					continue
				}
				applyHealToParticipant(target, totalHealing)
				targetName := participantName(target)
				resp.StateChanges = append(resp.StateChanges, models.StateChange{
					TargetID:    targetID,
					HPDelta:     totalHealing,
					Description: fmt.Sprintf("%s heals %d HP from %s", targetName, totalHealing, source.Spell.Name),
				})
				mutated = true
			}
		}
	}

	// Apply conditions
	if len(resp.ConditionApplied) > 0 {
		for _, ca := range resp.ConditionApplied {
			target, _, tErr := ed.FindParticipantByInstanceID(ca.TargetID)
			if tErr != nil {
				continue
			}

			var condEffect *models.ConditionEffect
			if spellDef != nil {
				for _, eff := range spellDef.Effects {
					if eff.Condition != nil && string(eff.Condition.Condition) == ca.Condition {
						condEffect = eff.Condition
						break
					}
				}
			}

			if condEffect != nil {
				ac := buildActiveCondition(condEffect, participant.InstanceID, source.Spell.Name, source.SpellSaveDC, ca.TargetID)
				appendConditionToTarget(target, ac)
				mutated = true
			}
		}
	}

	// Persist encounter data
	if mutated {
		if pErr := persistEncounterData(ctx, uc, ed, encounterID); pErr != nil {
			l.UsecasesError(pErr, userID, map[string]any{"encounterID": encounterID})
			return nil, fmt.Errorf("persist encounter: %w", pErr)
		}
	}

	return resp, nil
}

// resolveNpcSpellMechanics resolves spell mechanics using NPC-specific spell stats.
func resolveNpcSpellMechanics(
	ctx context.Context,
	uc *actionsUsecases,
	cmd *models.ActionCommand,
	source *npcSpellSource,
	spellDef *models.SpellDefinition,
	slotLevel int,
	ed *EncounterData,
	resp *models.ActionResponse,
	userID int,
) {
	l := logger.FromContext(ctx)

	targetIDs := resolveTargetIDs(cmd)
	isSingleTarget := len(targetIDs) <= 1

	var ts *TargetStats
	if isSingleTarget && len(targetIDs) == 1 {
		target, _, tErr := ed.FindParticipantByInstanceID(targetIDs[0])
		if tErr == nil {
			loaded, lErr := loadTargetStats(ctx, uc, target)
			if lErr != nil {
				l.UsecasesWarn(lErr, userID, map[string]any{"targetID": targetIDs[0]})
			} else {
				ts = loaded
			}
		}
	}

	var saveRes *spellSaveResult
	isCrit := false

	switch spellDef.Resolution.Type {
	case "attack":
		resolveNpcSpellAttack(cmd, source.SpellAttackBonus, ts, resp)
		if resp.Hit != nil && !*resp.Hit {
			return
		}
		isCrit = resp.RollResult != nil && resp.RollResult.Natural == 20

	case "save":
		if isSingleTarget {
			saveRes = resolveNpcSpellSave(source.SpellSaveDC, spellDef, ts, resp)
			if saveRes != nil && saveRes.noDamage {
				return
			}
			if saveRes == nil && spellDef.Resolution.Save != nil {
				return
			}
		} else {
			if spellDef.Resolution.Save != nil {
				ability := strings.ToUpper(string(spellDef.Resolution.Save.Ability))
				resp.Summary += fmt.Sprintf(", DC %d %s save", source.SpellSaveDC, ability)
			}
		}
	}

	// Compute caster level for cantrip scaling
	casterLevel := source.CasterLevel
	if casterLevel == 0 {
		// Fallback: estimate from creature's CR (rough approximation)
		casterLevel = 1
	}

	// Process each spell effect
	for _, effect := range spellDef.Effects {
		if effect.Damage != nil {
			rollDamageEffect(effect.Damage, spellDef, slotLevel, casterLevel, isCrit, saveRes, ts, resp)
		}
		if effect.Healing != nil {
			rollHealingEffect(effect.Healing, spellDef, slotLevel, resp)
		}
		if effect.Condition != nil {
			resolveConditionEffect(effect.Condition, spellDef, saveRes, targetIDs, resp)
		}
	}
}

// resolveNpcSpellAttack handles attack-type spell resolution for NPCs.
func resolveNpcSpellAttack(
	cmd *models.ActionCommand,
	spellAttackBonus int,
	ts *TargetStats,
	resp *models.ActionResponse,
) {
	natural, total, rolls := dice.RollD20(spellAttackBonus, cmd.Advantage, cmd.Disadvantage)
	resp.RollResult = &models.ActionRollResult{
		Expression: fmt.Sprintf("1d20%+d", spellAttackBonus),
		Rolls:      rolls,
		Modifier:   spellAttackBonus,
		Total:      total,
		Natural:    natural,
	}

	if ts != nil {
		hit := natural != 1 && (natural == 20 || total >= ts.AC)
		resp.Hit = &hit

		if hit {
			resp.Summary += fmt.Sprintf(", %d to hit vs AC %d — HIT", total, ts.AC)
			if natural == 20 {
				resp.Summary += " (CRITICAL!)"
			}
		} else {
			resp.Summary += fmt.Sprintf(", %d to hit vs AC %d — MISS", total, ts.AC)
		}
	} else {
		resp.Summary += fmt.Sprintf(", %d to hit", total)
	}
}

// resolveNpcSpellSave handles save-type spell resolution for NPCs.
func resolveNpcSpellSave(
	spellSaveDC int,
	spellDef *models.SpellDefinition,
	ts *TargetStats,
	resp *models.ActionResponse,
) *spellSaveResult {
	if spellDef.Resolution.Save == nil {
		return nil
	}

	ability := strings.ToLower(string(spellDef.Resolution.Save.Ability))

	if ts == nil {
		resp.Summary += fmt.Sprintf(", DC %d %s save",
			spellSaveDC, strings.ToUpper(ability))
		return nil
	}

	saveBonus := ts.SaveBonuses[ability]
	saveNatural, saveTotal, saveRolls := dice.RollD20(saveBonus, false, false)
	saved := saveTotal >= spellSaveDC

	resp.Summary += fmt.Sprintf(", DC %d %s save: %s rolls %d (%d%+d)",
		spellSaveDC, strings.ToUpper(ability), ts.Name, saveTotal, saveNatural, saveBonus)

	onSuccess := spellDef.Resolution.Save.OnSuccess
	result := &spellSaveResult{saved: saved}

	if saved {
		if onSuccess == "half" {
			resp.Summary += " — SAVES (half damage)"
			result.halfDamage = true
		} else {
			resp.Summary += " — SAVES (no effect)"
			result.noDamage = true
		}
	} else {
		resp.Summary += " — FAILS"
	}

	resp.StateChanges = append(resp.StateChanges, models.StateChange{
		Description: fmt.Sprintf("%s %s save: %d (1d20%+d = [%s])",
			ts.Name, strings.ToUpper(ability), saveTotal, saveBonus, formatRolls(saveRolls)),
	})

	return result
}
