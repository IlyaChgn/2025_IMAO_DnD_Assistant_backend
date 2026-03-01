# Creature Data Migration Rules

> **Type:** Implementation Notes
> **Status:** Draft
> **Branch:** `feature/maps-api-align-spec`
> **Last updated:** 2026-02-06

## Overview

This document defines the rules for migrating existing creature data from legacy unstructured fields to new structured fields. The migration is **additive** — legacy fields remain unchanged, new fields are populated alongside.

**Source database:** `bestiary_db.creatures` (1886 documents)

---

## A. Speed → Movement

### A.1 Source Format

```json
{
  "speed": [
    { "value": 30 },
    { "name": "летая", "value": 60, "additional": "парит" },
    { "name": "плавая", "value": 40 }
  ]
}
```

### A.2 Target Format

```json
{
  "movement": {
    "walk": 30,
    "fly": 60,
    "swim": 40,
    "hover": true
  }
}
```

### A.3 Mapping Rules

| Source `speed[].name` | Target Field | Count in DB |
|-----------------------|--------------|-------------|
| `null` / `undefined` / `""` | `walk` | ~1886 (first entry without name) |
| `"летая"` | `fly` | 463 |
| `"плавая"` | `swim` | 226 |
| `"лазая"` | `climb` | 200 |
| `"копая"` | `burrow` | 82 |

| Source `speed[].additional` | Target Field |
|-----------------------------|--------------|
| `"парит"` | `hover: true` | 130 |

### A.4 Value Type

- Source: `int32` (verified)
- Target: `int`
- No conversion needed

### A.5 Edge Cases

| Pattern | Count | Handling |
|---------|-------|----------|
| Complex strings in `name` (e.g., `"(40 фт. в облике волка)"`) | ~15 | **SKIP** — do not populate Movement, keep only legacy Speed |
| Multiple entries with same name | 0 | N/A |
| `value` as string | 0 | N/A (all are int32) |

### A.6 Algorithm

```
for each creature:
  movement = {}
  for each speed in creature.speed:
    if speed.name is null/empty:
      movement.walk = speed.value
    else if speed.name == "летая":
      movement.fly = speed.value
    else if speed.name == "плавая":
      movement.swim = speed.value
    else if speed.name == "лазая":
      movement.climb = speed.value
    else if speed.name == "копая":
      movement.burrow = speed.value
    else:
      // Complex string - skip this creature's Movement
      movement = null
      break

    if speed.additional == "парит":
      movement.hover = true

  if movement != null AND movement has at least walk:
    creature.movement = movement
```

### A.7 Validation

- [ ] `walk` should be > 0 for most creatures (except stationary ones)
- [ ] `fly` with `hover: true` should have ~130 creatures
- [ ] Total creatures with Movement populated: ~1870 (excluding edge cases)

---

## B. Senses → Vision

### B.1 Source Format

```json
{
  "senses": {
    "passivePerception": "13",
    "senses": [
      { "name": "тёмное зрение", "value": 60 },
      { "name": "слепое зрение", "value": 10, "additional": "слеп за пределами этого радиуса" }
    ]
  }
}
```

### B.2 Target Format

```json
{
  "vision": {
    "darkvision": 60,
    "blindsight": 10
  }
}
```

### B.3 Mapping Rules

| Source `senses.senses[].name` | Target Field | Count in DB |
|-------------------------------|--------------|-------------|
| `"тёмное зрение"` | `darkvision` | 1021 |
| `"слепое зрение"` | `blindsight` | 310 |
| `"истинное зрение"` | `truesight` | 156 |
| `"чувство вибрации"` | `tremorsense` | 37 |

### B.4 Edge Cases

| Pattern | Handling |
|---------|----------|
| `senses.senses` does not exist | `vision` = empty struct `{}` |
| `additional` field present | **IGNORE** — informational only |
| Multiple of same sense type | Take first occurrence |

### B.5 Algorithm

```
for each creature:
  vision = {}
  if creature.senses.senses exists:
    for each sense in creature.senses.senses:
      if sense.name == "тёмное зрение":
        vision.darkvision = sense.value
      else if sense.name == "слепое зрение":
        vision.blindsight = sense.value
      else if sense.name == "истинное зрение":
        vision.truesight = sense.value
      else if sense.name == "чувство вибрации":
        vision.tremorsense = sense.value

  creature.vision = vision  // even if empty
```

### B.6 Validation

- [ ] Creatures with darkvision: 1021
- [ ] Creatures with blindsight: 310
- [ ] Creatures with truesight: 156
- [ ] Creatures with tremorsense: 37
- [ ] Total creatures with at least one vision field: 1343

---

## C. llm_parsed_attack → StructuredActions

### C.1 Source Format

```json
{
  "llm_parsed_attack": [
    {
      "name": "Грязевое дыхание (перезарядка 6)",
      "type": "area",
      "attack_bonus": "+4",
      "reach": "5 фт.",
      "range": "30/120 фт.",
      "target": "одна цель",
      "damage": {
        "dice": "d6",
        "count": 2,
        "bonus": 3,
        "type": "колющий"
      },
      "save_dc": 15,
      "save_type": "Ловкости",
      "additional_effects": [
        {
          "condition": "отравленной",
          "duration": "1 минуту",
          "escape_dc": 13
        }
      ],
      "recharge": "5-6"
    }
  ]
}
```

### C.2 Target Format

```json
{
  "structuredActions": [
    {
      "id": "mud-breath",
      "name": "Грязевое дыхание",
      "description": "<original action text>",
      "category": "action",
      "recharge": { "minRoll": 6 },
      "savingThrow": {
        "ability": "DEX",
        "dc": 15,
        "onFail": "full effect",
        "onSuccess": "no effect"
      },
      "effects": [
        {
          "condition": {
            "condition": "poisoned",
            "duration": "1 minute",
            "saveEnds": false
          }
        }
      ]
    }
  ]
}
```

### C.3 Attack Type Mapping

| Source `type` | Count | Target `attack.type` |
|---------------|-------|----------------------|
| `"melee"` | 1682 | `melee_weapon` |
| `"ranged"` | 513 | `ranged_weapon` |
| `"area"` | 316 | (no attack, use savingThrow) |
| `"melee/ranged"` | 27 | `melee_weapon` (primary) |
| `"ranged/melee"` | 1 | `ranged_weapon` (primary) |
| `"ranged_or_melee"` | 3 | `melee_weapon` (primary) |
| `"versatile"` | 3 | `melee_weapon` |
| `"both"` | 1 | `melee_weapon` |
| `"either"` | 3 | `melee_weapon` |
| `"touch"` | 1 | `melee_spell` |
| `"none"` | 1 | (no attack field) |

### C.4 Damage Type Mapping

Source damage types are in Russian with various inflections. Normalize to English:

| Source Pattern (regex) | Target |
|------------------------|--------|
| `/дроб/i` | `bludgeoning` |
| `/кол[ющ]/i`, `/прокал/i`, `/проник/i` | `piercing` |
| `/руб/i` | `slashing` |
| `/огн/i`, `/огон/i` | `fire` |
| `/холод/i` | `cold` |
| `/электр/i`, `/молн/i` | `lightning` |
| `/кисл/i` | `acid` |
| `/яд/i` | `poison` |
| `/некрот/i` | `necrotic` |
| `/излуч/i`, `/свет/i` | `radiant` |
| `/псих/i` | `psychic` |
| `/звук/i`, `/гром/i` | `thunder` |
| `/сил[ов]/i` | `force` |

**Edge case:** Combined types like `"кислота, холод, огонь, молния или яд"` → `damageType: "varies"` or pick first.

### C.5 Save Type Mapping

| Source `save_type` | Target `ability` |
|--------------------|------------------|
| `"Силы"`, `"Сил"` | `STR` |
| `"Ловкости"`, `"Ловк"` | `DEX` |
| `"Телосложения"`, `"Тел"` | `CON` |
| `"Интеллекта"`, `"Инт"` | `INT` |
| `"Мудрости"`, `"Мудр"` | `WIS` |
| `"Харизмы"`, `"Хар"` | `CHA` |

### C.6 Condition Mapping

Source conditions have many inflections. Normalize using pattern matching:

| Source Pattern (regex) | Target `ConditionType` |
|------------------------|------------------------|
| `/слеп/i`, `/ослепл/i` | `blinded` |
| `/очаров/i` | `charmed` |
| `/оглох/i`, `/оглуш/i` | `deafened` |
| `/испуг/i`, `/напуг/i`, `/страх/i` | `frightened` |
| `/схвач/i`, `/захвач/i` | `grappled` |
| `/недееспособ/i` | `incapacitated` |
| `/невидим/i` | `invisible` |
| `/парализ/i` | `paralyzed` |
| `/окамен/i/ | `petrified` |
| `/отравл/i`, `/яд/i` (condition context) | `poisoned` |
| `/ничком/i`, `/сбит с ног/i` | `prone` |
| `/опутан/i`, `/удержив/i` | `restrained` |
| `/ошеломл/i/ | `stunned` |
| `/без сознан/i`, `/бессозн/i/, `/теряет сознан/i` | `unconscious` |
| `/истощ/i` | `exhaustion` |

### C.7 Recharge Parsing

Extract from action name:

| Source Pattern | Target `recharge.minRoll` |
|----------------|---------------------------|
| `(перезарядка 5–6)`, `(перезарядка 5-6)` | `5` |
| `(перезарядка 6)` | `6` |
| `(перезарядка 4–6)` | `4` |

Regex: `/\(перезарядка\s*(\d+)(?:[–-]6)?\)/i` → capture group 1 = minRoll

### C.8 Reach/Range Parsing

| Source | Target |
|--------|--------|
| `"5 фт."`, `"5 футов"` | `reach: 5` |
| `"10 фт."` | `reach: 10` |
| `"30/120 фт."` | `range: { normal: 30, long: 120 }` |
| `"80/320 фт."` | `range: { normal: 80, long: 320 }` |

Regex for reach: `/(\d+)\s*(?:фт|фут)/i`
Regex for range: `/(\d+)\/(\d+)\s*(?:фт|фут)/i`

### C.9 ID Generation

Generate stable ID from action name:
```
id = slugify(name_without_recharge)
   = "Грязевое дыхание (перезарядка 6)" → "gryazevoe-dyhanie"
   = "Короткий меч" → "korotkij-mech"
```

Or use transliteration + kebab-case.

### C.10 Category Detection

All `llm_parsed_attack` entries are `action` category by default.

For `bonusActions` and `reactions` — need separate migration from those arrays.

### C.11 Creatures Without llm_parsed_attack

5 creatures have `actions` but no `llm_parsed_attack`:
- Подружка невесты Заггтмой
- Вирмлинг медного дракона
- Б'рог
- Драконид манипулятор
- Истарианский дрон

**Handling:** SKIP — do not populate `structuredActions`, keep legacy `actions` only.

### C.12 Validation Checklist

- [ ] Total creatures with structuredActions: ~1852
- [ ] All damage types resolve to valid English type
- [ ] All save types resolve to valid ability
- [ ] Recharge parsed correctly from ~587 action names
- [ ] No null/undefined in required fields

---

## D. Migration Process

### D.1 Phases

| Phase | Description | Reversible |
|-------|-------------|------------|
| 1. Dry Run | Generate JSON diffs, no DB writes | N/A |
| 2. Review | Manual inspection of sample diffs | N/A |
| 3. Backup | `mongodump bestiary_db.creatures` | Yes |
| 4. Migrate | Run migration script with writes | Via restore |
| 5. Validate | Run validation queries | N/A |

### D.2 Dry Run Output

For each creature, output:
```json
{
  "_id": "...",
  "name": "Гоблин",
  "changes": {
    "movement": { "walk": 30 },
    "vision": { "darkvision": 60 },
    "structuredActions": [ ... ]
  },
  "skipped": {
    "movement": "complex speed string",
    "structuredActions": "no llm_parsed_attack"
  }
}
```

### D.3 Backup Location

**Backup created:** `2026-02-06 16:11:02`
**File:** `backups/creatures_backup_20260206_161102.json`
**Size:** 30.02 MB (1886 creatures)
**Format:** JSON array of all creature documents

### D.4 Rollback

To restore from backup, use the restore script:
```bash
go run ./cmd/restore_creatures/main.go -file=backups/creatures_backup_20260206_161102.json
```

Or manually via mongo shell / Compass.

---

## E. Examples

### E.1 Simple Creature (Goblin)

**Before:**
```json
{
  "name": { "rus": "Гоблин", "eng": "Goblin" },
  "speed": [{ "value": 30 }],
  "senses": { "passivePerception": "11", "senses": [{ "name": "тёмное зрение", "value": 60 }] },
  "actions": [{ "name": "Ятаган", "value": "<p>Рукопашная атака оружием...</p>" }],
  "llm_parsed_attack": [{
    "name": "Ятаган",
    "type": "melee",
    "attack_bonus": "+4",
    "reach": "5 фт.",
    "damage": { "dice": "d6", "count": 1, "bonus": 2, "type": "рубящий" }
  }]
}
```

**After (new fields added):**
```json
{
  "movement": { "walk": 30 },
  "vision": { "darkvision": 60 },
  "structuredActions": [{
    "id": "yatagan",
    "name": "Ятаган",
    "description": "<p>Рукопашная атака оружием...</p>",
    "category": "action",
    "attack": {
      "type": "melee_weapon",
      "bonus": 4,
      "reach": 5,
      "targets": 1,
      "damage": [{ "diceCount": 1, "diceType": "d6", "bonus": 2, "damageType": "slashing" }]
    }
  }]
}
```

### E.2 Flying Creature with Hover (Death Slaad)

**Before:**
```json
{
  "speed": [
    { "value": 60 },
    { "name": "летая", "value": 60, "additional": "парит" }
  ]
}
```

**After:**
```json
{
  "movement": { "walk": 60, "fly": 60, "hover": true }
}
```

### E.3 Breath Weapon (Mud Mephit)

**Before:**
```json
{
  "llm_parsed_attack": [{
    "name": "Грязевое дыхание (перезарядка 6)",
    "type": "area",
    "save_dc": 11,
    "save_type": "Ловкости",
    "additional_effects": [{ "condition": "опутанной", "duration": "1 минуту" }]
  }]
}
```

**After:**
```json
{
  "structuredActions": [{
    "id": "gryazevoe-dyhanie",
    "name": "Грязевое дыхание",
    "category": "action",
    "recharge": { "minRoll": 6 },
    "savingThrow": {
      "ability": "DEX",
      "dc": 11,
      "onFail": "effect applies",
      "onSuccess": "no effect"
    },
    "effects": [{
      "condition": {
        "condition": "restrained",
        "duration": "1 minute"
      }
    }]
  }]
}
```

---

## F. Decisions (Resolved)

| # | Question | Decision |
|---|----------|----------|
| 1 | Combined damage types like `"кислота, холод, огонь"` — how to represent? | `damageType: "varies"` + full list remains in `description` |
| 2 | Creatures with complex speed strings — populate partial Movement or skip entirely? | **SKIP entirely** — keep only legacy Speed, track in tech debt |
| 3 | `additional_effects` that aren't conditions (e.g., "урон от раны увеличивается") — how to handle? | Add `description` field to `ActionEffect` for non-standard effects |
| 4 | Generate `id` via transliteration or sequential? | **Transliteration** (e.g., `"gryazevoe-dyhanie"`) |
| 5 | Store original `actions[].value` in `structuredActions[].description`? | **YES** — store full HTML for fallback display |

---

## G. Technical Debt / Known Limitations

> **Purpose:** Track items intentionally skipped in v1 migration for future improvement.

### G.1 Skipped: Complex Speed Strings (~15 creatures)

**What:** Creatures with conditional/form-dependent speeds like werebeasts, vampires, druids.

**Examples:**
| Creature | Speed.name value |
|----------|------------------|
| Вервольф | `" (40 фт. в форме волка или гибрида)"` |
| Вампир | `"(лазая 30 фт., летая 60 фт. только в форме летучей мыши или гибрида)"` |
| Друид с формами | `", 40 фт. (только в облике волка), копая 5 фт. (только в облике лисы)..."` |

**Current behavior:** `movement` field is NOT populated. Frontend should fallback to legacy `speed` array.

**Future improvement:**
- Add `MovementByForm` struct: `map[string]CreatureMovement` keyed by form name
- Or add `conditionalSpeeds` array with conditions
- Requires manual review of each creature

**Query to find these:**
```javascript
db.creatures.find({
  "speed.name": { $regex: /фт\.|форм|облик/i }
}, { "name.rus": 1, "speed": 1 })
```

**Priority:** Low — affects shapeshifters only, legacy display works fine.

---

### G.2 Skipped: Creatures Without llm_parsed_attack (5 creatures)

**What:** Creatures that have `actions` array but no `llm_parsed_attack`.

**List:**
1. Подружка невесты Заггтмой
2. Вирмлинг медного дракона
3. Б'рог
4. Драконид манипулятор
5. Истарианский дрон

**Current behavior:** `structuredActions` is NOT populated. Frontend should fallback to legacy `actions` array.

**Future improvement:**
- Manually run LLM parsing for these 5 creatures
- Or manually create `structuredActions` entries

**Query to find these:**
```javascript
db.creatures.find({
  "llm_parsed_attack": { $exists: false },
  "actions.0": { $exists: true }
}, { "name.rus": 1 })
```

**Priority:** Low — only 5 creatures.

---

### G.3 Partial: Non-Standard Conditions/Effects

**What:** `additional_effects` that don't map to standard D&D conditions.

**Examples:**
- `"скорость уменьшается на 10 футов"` — speed reduction
- `"максимум хитов уменьшается"` — max HP reduction
- `"заражена синегнилью"` — disease
- `"проклята ликантропией"` — curse
- `"загорается"` — ongoing fire damage
- `"прикрепляется к цели"` — attached parasite

**Current behavior:** Stored as text in `effects[].description`. No automation.

**Future improvement:**
- Add specialized effect types: `SpeedModifier`, `MaxHPModifier`, `Disease`, `Curse`
- Parse numeric values from text (e.g., "10 футов" → `speedReduction: 10`)
- Requires extending `ActionEffect` struct

**Priority:** Medium — affects combat automation quality.

---

### G.4 Partial: Combined Damage Types

**What:** Attacks where caster chooses damage type from a list.

**Examples:**
- Хроматическая сфера: `"кислота, холод, огонь, молния или яд"`
- Призматический луч: `"кислота, холод, огонь, силовое поле, электричество, излучение или звук"`

**Current behavior:** `damageType: "varies"`. Full list preserved in `description`.

**Future improvement:**
- Add `damageTypeChoices: string[]` field to `DamageRoll`
- UI shows dropdown for type selection when rolling damage

**Priority:** Low — edge case, text description is sufficient.

---

### G.5 Not Migrated: BonusActions and Reactions

**What:** Only `actions` → `structuredActions` is migrated. `bonusActions` and `reactions` remain text-only.

**Current behavior:** Legacy `bonusActions` and `reactions` arrays unchanged.

**Future improvement:**
- Run LLM parsing on `bonusActions` array → add to `structuredActions` with `category: "bonus_action"`
- Run LLM parsing on `reactions` array → add to `structuredActions` with `category: "reaction"`
- Legendary actions in `legendary.list` also need parsing

**Priority:** Medium — needed for full combat automation.

---

### G.6 Summary Table

| Item | Count | Impact | Priority | Effort |
|------|-------|--------|----------|--------|
| Complex speed strings | ~15 | Low (display works) | Low | Medium |
| Missing llm_parsed_attack | 5 | Low | Low | Low |
| Non-standard effects | ~100+ | Medium (partial automation) | Medium | High |
| Combined damage types | ~20 | Low (text fallback) | Low | Low |
| BonusActions/Reactions | ~500+ | Medium (incomplete automation) | Medium | Medium |

**Total technical debt:** Manageable. Core migration covers 95%+ of data.

---

## Changelog

| Date | Change |
|------|--------|
| 2026-02-06 | Initial draft with all mapping rules |
| 2026-02-06 | Resolved all open questions, added Technical Debt section (G.1-G.6) |
