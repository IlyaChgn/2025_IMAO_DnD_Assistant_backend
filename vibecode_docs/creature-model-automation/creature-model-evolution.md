# Creature Model Evolution for Automation

> **Type:** Investigation Report
> **Status:** Implemented
> **Branch:** `feature/maps-api-align-spec`
> **Last updated:** 2026-02-06

## Executive Summary

This document describes the evolution of the `Creature` model from a text-based D&D statblock representation to a structured, automation-ready data model. The changes enable:

- **Fog of War / Vision system** — structured vision ranges (darkvision, blindsight)
- **Pathfinding** — structured movement speeds for terrain calculations
- **Combat automation** — machine-readable actions, conditions, resources
- **Spellcasting** — structured spell slots, innate spellcasting, spell effects

---

## A. Problem Statement

The original `Creature` model was designed for **display purposes** — storing D&D statblock data as it appears in source books. This worked for showing creature information but prevented automation:

| Problem | Impact |
|---------|--------|
| `Action.Value` is free-form text | Cannot auto-roll attacks or damage |
| `Speed` uses `interface{}` for value | Cannot calculate movement with difficult terrain |
| `Senses` uses string for PassivePerception | Cannot integrate with fog/vision system |
| No condition tracking | Cannot apply/track Frightened, Poisoned, etc. |
| No resource tracking | Cannot track spell slots, legendary actions |

---

## B. Original Model (Before)

### File: `internal/models/creature.go`

```go
type Speed struct {
    Value      interface{} `json:"value"`      // Could be int, string, or anything
    Name       string      `json:"name"`       // "walk", "fly", etc.
    Additional string      `json:"additional"` // Free-form notes
}

type Senses struct {
    PassivePerception string  `json:"passivePerception"` // "14" as string
    Sense             []Sense `json:"senses"`
}

type Action struct {
    Name  string `json:"name"`  // "Scimitar"
    Value string `json:"value"` // "Melee Weapon Attack: +4 to hit, reach 5 ft..."
}

type BonusAction struct {
    Name  string `json:"name"`
    Value string `json:"value"`
}

type Reaction struct {
    Name  string `json:"name"`
    Value string `json:"value"`
}
```

### Frontend Participant (before)

```typescript
type Participant = {
  _id: UUID;        // creature template ID
  id: UUID;         // instance ID
  initiative: number;
  cellsCoords?: CellsCoordinates;
  // No HP, no conditions, no resources
};
```

---

## C. New Model (After)

The changes are organized into 4 levels, each building on the previous:

### Level 1: Vision & Movement

New structured types for fog/vision and pathfinding integration.

**File:** `internal/models/creature.go`

```go
// CreatureMovement provides structured movement speeds in feet for pathfinding automation.
type CreatureMovement struct {
    Walk   int  `json:"walk" bson:"walk"`
    Fly    int  `json:"fly,omitempty" bson:"fly,omitempty"`
    Swim   int  `json:"swim,omitempty" bson:"swim,omitempty"`
    Climb  int  `json:"climb,omitempty" bson:"climb,omitempty"`
    Burrow int  `json:"burrow,omitempty" bson:"burrow,omitempty"`
    Hover  bool `json:"hover,omitempty" bson:"hover,omitempty"`
}

// CreatureVision provides structured vision ranges in feet for fog-of-war/lighting automation.
type CreatureVision struct {
    Darkvision  int `json:"darkvision,omitempty" bson:"darkvision,omitempty"`
    Blindsight  int `json:"blindsight,omitempty" bson:"blindsight,omitempty"`
    Truesight   int `json:"truesight,omitempty" bson:"truesight,omitempty"`
    Tremorsense int `json:"tremorsense,omitempty" bson:"tremorsense,omitempty"`
}
```

**Added to Creature:**
```go
type Creature struct {
    // ... existing fields ...
    Speed    []Speed          `json:"speed" bson:"speed"`              // Deprecated
    Movement CreatureMovement `json:"movement,omitempty" bson:"movement,omitempty"` // NEW

    Senses   Senses           `json:"senses" bson:"senses"`            // Deprecated
    Vision   CreatureVision   `json:"vision,omitempty" bson:"vision,omitempty"`     // NEW
}
```

### Level 2: Structured Actions

Machine-readable actions for combat automation.

**File:** `internal/models/action_structured.go`

```go
type StructuredAction struct {
    ID          string         `json:"id"`
    Name        string         `json:"name"`
    Description string         `json:"description"`     // Original text for display
    Category    ActionCategory `json:"category"`        // action/bonus_action/reaction/legendary

    Attack      *AttackRollData   `json:"attack,omitempty"`      // d20 vs AC
    SavingThrow *SavingThrowData  `json:"savingThrow,omitempty"` // DC-based effects
    Uses        *UsesData         `json:"uses,omitempty"`        // 3/day
    Recharge    *RechargeData     `json:"recharge,omitempty"`    // Recharge 5-6
    Effects     []ActionEffect    `json:"effects,omitempty"`     // Conditions, ongoing damage
}

type AttackRollData struct {
    Type    AttackRollType `json:"type"`    // melee_weapon, ranged_spell, etc.
    Bonus   int            `json:"bonus"`   // +7 to hit
    Reach   int            `json:"reach"`   // 5 ft
    Range   *RangeData     `json:"range"`   // 30/120 ft
    Targets int            `json:"targets"` // Usually 1
    Damage  []DamageRoll   `json:"damage"`  // 2d6+4 slashing
}

type SavingThrowData struct {
    Ability   AbilityType   `json:"ability"`   // DEX, CON, WIS
    DC        int           `json:"dc"`        // 15
    OnFail    string        `json:"onFail"`    // "full damage"
    OnSuccess string        `json:"onSuccess"` // "half damage"
    Damage    []DamageRoll  `json:"damage"`
    Area      *AreaOfEffect `json:"area"`      // 30-foot cone
}

type AreaOfEffect struct {
    Shape  AreaShape `json:"shape"`  // cone, sphere, line, cube, cylinder
    Size   int       `json:"size"`   // in feet
    Width  int       `json:"width"`  // for lines
    Origin string    `json:"origin"` // self, point
}

type ActionEffect struct {
    Condition     *ConditionEffect     `json:"condition,omitempty"`
    OngoingDamage *OngoingDamageEffect `json:"ongoingDamage,omitempty"`
    Movement      *MovementEffect      `json:"movement,omitempty"`  // push/pull
    Healing       *HealingEffect       `json:"healing,omitempty"`
}
```

**Constants defined:**
- `ActionCategory`: action, bonus_action, reaction, legendary, lair, free
- `AttackRollType`: melee_weapon, ranged_weapon, melee_spell, ranged_spell
- `AreaShape`: cone, cube, cylinder, line, sphere
- `ConditionType`: All 15 D&D 5e conditions (blinded, charmed, frightened, etc.)
- `AbilityType`: STR, DEX, CON, INT, WIS, CHA

### Level 3: Runtime Combat State

State tracking for creatures in an encounter (HP, conditions, resources).

**File:** `internal/models/creature_runtime.go`

```go
type CreatureRuntimeState struct {
    CurrentHP     int                `json:"currentHP"`
    MaxHP         int                `json:"maxHP"`
    TempHP        int                `json:"tempHP,omitempty"`
    Conditions    []ActiveCondition  `json:"conditions,omitempty"`
    Resources     ResourceState      `json:"resources,omitempty"`
    Concentration *ConcentrationState `json:"concentration,omitempty"`
    DeathSaves    *DeathSaveState    `json:"deathSaves,omitempty"`
    StatModifiers []StatModifier     `json:"statModifiers,omitempty"`
}

type ActiveCondition struct {
    ID           string        `json:"id"`
    Condition    ConditionType `json:"condition"`    // frightened, poisoned
    SourceID     string        `json:"sourceID"`     // who applied it
    Duration     DurationType  `json:"duration"`     // rounds, until_save, permanent
    RoundsLeft   int           `json:"roundsLeft"`
    SaveToEnd    *SaveToEndCondition `json:"saveToEnd,omitempty"`
    EscapeDC     int           `json:"escapeDC,omitempty"`  // for grapple
    Level        int           `json:"level,omitempty"`      // exhaustion 1-6
}

type ResourceState struct {
    SpellSlots           map[int]int    `json:"spellSlots,omitempty"`    // level -> remaining
    AbilityUses          map[string]int `json:"abilityUses,omitempty"`   // actionID -> remaining
    LegendaryActions     int            `json:"legendaryActions,omitempty"`
    LegendaryResistances int            `json:"legendaryResistances,omitempty"`
    RechargeReady        map[string]bool `json:"rechargeReady,omitempty"` // actionID -> ready
    ReactionUsed         bool           `json:"reactionUsed,omitempty"`
}

type ParticipantFull struct {
    CreatureID   string               `json:"_id"`
    InstanceID   string               `json:"id"`
    Initiative   int                  `json:"initiative"`
    CellsCoords  *CellsCoordinates    `json:"cellsCoords,omitempty"`
    DisplayName  string               `json:"displayName,omitempty"`
    OwnerID      string               `json:"ownerID,omitempty"`     // for per-player fog
    RuntimeState CreatureRuntimeState `json:"runtimeState,omitempty"`
    IsPlayerCharacter bool            `json:"isPlayerCharacter,omitempty"`
    Hidden       bool                 `json:"hidden,omitempty"`
}
```

**Duration types:** rounds, until_turn, until_save, permanent, concentration

**Stat modifiers** support: AC, attack_rolls, saving_throws, speed, ability scores
**Modifier operations:** add, multiply, set, advantage, disadvantage, dice_bonus

### Level 4: Spellcasting

Structured spellcasting for slot-based and innate casters.

**File:** `internal/models/spellcasting.go`

```go
type Spellcasting struct {
    Ability          AbilityType         `json:"ability"`          // INT, WIS, CHA
    SpellSaveDC      int                 `json:"spellSaveDC"`
    SpellAttackBonus int                 `json:"spellAttackBonus"`
    SpellSlots       map[int]int         `json:"spellSlots"`       // level -> count
    CasterLevel      int                 `json:"casterLevel"`
    SpellsByLevel    map[int][]SpellKnown `json:"spellsByLevel"`
}

type InnateSpellcasting struct {
    Ability          AbilityType         `json:"ability"`
    SpellSaveDC      int                 `json:"spellSaveDC"`
    AtWill           []SpellKnown        `json:"atWill"`           // unlimited
    PerDay           map[int][]SpellKnown `json:"perDay"`          // 3/day, 1/day
}

type SpellKnown struct {
    SpellID  string       `json:"spellID,omitempty"`
    Name     string       `json:"name"`
    Level    int          `json:"level"`
    School   SpellSchool  `json:"school,omitempty"`
    QuickRef *SpellQuickRef `json:"quickRef,omitempty"`
}

type Spell struct {
    ID          string          `json:"id"`
    Name        Name            `json:"name"`
    Level       int             `json:"level"`
    School      SpellSchool     `json:"school"`
    CastingTime CastingTime     `json:"castingTime"`
    Range       SpellRange      `json:"range"`
    Components  SpellComponents `json:"components"`
    Duration    SpellDuration   `json:"duration"`
    Description string          `json:"description"`
    Effects     []SpellEffect   `json:"effects,omitempty"` // For automation
}
```

**Spell schools:** abjuration, conjuration, divination, enchantment, evocation, illusion, necromancy, transmutation

**Spell effect triggers:** on_cast, on_hit, on_failed_save, start_of_turn, end_of_turn, on_enter, on_exit

---

## D. File Summary

| File | Contents |
|------|----------|
| `internal/models/creature.go` | Updated Creature with Movement, Vision fields; deprecated Speed, Senses, Action, BonusAction, Reaction |
| `internal/models/action_structured.go` | StructuredAction, AttackRollData, SavingThrowData, AreaOfEffect, ActionEffect, all condition/effect types |
| `internal/models/creature_runtime.go` | CreatureRuntimeState, ActiveCondition, ResourceState, StatModifier, ParticipantFull |
| `internal/models/spellcasting.go` | Spellcasting, InnateSpellcasting, SpellKnown, Spell, SpellEffect |

---

## E. Migration Strategy

### Phase 1: Parallel Fields (Current)

New structured fields exist alongside deprecated fields:
- `Speed []Speed` + `Movement CreatureMovement`
- `Senses Senses` + `Vision CreatureVision`
- `Actions []Action` + `StructuredActions []StructuredAction`

Frontend uses new fields when available, falls back to legacy.

### Phase 2: LLM Population

Update Gemini prompts to populate new structured fields when creating creatures.

### Phase 3: Data Migration

Write migration script to:
1. Parse legacy `Speed` → populate `Movement`
2. Parse legacy `Senses` → populate `Vision`
3. Optionally parse `Actions` text → populate `StructuredActions` (complex, may require LLM)

### Phase 4: Frontend Integration

1. Update `Participant` type to use `ParticipantFull` (or embed `RuntimeState`)
2. Implement condition tracker UI
3. Implement resource tracking UI
4. Integrate `Vision.Darkvision` with fog-of-war system

---

## F. Frontend Integration Points

| Frontend Feature | Backend Model | Integration |
|-----------------|---------------|-------------|
| Fog of War vision | `CreatureVision.Darkvision` | Pass to `computeVisionGrid()` |
| Pathfinding cost | `CreatureMovement.Walk` | Calculate movement with difficult terrain |
| Condition tracker | `ActiveCondition[]` | Display/edit in participant panel |
| HP tracking | `RuntimeState.CurrentHP/MaxHP` | HP bar component |
| Death saves | `RuntimeState.DeathSaves` | Death save UI at 0 HP |
| Spell slots | `RuntimeState.Resources.SpellSlots` | Spell slot tracker |
| Legendary actions | `RuntimeState.Resources.LegendaryActions` | Legendary action counter |
| Concentration | `RuntimeState.Concentration` | Concentration indicator |

---

## G. Example: Goblin (Structured)

```json
{
  "name": { "rus": "Гоблин", "eng": "Goblin" },
  "movement": { "walk": 30 },
  "vision": { "darkvision": 60 },
  "structuredActions": [
    {
      "id": "scimitar",
      "name": "Scimitar",
      "description": "Melee Weapon Attack: +4 to hit, reach 5 ft., one target. Hit: 5 (1d6 + 2) slashing damage.",
      "category": "action",
      "attack": {
        "type": "melee_weapon",
        "bonus": 4,
        "reach": 5,
        "targets": 1,
        "damage": [
          { "diceCount": 1, "diceType": "d6", "bonus": 2, "damageType": "slashing" }
        ]
      }
    },
    {
      "id": "shortbow",
      "name": "Shortbow",
      "description": "Ranged Weapon Attack: +4 to hit, range 80/320 ft., one target. Hit: 5 (1d6 + 2) piercing damage.",
      "category": "action",
      "attack": {
        "type": "ranged_weapon",
        "bonus": 4,
        "range": { "normal": 80, "long": 320 },
        "targets": 1,
        "damage": [
          { "diceCount": 1, "diceType": "d6", "bonus": 2, "damageType": "piercing" }
        ]
      }
    }
  ]
}
```

---

## H. Example: Adult Red Dragon (Structured)

```json
{
  "name": { "rus": "Взрослый красный дракон", "eng": "Adult Red Dragon" },
  "movement": { "walk": 40, "fly": 80, "climb": 40 },
  "vision": { "darkvision": 120, "blindsight": 60 },
  "structuredActions": [
    {
      "id": "fire-breath",
      "name": "Fire Breath",
      "description": "The dragon exhales fire in a 60-foot cone...",
      "category": "action",
      "recharge": { "minRoll": 5 },
      "savingThrow": {
        "ability": "DEX",
        "dc": 21,
        "onFail": "full damage",
        "onSuccess": "half damage",
        "damage": [
          { "diceCount": 18, "diceType": "d6", "bonus": 0, "damageType": "fire" }
        ],
        "area": { "shape": "cone", "size": 60, "origin": "self" }
      }
    }
  ],
  "spellcasting": null,
  "innateSpellcasting": {
    "ability": "CHA",
    "spellSaveDC": 19,
    "atWill": [
      { "name": "Detect Magic", "level": 1 }
    ],
    "perDay": {
      "3": [{ "name": "Suggestion", "level": 2 }],
      "1": [{ "name": "Wall of Fire", "level": 4 }]
    }
  }
}
```

---

## I. Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| Breaking existing creatures | Parallel fields; legacy fields remain functional |
| LLM parsing inconsistency | Validate and sanitize LLM output; fallback to text |
| Frontend Participant type mismatch | Gradual migration; ParticipantFull is superset |
| Large JSON payload | All new fields are `omitempty`; minimal overhead for simple creatures |

---

## J. Next Steps

1. [ ] Create helper functions: `SpeedToMovement()`, `SensesToVision()`
2. [ ] Update LLM prompts for creature generation
3. [ ] Write MongoDB migration script
4. [ ] Update frontend `Participant` type
5. [ ] Implement condition tracker component
6. [ ] Integrate `Vision.Darkvision` with fog system

---

## Changelog

| Date | Change |
|------|--------|
| 2026-02-06 | Initial document: Levels 1-4 implemented |
