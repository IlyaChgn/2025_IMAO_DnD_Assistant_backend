package usecases

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/dice"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
)

// resolveSpellCast handles the "spell_cast" action type.
// Validates spell knowledge, deducts a spell slot (or pact slot, or ritual),
// handles concentration, resolves mechanics (damage, healing, conditions),
// and applies effects to all targets.
func resolveSpellCast(
	ctx context.Context,
	uc *actionsUsecases,
	cmd *models.ActionCommand,
	encounterID string,
	charBase *models.CharacterBase,
	derived *models.DerivedStats,
	participant *models.ParticipantFull,
	ed *EncounterData,
	userID int,
) (*models.ActionResponse, error) {
	l := logger.FromContext(ctx)

	if cmd.SpellID == "" {
		return nil, apperrors.MissingSpellIDErr
	}

	// Verify the character has spellcasting
	if charBase.Spellcasting == nil {
		return nil, apperrors.SpellNotKnownErr
	}

	// Find spell in character's known/prepared spells
	spellRef := findSpellRef(charBase, cmd.SpellID)
	if spellRef == nil {
		return nil, apperrors.SpellNotKnownErr
	}

	// Load spell definition (if spells repo is available)
	var spellDef *models.SpellDefinition
	if uc.spellsRepo != nil {
		var err error
		spellDef, err = uc.spellsRepo.GetSpellByID(ctx, cmd.SpellID)
		if err != nil {
			l.UsecasesWarn(err, userID, map[string]any{"spellID": cmd.SpellID})
			// Continue without definition — we can still deduct slots
		}
	}

	runtime := participant.CharacterRuntime
	if runtime == nil {
		return nil, apperrors.ParticipantNotFoundErr
	}

	var stateChanges []models.StateChange
	mutated := false
	slotLevel := cmd.SlotLevel

	// Upcast validation: slot level must be >= spell level
	// Skip for pact slots — their level is reassigned later from PactMagic.SlotLevel
	if spellDef != nil && slotLevel > 0 && slotLevel < spellDef.Level && !cmd.IsPactSlot {
		return nil, apperrors.InvalidUpcastLevelErr
	}

	// Auto-resolve self-targeting
	if spellDef != nil && spellDef.Targeting.Type == models.TargetSelf &&
		cmd.TargetID == "" && len(cmd.TargetIDs) == 0 {
		cmd.TargetSelf = true
		cmd.TargetID = participant.InstanceID
	}

	// Slot deduction (cantrips and rituals skip this)
	if slotLevel > 0 && !cmd.IsRitual {
		if derived.Spellcasting == nil {
			return nil, apperrors.InsufficientSlotsErr
		}

		if cmd.IsPactSlot {
			// Pact Magic slot deduction (warlock)
			pm := derived.Spellcasting.PactMagic
			if pm == nil {
				return nil, apperrors.InsufficientPactSlotsErr
			}
			available := pm.MaxSlots - runtime.SpentPactSlots
			if available <= 0 {
				return nil, apperrors.InsufficientPactSlotsErr
			}
			runtime.SpentPactSlots++
			slotLevel = pm.SlotLevel // warlock auto-upcast to pact slot level
			mutated = true
			stateChanges = append(stateChanges, models.StateChange{
				SlotSpent:   slotLevel,
				Description: fmt.Sprintf("Spent a pact magic slot (level %d, %d/%d remaining)", slotLevel, available-1, pm.MaxSlots),
			})
		} else {
			// Normal spell slot deduction
			maxSlots := derived.Spellcasting.MaxSpellSlots[slotLevel]
			if runtime.SpentSpellSlots == nil {
				runtime.SpentSpellSlots = make(map[int]int)
			}
			spent := runtime.SpentSpellSlots[slotLevel]

			if spent >= maxSlots {
				return nil, apperrors.InsufficientSlotsErr
			}

			runtime.SpentSpellSlots[slotLevel]++
			mutated = true
			stateChanges = append(stateChanges, models.StateChange{
				SlotSpent:   slotLevel,
				Description: fmt.Sprintf("Spent a level %d spell slot (%d/%d remaining)", slotLevel, maxSlots-spent-1, maxSlots),
			})
		}
	}

	// Ritual casting validation
	if cmd.IsRitual {
		if spellDef == nil || !spellDef.Ritual {
			return nil, apperrors.SpellNotRitualErr
		}
		// Rituals always cast at base spell level — no free upcasting
		slotLevel = spellDef.Level
		stateChanges = append(stateChanges, models.StateChange{
			Description: fmt.Sprintf("Cast %s as a ritual (no slot spent)", spellRef.Name),
		})
	}

	// Build response before Counterspell check (concentration handled after).
	resp := &models.ActionResponse{
		StateChanges: stateChanges,
		Summary:      fmt.Sprintf("%s casts %s", charBase.Name, spellRef.Name),
	}

	if cmd.IsRitual {
		resp.Summary += " (ritual)"
	} else if slotLevel > 0 {
		resp.Summary += fmt.Sprintf(" (level %d slot)", slotLevel)
	}

	// Evaluate Counterspell reaction against PC spell.
	// Must happen BEFORE concentration swap — a countered spell never takes effect,
	// so the caster should keep their existing concentration.
	// Rituals cannot be counterspelled (D&D 5e: casting time is 10+ minutes).
	if uc.reactionEval != nil && spellDef != nil && spellDef.Level > 0 && !cmd.IsRitual {
		csResult, csErr := uc.reactionEval.EvaluateCounterspell(ctx, ed, participant.InstanceID, slotLevel)
		if csErr != nil {
			l.UsecasesWarn(csErr, userID, map[string]any{"casterID": participant.InstanceID, "reaction": "counterspell"})
		}
		if csErr == nil && csResult != nil {
			applyCounterspellReaction(csResult, ed)
			resp.ReactionSummary = append(resp.ReactionSummary, buildCounterspellSummary(csResult))
			mutated = true
			if csResult.Success {
				resp.Summary += fmt.Sprintf(" — COUNTERED by %s!", csResult.ReactorName)
				if pErr := persistEncounterData(ctx, uc, ed, encounterID); pErr != nil {
					l.UsecasesError(pErr, userID, map[string]any{"encounterID": encounterID})
					return nil, fmt.Errorf("persist encounter: %w", pErr)
				}
				return resp, nil
			}
		}
	}

	// Handle concentration (after Counterspell — countered spells don't swap concentration).
	if spellDef != nil && spellDef.Concentration {
		if runtime.Concentration != nil {
			resp.StateChanges = append(resp.StateChanges, models.StateChange{
				Description: fmt.Sprintf("Dropped concentration on %s", runtime.Concentration.SpellName),
			})
		}

		runtime.Concentration = &models.CharacterConcentration{
			SpellID:   cmd.SpellID,
			SpellName: spellRef.Name,
		}
		mutated = true
		resp.StateChanges = append(resp.StateChanges, models.StateChange{
			Description: fmt.Sprintf("Concentrating on %s", spellRef.Name),
		})
	}

	if spellDef != nil {
		if resolveSpellMechanics(ctx, uc, cmd, charBase, derived, spellDef, slotLevel, ed, resp, userID) {
			mutated = true
		}
	}

	// Apply effects to targets
	targetIDs := resolveTargetIDs(cmd)

	// Skip damage application when attack missed or couldn't determine hit
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
				l.UsecasesWarn(tErr, userID, map[string]any{"targetID": targetID, "phase": "damage"})
				continue
			}

			targetDamage := totalDamage

			// Multi-target save spells: roll individual save per target
			if len(targetIDs) > 1 && spellDef != nil && spellDef.Resolution.Type == "save" {
				ts, tsErr := loadTargetStats(ctx, uc, target)
				if tsErr == nil {
					perTargetSave := resolveSpellSave(derived, spellDef, ts, resp)

					// Apply save + per-type resistance for each damage roll
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

					// Multi-target conditions: apply based on individual save
					if perTargetSave != nil && !perTargetSave.saved {
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
					Description: fmt.Sprintf("%s takes %d damage from %s", targetName, targetDamage, spellRef.Name),
				})
				mutated = true
			}
		}
	}

	// Apply healing to targets
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
					l.UsecasesWarn(tErr, userID, map[string]any{"targetID": targetID, "phase": "healing"})
					continue
				}
				applyHealToParticipant(target, totalHealing)
				targetName := participantName(target)
				resp.StateChanges = append(resp.StateChanges, models.StateChange{
					TargetID:    targetID,
					HPDelta:     totalHealing,
					Description: fmt.Sprintf("%s heals %d HP from %s", targetName, totalHealing, spellRef.Name),
				})
				mutated = true
			}
		}
	}

	// Apply conditions to targets
	if len(resp.ConditionApplied) > 0 {
		for _, ca := range resp.ConditionApplied {
			target, _, tErr := ed.FindParticipantByInstanceID(ca.TargetID)
			if tErr != nil {
				l.UsecasesWarn(tErr, userID, map[string]any{"targetID": ca.TargetID, "phase": "condition"})
				continue
			}

			dc := 0
			if derived.Spellcasting != nil {
				dc = derived.Spellcasting.SpellSaveDC
			}

			// Find the matching condition effect from spell definition
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
				ac := buildActiveCondition(condEffect, participant.InstanceID, spellRef.Name, dc, ca.TargetID)
				appendConditionToTarget(target, ac)
				mutated = true
			}
		}
	}

	// Persist encounter data only if state was mutated
	if mutated {
		if err := persistEncounterData(ctx, uc, ed, encounterID); err != nil {
			l.UsecasesError(err, userID, map[string]any{"encounterID": encounterID})
			return nil, fmt.Errorf("persist encounter: %w", err)
		}
	}

	return resp, nil
}

func findSpellRef(charBase *models.CharacterBase, spellID string) *models.SpellRef {
	sc := charBase.Spellcasting

	for i := range sc.CantripsKnown {
		if sc.CantripsKnown[i].SpellID == spellID {
			return &sc.CantripsKnown[i]
		}
	}
	for i := range sc.SpellsKnown {
		if sc.SpellsKnown[i].SpellID == spellID {
			return &sc.SpellsKnown[i]
		}
	}
	for i := range sc.PreparedSpells {
		if sc.PreparedSpells[i].SpellID == spellID {
			return &sc.PreparedSpells[i]
		}
	}
	for i := range sc.AlwaysPrepared {
		if sc.AlwaysPrepared[i].SpellID == spellID {
			return &sc.AlwaysPrepared[i]
		}
	}
	for i := range sc.Spellbook {
		if sc.Spellbook[i].SpellID == spellID {
			return &sc.Spellbook[i]
		}
	}

	return nil
}

// resolveTargetIDs unifies legacy TargetID and multi-target TargetIDs,
// deduplicating while preserving order.
func resolveTargetIDs(cmd *models.ActionCommand) []string {
	var ids []string
	if len(cmd.TargetIDs) > 0 {
		ids = cmd.TargetIDs
	} else if cmd.TargetID != "" {
		return []string{cmd.TargetID}
	} else {
		return nil
	}

	seen := make(map[string]struct{}, len(ids))
	deduped := make([]string, 0, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; !ok {
			seen[id] = struct{}{}
			deduped = append(deduped, id)
		}
	}
	return deduped
}

// casterLevel computes the total character level as the sum of all class levels.
func casterLevel(charBase *models.CharacterBase) int {
	total := 0
	for _, c := range charBase.Classes {
		total += c.Level
	}
	return total
}

// resolveCantripDice finds the appropriate cantrip damage tier for the given caster level.
// Returns (diceCount, diceType) from the highest tier with MinLevel <= charLevel.
func resolveCantripDice(spellDef *models.SpellDefinition, charLevel int) (int, string) {
	if spellDef.CantripScaling == nil || len(spellDef.CantripScaling.DamageDice) == 0 {
		return 0, ""
	}

	var best *models.CantripScalingTier
	for i := range spellDef.CantripScaling.DamageDice {
		tier := &spellDef.CantripScaling.DamageDice[i]
		if tier.MinLevel <= charLevel {
			if best == nil || tier.MinLevel > best.MinLevel {
				best = tier
			}
		}
	}

	if best == nil {
		return 0, ""
	}
	return best.DiceCount, best.DiceType
}

// resolveUpcastDamage finds the best upcast damage scaling entry for the given slot level.
// Returns the extra damage dice to add on top of base damage, or nil if no upcast scaling.
func resolveUpcastDamage(spellDef *models.SpellDefinition, slotLevel int) *models.DamageRoll {
	if spellDef.Upcast == nil || len(spellDef.Upcast.Scaling) == 0 {
		return nil
	}
	if slotLevel <= spellDef.Level {
		return nil
	}

	var best *models.UpcastScaling
	for i := range spellDef.Upcast.Scaling {
		entry := &spellDef.Upcast.Scaling[i]
		if entry.Level <= slotLevel && entry.Damage != nil {
			if best == nil || entry.Level > best.Level {
				best = entry
			}
		}
	}

	if best == nil {
		return nil
	}

	// Scale the extra dice by how many levels above the threshold
	levelsAbove := slotLevel - spellDef.Level
	scaled := *best.Damage
	scaled.DiceCount = best.Damage.DiceCount * levelsAbove
	return &scaled
}

// resolveUpcastHealing returns extra healing dice and flat bonus for upcasting.
func resolveUpcastHealing(spellDef *models.SpellDefinition, slotLevel int) (extraDice int, flatBonus int) {
	if spellDef.Upcast == nil || len(spellDef.Upcast.Scaling) == 0 {
		return 0, 0
	}
	if slotLevel <= spellDef.Level {
		return 0, 0
	}

	var best *models.UpcastScaling
	for i := range spellDef.Upcast.Scaling {
		entry := &spellDef.Upcast.Scaling[i]
		if entry.Level <= slotLevel && (entry.HealingAdd > 0 || entry.HealingAddFlat > 0) {
			if best == nil || entry.Level > best.Level {
				best = entry
			}
		}
	}

	if best == nil {
		return 0, 0
	}

	levelsAbove := slotLevel - spellDef.Level
	return best.HealingAdd * levelsAbove, best.HealingAddFlat * levelsAbove
}

// buildActiveCondition maps a ConditionEffect to a runtime ActiveCondition.
// targetID is included in the ID to ensure uniqueness across multiple targets.
func buildActiveCondition(cond *models.ConditionEffect, casterID string, spellName string, dc int, targetID string) models.ActiveCondition {
	ac := models.ActiveCondition{
		ID:        fmt.Sprintf("spell_%s_%s_%s", spellName, string(cond.Condition), targetID),
		Condition: cond.Condition,
		SourceID:  casterID,
	}

	// Map duration string to DurationType
	dur := strings.ToLower(cond.Duration)
	switch {
	case strings.Contains(dur, "until saved") || strings.Contains(dur, "until save"):
		ac.Duration = models.DurationUntilSave
	case strings.Contains(dur, "concentration"):
		ac.Duration = models.DurationConcentration
	case strings.Contains(dur, "1 minute"):
		ac.Duration = models.DurationRounds
		ac.RoundsLeft = 10
	case strings.Contains(dur, "end of next turn"):
		ac.Duration = models.DurationUntilTurn
		ac.EndsOnTurn = "end"
	case strings.Contains(dur, "start of next turn"):
		ac.Duration = models.DurationUntilTurn
		ac.EndsOnTurn = "start"
	default:
		// Try to parse "N minute(s)" or "N round(s)"
		if rounds := parseDurationRounds(dur); rounds > 0 {
			ac.Duration = models.DurationRounds
			ac.RoundsLeft = rounds
		} else {
			ac.Duration = models.DurationPermanent
		}
	}

	// Set up save-to-end if applicable
	if cond.SaveEnds && cond.SaveAbility != "" {
		saveDC := dc
		if cond.EscapeDC > 0 {
			saveDC = cond.EscapeDC
		}
		ac.SaveToEnd = &models.SaveToEndCondition{
			Ability: cond.SaveAbility,
			DC:      saveDC,
			Timing:  "end_of_turn",
		}
	}

	// Set up escape for grapple/restrain
	if cond.EscapeDC > 0 {
		ac.EscapeDC = cond.EscapeDC
		ac.EscapeType = cond.EscapeType
	}

	return ac
}

// parseDurationRounds tries to extract rounds from a duration string like "10 minutes", "1 hour".
// Returns the number of rounds (1 round = 6 seconds). Returns 0 if unparseable.
func parseDurationRounds(dur string) int {
	dur = strings.TrimSpace(strings.ToLower(dur))

	// Try "N round(s)"
	if n, ok := extractNumber(dur, "round"); ok {
		return n
	}
	// "N minute(s)" -> N*10 rounds
	if n, ok := extractNumber(dur, "minute"); ok {
		return n * 10
	}
	// "N hour(s)" -> N*600 rounds
	if n, ok := extractNumber(dur, "hour"); ok {
		return n * 600
	}
	return 0
}

// extractNumber parses "N unit" or "N units" from a string.
func extractNumber(s string, unit string) (int, bool) {
	// Match "N unit" or "N units"
	for _, suffix := range []string{unit + "s", unit} {
		if strings.Contains(s, suffix) {
			parts := strings.Fields(s)
			for i, p := range parts {
				if strings.HasPrefix(p, unit) && i > 0 {
					n, err := strconv.Atoi(parts[i-1])
					if err == nil {
						return n, true
					}
				}
			}
		}
	}
	return 0, false
}

// appendConditionToTarget adds an ActiveCondition to the appropriate condition list.
func appendConditionToTarget(target *models.ParticipantFull, ac models.ActiveCondition) {
	if target.CharacterRuntime != nil {
		// PC target: use ConditionInstance
		ci := models.ConditionInstance{
			ID:   ac.ID,
			Type: ac.Condition,
			Duration: models.ConditionDuration{
				Type: string(ac.Duration),
			},
		}
		if ac.Duration == models.DurationRounds {
			ci.Duration.Remaining = ac.RoundsLeft
		}
		if ac.Duration == models.DurationConcentration {
			ci.Duration.CasterID = ac.SourceID
		}
		if ac.SaveToEnd != nil {
			ci.SaveRetry = &models.SaveRetry{
				Timing:          ac.SaveToEnd.Timing,
				DC:              ac.SaveToEnd.DC,
				Ability:         string(ac.SaveToEnd.Ability),
				SuccessesNeeded: 1,
			}
		}
		ci.SourceCreatureID = ac.SourceID
		target.CharacterRuntime.Conditions = append(target.CharacterRuntime.Conditions, ci)
		return
	}

	// Creature target: use ActiveCondition directly
	target.RuntimeState.Conditions = append(target.RuntimeState.Conditions, ac)
}

// spellSaveResult tracks the outcome of a target's saving throw.
type spellSaveResult struct {
	saved      bool
	halfDamage bool // true when target saved and spell has onSuccess="half"
	noDamage   bool // true when target saved and spell has onSuccess="none"
}

// resolveSpellMechanics adds roll results based on the spell definition's resolution type.
// Handles damage (with upcast and cantrip scaling), healing, and conditions.
// Returns true if encounter data was mutated (e.g., Shield reaction fired).
func resolveSpellMechanics(
	ctx context.Context,
	uc *actionsUsecases,
	cmd *models.ActionCommand,
	charBase *models.CharacterBase,
	derived *models.DerivedStats,
	spellDef *models.SpellDefinition,
	slotLevel int,
	ed *EncounterData,
	resp *models.ActionResponse,
	userID int,
) bool {
	l := logger.FromContext(ctx)

	// Load target stats for single-target resolution
	// (multi-target saves are handled per-target in resolveSpellCast)
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
	mechanicsMutated := false

	switch spellDef.Resolution.Type {
	case "attack":
		resolveSpellAttack(cmd, derived, ts, resp)

		// Shield reaction for spell attacks against NPC targets.
		// Skip on natural 20 (auto-hit) and natural 1 (auto-miss) — Shield has no effect.
		if uc.reactionEval != nil && isSingleTarget && len(targetIDs) == 1 && resp.RollResult != nil && resp.RollResult.Natural != 20 && resp.RollResult.Natural != 1 {
			target, _, tErr := ed.FindParticipantByInstanceID(targetIDs[0])
			if tErr == nil && !target.IsPlayerCharacter {
				shieldResult, sErr := uc.reactionEval.EvaluateShield(ctx, ed, targetIDs[0], resp.RollResult.Total)
				if sErr != nil {
					l.UsecasesWarn(sErr, userID, map[string]any{"targetID": targetIDs[0], "reaction": "shield"})
				}
				if sErr == nil && shieldResult != nil {
					applyShieldReaction(shieldResult, ed)
					resp.ReactionSummary = append(resp.ReactionSummary, buildShieldSummary(shieldResult))
					mechanicsMutated = true
					// Re-evaluate hit with new AC.
					if ts != nil {
						oldAC := ts.AC
						ts.AC = shieldResult.NewEffectiveAC
						natural := resp.RollResult.Natural
						hit := natural != 1 && (natural == 20 || resp.RollResult.Total >= ts.AC)
						resp.Hit = &hit
						// Fix Summary: replace stale "HIT" + old AC with "MISS" + new AC.
						if !hit {
							oldSuffix := fmt.Sprintf(", %d to hit vs AC %d — HIT", resp.RollResult.Total, oldAC)
							newSuffix := fmt.Sprintf(", %d to hit vs AC %d — MISS", resp.RollResult.Total, ts.AC)
							resp.Summary = strings.Replace(resp.Summary, oldSuffix, newSuffix, 1)
						}
					}
				}
			}
		}

		if resp.Hit != nil && !*resp.Hit {
			return mechanicsMutated // Miss — skip damage
		}
		isCrit = resp.RollResult != nil && resp.RollResult.Natural == 20

	case "save":
		// Only resolve save here for single-target; multi-target saves are per-target in resolveSpellCast
		if isSingleTarget {
			saveRes = resolveSpellSave(derived, spellDef, ts, resp)
			if saveRes != nil && saveRes.noDamage {
				// For save-none spells, conditions don't apply on success either
				return false
			}
			if saveRes == nil && spellDef.Resolution.Save != nil {
				return false
			}
		} else {
			// Multi-target: announce DC in summary, per-target saves handled in resolveSpellCast
			if derived.Spellcasting != nil && spellDef.Resolution.Save != nil {
				ability := strings.ToUpper(string(spellDef.Resolution.Save.Ability))
				resp.Summary += fmt.Sprintf(", DC %d %s save", derived.Spellcasting.SpellSaveDC, ability)
			}
		}
	}

	// Compute cantrip scaling
	charLvl := casterLevel(charBase)

	// Process each spell effect
	for _, effect := range spellDef.Effects {
		// Damage
		if effect.Damage != nil {
			rollDamageEffect(effect.Damage, spellDef, slotLevel, charLvl, isCrit, saveRes, ts, resp)
		}

		// Healing
		if effect.Healing != nil {
			rollHealingEffect(effect.Healing, spellDef, slotLevel, resp)
		}

		// Conditions
		if effect.Condition != nil {
			resolveConditionEffect(effect.Condition, spellDef, saveRes, targetIDs, resp)
		}
	}
	return mechanicsMutated
}

// rollDamageEffect rolls damage for a single spell damage effect,
// applying cantrip scaling, upcast scaling, crit, save-half, and resistance.
func rollDamageEffect(
	spellDmg *models.SpellDamage,
	spellDef *models.SpellDefinition,
	slotLevel int,
	charLvl int,
	isCrit bool,
	saveRes *spellSaveResult,
	ts *TargetStats,
	resp *models.ActionResponse,
) {
	dmg := spellDmg.Base
	diceCount := dmg.DiceCount
	diceType := strings.TrimPrefix(dmg.DiceType, "d")
	bonus := dmg.Bonus

	// Cantrip scaling: replace base dice with scaled values
	if spellDef.Level == 0 && spellDmg.CantripsScale && spellDef.CantripScaling != nil {
		scaledCount, scaledType := resolveCantripDice(spellDef, charLvl)
		if scaledCount > 0 && scaledType != "" {
			diceCount = scaledCount
			diceType = strings.TrimPrefix(scaledType, "d")
		}
	}

	if diceCount <= 0 || diceType == "" {
		return
	}

	// Build base expression
	expr := fmt.Sprintf("%dd%s", diceCount, diceType)
	if bonus != 0 {
		expr = fmt.Sprintf("%s%+d", expr, bonus)
	}
	result, err := dice.Roll(expr)
	if err != nil {
		return
	}

	rollResult := models.ActionRollResult{
		Expression: expr,
		Rolls:      result.Rolls,
		Modifier:   result.Modifier,
		Total:      result.Total,
		DamageType: dmg.DamageType,
	}

	finalDamage := result.Total

	// Upcast damage scaling — track dice for crit doubling
	var upcastDiceCount int
	var upcastSides string
	if slotLevel > spellDef.Level {
		upcastDmg := resolveUpcastDamage(spellDef, slotLevel)
		if upcastDmg != nil && upcastDmg.DiceCount > 0 {
			upDiceType := strings.TrimPrefix(upcastDmg.DiceType, "d")
			if upDiceType == "" {
				upDiceType = diceType // inherit from base if not specified
			}
			upcastDiceCount = upcastDmg.DiceCount
			upcastSides = upDiceType
			upExpr := fmt.Sprintf("%dd%s", upcastDmg.DiceCount, upDiceType)
			upResult, upErr := dice.Roll(upExpr)
			if upErr == nil {
				rollResult.Rolls = append(rollResult.Rolls, upResult.Rolls...)
				rollResult.Total += upResult.Total
				rollResult.Expression += "+" + upExpr
				finalDamage += upResult.Total
			}
		}
	}

	// Double ALL dice on critical hit (base + upcast)
	if isCrit {
		// Double base dice
		sides, pErr := strconv.Atoi(diceType)
		if pErr == nil {
			critRolls, critTotal := dice.RollDice(diceCount, sides)
			rollResult.Rolls = append(rollResult.Rolls, critRolls...)
			rollResult.Total += critTotal
			finalDamage += critTotal
		}
		// Double upcast dice
		if upcastDiceCount > 0 {
			upSides, upErr := strconv.Atoi(upcastSides)
			if upErr == nil {
				critRolls, critTotal := dice.RollDice(upcastDiceCount, upSides)
				rollResult.Rolls = append(rollResult.Rolls, critRolls...)
				rollResult.Total += critTotal
				finalDamage += critTotal
			}
		}
	}

	// Apply half damage for successful save
	if saveRes != nil && saveRes.halfDamage {
		finalDamage = finalDamage / 2
	}

	// Apply resistance/vulnerability/immunity (single-target only; multi-target is per-target)
	if ts != nil && dmg.DamageType != "" {
		adjusted, appliedMod := applyResistance(finalDamage, dmg.DamageType, ts)
		rollResult.AppliedModifier = appliedMod
		rollResult.FinalDamage = intPtr(adjusted)
		finalDamage = adjusted
	} else {
		rollResult.FinalDamage = intPtr(finalDamage)
	}

	resp.DamageRolls = append(resp.DamageRolls, rollResult)
	resp.Summary += fmt.Sprintf(", %d %s damage", finalDamage, dmg.DamageType)
	if rollResult.AppliedModifier != "" && rollResult.AppliedModifier != "normal" {
		resp.Summary += fmt.Sprintf(" (%s)", rollResult.AppliedModifier)
	}
}

// rollHealingEffect rolls healing for a spell healing effect, including upcast scaling.
func rollHealingEffect(
	healing *models.HealingEffect,
	spellDef *models.SpellDefinition,
	slotLevel int,
	resp *models.ActionResponse,
) {
	totalHealing := 0
	diceType := strings.TrimPrefix(healing.DiceType, "d")

	var rollResult models.ActionRollResult

	if healing.DiceCount > 0 && diceType != "" {
		expr := fmt.Sprintf("%dd%s", healing.DiceCount, diceType)
		if healing.Bonus != 0 {
			expr = fmt.Sprintf("%s%+d", expr, healing.Bonus)
		}
		result, err := dice.Roll(expr)
		if err != nil {
			return
		}

		rollResult = models.ActionRollResult{
			Expression: expr,
			Rolls:      result.Rolls,
			Modifier:   result.Modifier,
			Total:      result.Total,
		}
		totalHealing = result.Total
	} else if healing.Bonus > 0 {
		// Flat-only healing (e.g., Heal spell: bonus=70, no dice)
		rollResult = models.ActionRollResult{
			Expression: fmt.Sprintf("%d", healing.Bonus),
			Total:      healing.Bonus,
		}
		totalHealing = healing.Bonus
	} else {
		return
	}

	// Upcast healing scaling
	if slotLevel > spellDef.Level {
		extraDice, flatBonus := resolveUpcastHealing(spellDef, slotLevel)

		if extraDice > 0 && diceType != "" {
			upExpr := fmt.Sprintf("%dd%s", extraDice, diceType)
			upResult, upErr := dice.Roll(upExpr)
			if upErr == nil {
				rollResult.Rolls = append(rollResult.Rolls, upResult.Rolls...)
				rollResult.Total += upResult.Total
				rollResult.Expression += "+" + upExpr
				totalHealing += upResult.Total
			}
		}

		if flatBonus > 0 {
			rollResult.Total += flatBonus
			totalHealing += flatBonus
		}
	}

	resp.HealingRolls = append(resp.HealingRolls, rollResult)
	resp.Summary += fmt.Sprintf(", heals %d HP", totalHealing)
}

// resolveConditionEffect determines if a condition should apply and adds it to the response.
func resolveConditionEffect(
	cond *models.ConditionEffect,
	spellDef *models.SpellDefinition,
	saveRes *spellSaveResult,
	targetIDs []string,
	resp *models.ActionResponse,
) {
	// Conditions apply when:
	// 1. auto resolution (Power Word Stun — no save)
	// 2. save-type spell and save was failed
	shouldApply := false
	if spellDef.Resolution.Type == "auto" {
		shouldApply = true
	}
	if spellDef.Resolution.Type == "save" && saveRes != nil && !saveRes.saved {
		shouldApply = true
	}

	if !shouldApply {
		return
	}

	durationStr := cond.Duration
	if durationStr == "" {
		durationStr = "until removed"
	}

	for _, targetID := range targetIDs {
		resp.ConditionApplied = append(resp.ConditionApplied, models.ConditionApplied{
			TargetID:  targetID,
			Condition: string(cond.Condition),
			Duration:  durationStr,
			SaveEnds:  cond.SaveEnds,
		})
	}
}

// sumDamageRolls totals the final damage from all damage rolls.
func sumDamageRolls(rolls []models.ActionRollResult) int {
	total := 0
	for _, dr := range rolls {
		if dr.FinalDamage != nil {
			total += *dr.FinalDamage
		} else {
			total += dr.Total
		}
	}
	return total
}

// resolveSpellAttack handles attack-type spell resolution.
func resolveSpellAttack(
	cmd *models.ActionCommand,
	derived *models.DerivedStats,
	ts *TargetStats,
	resp *models.ActionResponse,
) {
	if derived.Spellcasting == nil {
		return
	}

	attackBonus := derived.Spellcasting.SpellAttackBonus
	natural, total, rolls := dice.RollD20(attackBonus, cmd.Advantage, cmd.Disadvantage)
	resp.RollResult = &models.ActionRollResult{
		Expression: fmt.Sprintf("1d20%+d", attackBonus),
		Rolls:      rolls,
		Modifier:   attackBonus,
		Total:      total,
		Natural:    natural,
	}

	if ts != nil {
		// D&D 5e: nat 1 always misses, nat 20 always hits
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

// resolveSpellSave handles save-type spell resolution.
// Returns save result so the caller can decide on damage.
func resolveSpellSave(
	derived *models.DerivedStats,
	spellDef *models.SpellDefinition,
	ts *TargetStats,
	resp *models.ActionResponse,
) *spellSaveResult {
	if spellDef.Resolution.Save == nil || derived.Spellcasting == nil {
		return nil
	}

	dc := derived.Spellcasting.SpellSaveDC
	ability := strings.ToLower(string(spellDef.Resolution.Save.Ability))

	if ts == nil {
		resp.Summary += fmt.Sprintf(", DC %d %s save",
			dc, strings.ToUpper(ability))
		return nil
	}

	// Roll target's saving throw
	saveBonus := ts.SaveBonuses[ability]
	saveNatural, saveTotal, saveRolls := dice.RollD20(saveBonus, false, false)
	saved := saveTotal >= dc

	resp.Summary += fmt.Sprintf(", DC %d %s save: %s rolls %d (%d%+d)",
		dc, strings.ToUpper(ability), ts.Name, saveTotal, saveNatural, saveBonus)

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

	// Add save roll as state change for visibility
	resp.StateChanges = append(resp.StateChanges, models.StateChange{
		Description: fmt.Sprintf("%s %s save: %d (1d20%+d = [%s])",
			ts.Name, strings.ToUpper(ability), saveTotal, saveBonus, formatRolls(saveRolls)),
	})

	return result
}

// formatRolls converts a slice of ints to a comma-separated string.
func formatRolls(rolls []int) string {
	parts := make([]string, len(rolls))
	for i, r := range rolls {
		parts[i] = fmt.Sprintf("%d", r)
	}
	return strings.Join(parts, ", ")
}
