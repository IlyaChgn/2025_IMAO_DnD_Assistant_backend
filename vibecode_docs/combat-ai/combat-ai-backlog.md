# Combat AI — Task Backlog

> **Type:** Implementation Backlog
> **Status:** Draft
> **Feature Plan:** [combat-ai-design-plan.md](combat-ai-design-plan.md)
> **Branch:** `feature/combat-ai-mvp`
> **Last updated:** 2026-02-21

---

## Overview

Задачи для реализации Combat AI, Phase 1 (MVP). Порядок задач отражает зависимости — каждая следующая зависит от предыдущей. Acceptance criteria взяты из Feature Plan (секция I).

**Dependency graph:**

```
PR-0 ──┐
       ├── PR-6
PR-1 → PR-2 → PR-3 → PR-4 → PR-5 → PR-6 → PR-7
```

---

## Phase 1: MVP

### PR-0: Action Pipeline Extension for NPC Actions

**Status:** Done ✅ (PR #30, merged 2026-02-22)
**Dependencies:** None (can be done in parallel with PR-1 through PR-5)
**Branch:** `feature/action-pipeline-npc-actions`

**Problem:** Existing `ExecuteAction()` in `internal/pkg/actions/usecases/actions.go` is **entirely PC-centric**:

1. **Dispatcher (actions.go:70-94):** Always loads `CharacterBase` via `characterRepo` и `compute.ComputeDerived()`. Для NPC `CharacterBase` не существует — у них `Creature` template.
2. **resolveWeaponAttack (resolve_weapon.go):** Ищет оружие в `CharacterBase.Weapons[]` через `findWeapon(charBase, cmd.WeaponID)`. У NPC оружие — это `StructuredAction` с `AttackRollData`.
3. **resolveSpellCast (resolve_spell.go):** Ищет заклинания в `CharacterSpellcasting.Spells` (PC-модель). У NPC заклинания — в `Creature.Spellcasting.SpellsByLevel` (другая модель).
4. **resolveUseFeature (resolve_feature.go):** Ищет features в `CharacterBase.Features[]`. У NPC abilities с SavingThrow (дыхание дракона и т.д.) — это `StructuredAction` с `SavingThrowData`.

**Scope:**

Добавить NPC-ветку в `ExecuteAction()` dispatcher. Определять NPC/PC по `participant.IsPlayerCharacter`:

- [x] **NPC branch в dispatcher (actions.go):**
  - Если `participant.IsPlayerCharacter == false`:
    - Загрузить `Creature` template по `participant.CreatureID` из `bestiaryRepo` (уже инжектирован в `actionsUsecases`)
    - Передать `creature` вместо `charBase`/`derived` в NPC-resolvers
  - Если `participant.IsPlayerCharacter == true`:
    - Существующая PC-логика без изменений

- [x] **resolveNPCWeaponAttack (новый файл: resolve_npc_weapon.go):**
  - Найти `StructuredAction` по `cmd.WeaponID` == `StructuredAction.ID` в `creature.StructuredActions`
  - Бросок атаки: d20 + `AttackRollData.Bonus`
  - Бросок урона: для каждого `AttackRollData.Damage[]` → `DiceCount` × `DiceType` + `Bonus`
  - Применить урон к цели (переиспользовать `applyDamageToTarget()` — он уже различает NPC/PC target)
  - Формат `ActionResponse` идентичен PC-атакам

- [x] **resolveNPCUseFeature (новый файл: resolve_npc_feature.go):**
  - Найти `StructuredAction` по `cmd.FeatureID` == `StructuredAction.ID`
  - Если `StructuredAction.SavingThrow != nil` (дыхание дракона):
    - Для каждой цели: бросок спасброска d20 + save bonus vs DC
    - Урон при провале: `SavingThrowData.Damage[]`
    - `OnSuccess == "half damage"` → половина урона при успехе
  - Если `StructuredAction.Uses != nil` → проверить/уменьшить `RuntimeState.Resources.AbilityUses[actionID]`
  - Если `StructuredAction.Recharge != nil` → сбросить `RuntimeState.Resources.RechargeReady[actionID] = false`
  - Применить `ActionEffect[]` (conditions, ongoing damage, movement)

- [x] **resolveNPCSpellCast (новый файл: resolve_npc_spell.go):**
  - Phase 1 (минимальная реализация):
    - Найти spell по `cmd.SpellID` в `creature.Spellcasting.SpellsByLevel` или `creature.Spellcasting.Spells`
    - Использовать `Spellcasting.SpellAttackBonus` для spell attacks
    - Использовать `Spellcasting.SpellSaveDC` для spell saves
    - Уменьшить `RuntimeState.Resources.SpellSlots[slotLevel]`
    - Если `SpellDefinition` доступна (через `spellsRepo`) — использовать полную resolution
    - Иначе — записать в log "spell cast" без автоматического урона (DM разрешит вручную)
  - Phase 2: полная интеграция с SpellDefinition

- [x] **Не ломать PC pipeline:** все существующие resolve-функции остаются без изменений
- [x] **Audit log:** NPC-действия записываются аналогично PC-действиям
- [ ] **Unit tests:** NPC weapon attack, NPC use_feature (breath weapon), PC weapon attack (regression)

**Key files:**
- `internal/pkg/actions/usecases/actions.go` — dispatcher: добавить NPC-ветку (строки 70-128)
- `internal/pkg/actions/usecases/resolve_weapon.go` — существующий PC resolver (НЕ трогать)
- `internal/pkg/actions/usecases/resolve_npc_weapon.go` — **новый файл**
- `internal/pkg/actions/usecases/resolve_npc_feature.go` — **новый файл**
- `internal/pkg/actions/usecases/resolve_npc_spell.go` — **новый файл**
- `internal/pkg/actions/usecases/resolve_target.go` — уже различает NPC/PC target, переиспользовать
- `internal/models/action_structured.go` — `StructuredAction`, `AttackRollData`, `SavingThrowData`
- `internal/models/creature.go` — `Creature.StructuredActions`, `Creature.Spellcasting`

**Acceptance criteria:**
- [x] `ExecuteAction()` с `weapon_attack` type для NPC → находит `StructuredAction` по ID, бросает d20 + bonus, считает урон
- [x] `ExecuteAction()` с `use_feature` type для NPC → обрабатывает SavingThrow-based abilities (дыхание дракона)
- [x] `ExecuteAction()` с `spell_cast` type для NPC → использует `Spellcasting.SpellAttackBonus/SpellSaveDC`, тратит слот
- [x] PC weapon/spell/feature attacks continue to work (no regression)
- [x] `applyDamageToTarget()` корректно мутирует HP для NPC и PC targets (уже работает)
- [x] Recharge-способность после использования: `RechargeReady[actionID] = false`
- [x] Limited-use способность: `AbilityUses[actionID]` уменьшается
- [x] Audit log записывает NPC-действия

---

### PR-1: MultiattackGroup Model + Creature Migration

**Status:** Done ✅ (PR #31, merged 2026-02-22)
**Dependencies:** None
**Branch:** `feature/combat-ai-multiattack-model`

**Контекст:** У существ есть два формата action data:
- **Legacy:** `AttackLLM` (в `internal/models/attack.go`) — содержит `Attacks []MultiAttackLLM` с multiattack info в формате `{Type: "bite", Count: 1}`. Генерируется ActionProcessorService (gRPC LLM, вызывается при генерации существа).
- **New:** `StructuredAction` (в `internal/models/action_structured.go`) — машиночитаемый формат. **Не содержит multiattack info** — это отдельная сущность.
- **Migration tool:** `cmd/migrate_creatures/main.go` — конвертирует legacy data → StructuredActions. Нужно расширить для генерации MultiattackGroup.

**Scope:**
- [ ] Создать `internal/models/multiattack.go` с struct'ами (определения — в Feature Plan E.7):
  ```go
  type MultiattackGroup struct {
      ID      string             `json:"id" bson:"id"`
      Name    string             `json:"name" bson:"name"`
      Actions []MultiattackEntry `json:"actions" bson:"actions"`
  }
  type MultiattackEntry struct {
      ActionID string `json:"actionId" bson:"actionId"` // → StructuredAction.ID
      Count    int    `json:"count" bson:"count"`
  }
  ```
- [ ] Добавить `Multiattacks []MultiattackGroup` field в `Creature` struct (`internal/models/creature.go`)
- [ ] Расширить `cmd/migrate_creatures/main.go`:
  - Для существ с `AttackLLM` содержащим `Attacks []MultiAttackLLM`:
    - Создать `MultiattackGroup` где каждый `MultiAttackLLM.Type` → найти соответствующий `StructuredAction.ID` по имени
    - `MultiAttackLLM.Count` → `MultiattackEntry.Count`
  - Для существ без legacy multiattack data → пропустить (не все существа имеют multiattack)
- [ ] **(Опционально, можно отложить)** Обновить ActionProcessorService gRPC proto (`internal/pkg/bestiary/delivery/protobuf/actions_processor_llm.proto`) и LLM-сервис для генерации MultiattackGroup при создании новых существ
- [ ] Unit tests: JSON/BSON round-trip для MultiattackGroup, migration logic для типичного дракона

**Key files:**
- `internal/models/multiattack.go` — **новый файл**
- `internal/models/creature.go` — добавить `Multiattacks` field
- `cmd/migrate_creatures/main.go` — расширить migration logic
- `internal/models/attack.go` — `AttackLLM.Attacks []MultiAttackLLM` (legacy source data, НЕ трогать)
- `internal/pkg/bestiary/delivery/protobuf/actions_processor_llm.proto` — (опционально) proto update

**Acceptance criteria:**
- [ ] `Creature` model имеет `Multiattacks []MultiattackGroup` field с `omitempty`
- [ ] `MultiattackEntry.ActionID` корректно ссылается на `StructuredAction.ID` существа
- [ ] Migration tool конвертирует существо с legacy multiattack data (напр. Dragon: `Attacks: [{Type: "bite", Count: 1}, {Type: "claw", Count: 2}]`) в `MultiattackGroup`
- [ ] Существа без multiattack → `Multiattacks` остаётся nil/empty
- [ ] Existing CRUD endpoints (bestiary) прозрачно сохраняют/загружают новое поле (MongoDB schemaless)
- [ ] Dry-run migration проходит без ошибок на существующих данных

**Design reference:** Feature Plan, section E.7

---

### PR-2: Core Interfaces, Models, Intelligence System

**Status:** Done ✅ (PR #32, merged 2026-02-22)
**Dependencies:** PR-1
**Branch:** `feature/combat-ai-core`

**Scope:**
- [ ] Create `internal/pkg/combatai/` package
- [ ] `interfaces.go` — `CombatAI` interface:
  ```go
  type CombatAI interface {
      DecideTurn(input *TurnInput) (*TurnDecision, error)
  }
  ```
- [ ] `models.go` — все struct'ы из Feature Plan E.2 (скопировать дословно):
  - `TurnInput` — полное определение с комментариями (ActiveNPC, CreatureTemplate, Participants, CombatantStats map, Intelligence, PreviousTargetID, MapWidth/Height, WalkabilityGrid)
  - `CombatantStats` — AC, SaveBonuses, Resistances, Immunities, Vulnerabilities, IsPC
  - `TurnDecision` — Movement, Action, BonusAction, Reaction, Reasoning
  - `ActionDecision` — ActionType, ActionID, TargetIDs, SlotLevel, MultiattackGroupID, MultiattackSteps, ActionName, ExpectedDamage, Reasoning
  - `MultiattackStep` — ActionType, ActionID, TargetIDs
  - `MovementDecision`, `ReactionRule`, `CreatureRole` (constants), `UniversalAction`
- [ ] **HP helper** (`models.go` или `helpers.go`):
  - **Важно:** NPC и PC хранят HP в разных местах:
    - NPC: `participant.RuntimeState.CurrentHP` / `RuntimeState.MaxHP`
    - PC: `participant.CharacterRuntime.CurrentHP` / (MaxHP вычисляется из CharacterBase, но для AI достаточно CurrentHP)
  - Helper: `GetCurrentHP(p models.ParticipantFull) int` — возвращает HP из правильного поля
  - Helper: `IsAlive(p models.ParticipantFull) bool` — `GetCurrentHP(p) > 0`
- [ ] `intelligence.go` — `ComputeIntelligence(intScore int, difficultyMod float64) float64`:
  - Формула: `clamp((float64(intScore) - 1.0) / 19.0 + difficultyMod, 0.05, 1.0)`
  - `intScore` берётся из `models.Creature.Ability.Int`
- [ ] `distance.go` — `DistanceFt(a, b *models.CellsCoordinates) int`:
  - Chebyshev: `max(|dx|, |dy|) * 5`
  - Если `a == nil` или `b == nil` → `math.MaxInt32`
- [ ] Unit tests для ComputeIntelligence, DistanceFt, GetCurrentHP

**Key files:**
- `internal/pkg/combatai/interfaces.go`
- `internal/pkg/combatai/models.go`
- `internal/pkg/combatai/intelligence.go`
- `internal/pkg/combatai/distance.go`

**Acceptance criteria:**
- [ ] `ComputeIntelligence(3, 0.0)` returns 0.11 (±0.01)
- [ ] `ComputeIntelligence(20, 0.0)` returns 1.0
- [ ] `ComputeIntelligence(10, -0.3)` returns 0.17 (±0.01)
- [ ] `ComputeIntelligence(1, 0.0)` returns 0.05 (clamped minimum)
- [ ] `DistanceFt` returns `max(|dx|, |dy|) * 5`, `MaxInt32` if either coord is nil
- [ ] `GetCurrentHP()` корректно возвращает HP для NPC (RuntimeState) и PC (CharacterRuntime)
- [ ] All types compile and are importable from other packages

**Design reference:** Feature Plan, sections E.1 (interface), E.2 (struct definitions — **копировать дословно**), E.8 (intelligence formula)

---

### PR-3: Role Classifier + Expected Value Calculator

**Status:** Done ✅ (PR #33, merged 2026-02-22)
**Dependencies:** PR-2
**Branch:** `feature/combat-ai-classifier-ev`

**Scope:**

- [ ] `role_classifier.go` — `ClassifyRole(creature models.Creature) CreatureRole`
  - Алгоритм классификации (7 шагов, порядок приоритетов — в Feature Plan E.3):
    1. Есть `Spellcasting` с attack/damage → Caster
    2. Все `StructuredActions` ranged → Ranged
    3. `Ability.Str >= 16` И `HP.Average >= parseCR(CR) * 15` → Brute
    4. `Ability.Dex >= Ability.Str + 4` И `Movement.Walk >= 40` → Skirmisher
    5. `ArmorClass >= 18` ИЛИ (HP высокий и есть melee) → Tank
    6. >= 2 StructuredActions с SavingThrow или ConditionEffect → Controller
    7. Fallback: есть ranged → Ranged, иначе → Brute
  - Данные берутся из `models.Creature`: `Ability` (struct с Int, Str, Dex, ...), `ArmorClass` (int), `HP.Average` (string → parse), `StructuredActions[]`, `Spellcasting`

- [ ] **String→number converters** (в `role_classifier.go`):
  - `parseCR(cr string) float64` — `"1/4"→0.25`, `"1/2"→0.5`, `"3"→3.0`. Поле: `Creature.ChallengeRating` (string)
  - `parseProfBonus(pb string) int` — `"+2"→2`, `"+6"→6`. Поле: `Creature.ProficiencyBonus` (string)

- [ ] `expected_value.go`:
  - `ComputeExpectedDamage(action models.StructuredAction, targetStats CombatantStats) float64`
    - Для `action.Attack != nil` (AttackRoll):
      ```
      hit_chance = clamp((21 - (targetAC - attack.Bonus)) / 20, 0.05, 0.95)
      avg_damage = sum(diceCount * (parseDiceMax(diceType)+1)/2 + bonus) for each Damage[]
      expected = hit_chance * avg_damage + 0.05 * avg_damage  // crit
      ```
    - Для `action.SavingThrow != nil` (SaveDC):
      ```
      fail_chance = clamp((DC - targetSaveBonus - 1) / 20, 0.05, 0.95)
      half_on_success = (OnSuccess == "half damage")
      expected = fail_chance * avg + (1-fail_chance) * (half ? avg/2 : 0)
      ```
    - Полные формулы — в Feature Plan E.5
  - `parseDiceMax(dt string) int` — `"d6"→6`, `"d8"→8`, `"d10"→10`. Поле: `DamageRoll.DiceType` (string)

- [ ] **Spell evaluation heuristic** (`expected_value.go`):
  - `EstimateSpellDamage(spell models.SpellKnown, spellcasting models.Spellcasting, targetStats CombatantStats) float64`
    - `SpellKnown` содержит: `Name`, `Level`, `QuickRef *SpellQuickRef` (с `Range`, `Concentration`)
    - Если `QuickRef == nil` → вернуть 0 (не можем оценить, пропускаем)
    - Если `QuickRef.Range == "Self"` → вернуть 0 (не damage spell)
    - Для spell attack: `hit_chance * (spell_level * 3.5 + 3)`, hit_chance из `spellcasting.SpellAttackBonus`
    - Для save-based: `fail_chance * (spell_level * 4.5)`, fail_chance из `spellcasting.SpellSaveDC`
    - Для cantrips (level 0): base = 5.5, масштабирование по `spellcasting.CasterLevel` (>=5→×2, >=11→×3, >=17→×4)
    - Полные формулы — в Feature Plan E.5 "Оценка заклинаний"

- [ ] Unit tests для каждого classification rule + expected value расчётов

**Key files:**
- `internal/pkg/combatai/role_classifier.go`
- `internal/pkg/combatai/expected_value.go`
- `internal/models/creature.go` — `Creature` struct (данные: `Ability`, `ArmorClass`, `HP`, `ChallengeRating`, `Spellcasting`)
- `internal/models/action_structured.go` — `StructuredAction`, `AttackRollData`, `DamageRoll`
- `internal/models/spellcasting.go` — `Spellcasting`, `SpellKnown`, `SpellQuickRef`

**Acceptance criteria:**
- [ ] Ogre (STR 19, HP 59, melee only) → Brute
- [ ] Skeleton Archer (ranged only, no melee StructuredActions) → Ranged
- [ ] Lich (has Spellcasting with SpellsByLevel) → Caster
- [ ] `parseCR("1/4")` → 0.25, `parseCR("1/2")` → 0.5, `parseCR("3")` → 3.0
- [ ] `parseDiceMax("d6")` → 6, `parseDiceMax("d10")` → 10
- [ ] Attack vs AC 15, bonus +5: hit_chance ≈ 0.55, expected includes crit bonus
- [ ] SavingThrow DC 15 vs +2 save: fail_chance ≈ 0.60
- [ ] Cantrip at CasterLevel 11: damage ≈ 16.5 (5.5 × 3)
- [ ] Spell with QuickRef == nil: returns 0 (skipped)

**Design reference:** Feature Plan, sections E.3 (classification), E.5 (expected value + spell heuristic)

---

### PR-4: Target Selector + Action Selector + Multiattack Evaluator

**Status:** Done ✅ (PR #34, merged 2026-02-22)
**Dependencies:** PR-3
**Branch:** `feature/combat-ai-selectors`

**Контекст — откуда брать данные в `TurnInput`:**

Все функции в этом PR работают с `TurnInput` (из PR-2). Ключевые поля:
- **Враги NPC:** `input.Participants` где `IsPlayerCharacter == true` (PC — враги для NPC)
- **HP участника:** `participant.RuntimeState.CurrentHP` / `RuntimeState.MaxHP` (NPC), `participant.CharacterRuntime.CurrentHP` (PC). **Важно: оба случая.** Использовать хелпер `GetCurrentHP(p)` / `IsAlive(p)` из PR-2.
- **AC/saves/resistances цели:** `input.CombatantStats[participant.InstanceID]` — универсальная карта (и NPC, и PC)
- **Доступные действия NPC:** `input.CreatureTemplate.StructuredActions` — фильтровать по `Category == "action"` для основного действия, `"bonus_action"` для бонусного
- **Заряженные recharge-способности:** `input.ActiveNPC.RuntimeState.Resources.RechargeReady[actionID] == true`
- **Доступные spell slots:** `input.ActiveNPC.RuntimeState.Resources.SpellSlots[level] > 0`
- **Заклинания NPC:** `input.CreatureTemplate.Spellcasting.SpellsByLevel` (map[int][]SpellKnown) + `Spellcasting.Spells` (flat list)
- **Расстояние до цели:** `DistanceFt(input.ActiveNPC.CellsCoords, target.CellsCoords)` (из PR-2)
- **Досягаемость атаки:** `action.Attack.Reach` (melee, в футах) или `action.Attack.Range.Normal` (ranged, в футах)

**Scope:**

- [ ] `target_selector.go` — `SelectTarget(input *TurnInput, action *models.StructuredAction) []string`
  - Intelligence тир определяется из `input.Intelligence` (уже предвычислен)
  - Алгоритм по тирам (полная логика — Feature Plan E.4):
    - `< 0.15`: sticky → `input.PreviousTargetID` (если жив), иначе ближайший враг
    - `0.15-0.35`: ближайший, но если есть раненый (<25% HP) в reach → добить
    - `0.35-0.55`: melee NPC → минимум HP в reach; ranged NPC → минимум AC в range
    - `>= 0.55`: как 0.35-0.55, но бонус за концентрацию (`target.RuntimeState.Concentration != nil`) и low HP
  - Враг = `participant.IsPlayerCharacter == true` (для NPC) и `CurrentHP > 0` (живой)

- [ ] `action_selector.go` — `SelectAction(input *TurnInput, role CreatureRole) *ActionDecision`
  - Собрать кандидатов:
    1. Все `StructuredActions` с `Category == "action"` (или "legendary" если есть)
    2. Отфильтровать: uses exhausted? recharge not ready? spell slot unavailable?
    3. Для каждого кандидата: вызвать `ComputeExpectedDamage()` + `SelectTarget()` → получить EV
    4. Для spell кандидатов: вызвать `EstimateSpellDamage()` → получить EV
  - Приоритизация (Feature Plan E.5):
    1. Recharge ready → использовать (это самые мощные способности)
    2. Multiattack (если есть, см. ниже)
    3. Max expected value
    4. Dodge (если HP < 25% и нет действий с EV > 0)
  - Intelligence gate: `rand() < intelligence` → оптимальное, иначе случайное из допустимых
  - Вернуть `*ActionDecision` с заполненными полями (ActionType, ActionID, TargetIDs, ExpectedDamage, Reasoning)

- [ ] `multiattack.go` — `EvaluateMultiattack(input *TurnInput, groups []models.MultiattackGroup) *ActionDecision`
  - Для каждой группы:
    - Для каждого `MultiattackEntry`: найти `StructuredAction` по `ActionID`, вычислить EV компонента
    - Сумма EV всех компонентов = EV группы
  - Выбрать группу с максимальным суммарным EV
  - Результат: `ActionDecision` с заполненным `MultiattackSteps[]` (каждый шаг с ActionID и TargetIDs)
  - Каждый шаг multiattack может иметь свою цель (bite → tank, claws → caster)

- [ ] **Test fixtures** (в `internal/pkg/combatai/testdata/` или `fixtures_test.go`):
  - `zombie_fixture.go` — `Creature{Ability.Int: 3, StructuredActions: [{melee slam}]}`, `ParticipantFull` с RuntimeState
  - `goblin_fixture.go` — `Creature{Ability.Int: 10, StructuredActions: [{scimitar, melee}, {shortbow, ranged}]}`
  - `dragon_fixture.go` — `Creature{Ability.Int: 16, StructuredActions: [{bite}, {claw}, {fire_breath with Recharge}], Multiattacks: [{ID: "multi1", Actions: [{bite,1},{claw,2}]}]}`
  - Каждый fixture включает: `Creature` template, `ParticipantFull` с RuntimeState, `CombatantStats` для целей

- [ ] Unit tests для каждого intelligence tier и multiattack selection

**Key files:**
- `internal/pkg/combatai/target_selector.go`
- `internal/pkg/combatai/action_selector.go`
- `internal/pkg/combatai/multiattack.go`

**Acceptance criteria:**
- [ ] Zombie (INT 3) атакует ближайшего врага, не переключает цели (sticky targeting)
- [ ] Goblin (INT 10) атакует ближайшего, но добивает раненого врага в reach
- [ ] Lich (INT 20) выбирает оптимальное действие и лучшую цель
- [ ] Dragon: `RechargeReady["fire_breath"] == true` → выбирает breath weapon
- [ ] Dragon: breath не заряжен → multiattack (bite + 2 claws)
- [ ] Skeleton archer: цель с наименьшим AC среди врагов в range
- [ ] Несколько MultiattackGroup: AI выбирает группу с максимальным суммарным EV
- [ ] Intelligence gate: при intelligence = 0.1, ~90% ходов неоптимальные; при 1.0 — всегда оптимальные

**Design reference:** Feature Plan, sections E.4 (target), E.5 (action), E.7 (multiattack)

---

### PR-5: AI Engine + Universal Actions (Dodge)

**Status:** Done ✅ (PR #36)
**Dependencies:** PR-4
**Branch:** `feature/combat-ai-engine`

**Scope:**

- [ ] `ai_engine.go` — `RuleBasedAI` struct implementing `CombatAI` interface:
  ```go
  type RuleBasedAI struct{}
  func NewRuleBasedAI() CombatAI { return &RuleBasedAI{} }
  func (ai *RuleBasedAI) DecideTurn(input *TurnInput) (*TurnDecision, error)
  ```
  - Оркестрация `DecideTurn()`:
    1. Проверить состояние: `CurrentHP <= 0` → return nil (мёртв). `hasCondition(ConditionIncapacitated)` → return nil (incapacitated)
    2. `role := ClassifyRole(input.CreatureTemplate)` (из PR-3)
    3. `action := SelectAction(input, role)` (из PR-4, внутри вызывает SelectTarget + ComputeExpectedDamage)
    4. Если `action == nil` и HP < 25% → `action = dodgeDecision()` (см. ниже)
    5. Собрать `TurnDecision{Action: action, Movement: nil, BonusAction: nil, Reasoning: ...}`
    6. Генерировать `Reasoning` string: `"Brute role: attack nearest threat with Longsword (EV=8.5)"`
  - **Важно:** `DecideTurn()` — чистая функция. Не мутирует state, не обращается к DB. Только принимает `TurnInput`, возвращает `TurnDecision`.

- [ ] `universal_actions.go` — **только решение**, не исполнение:
  - `dodgeDecision() *ActionDecision` — возвращает `ActionDecision{ActionID: "dodge", ActionName: "Dodge", ActionType: "", Reasoning: "Low HP, no good attacks"}`
  - `ActionType` пуст — Dodge не маппится на существующие `models.ActionType`, потому что не проходит через action pipeline
  - Условие использования: HP < 25% И `SelectAction()` не нашёл действия с EV > 0
  - **Исполнение Dodge** (добавление StatModifier, broadcast) — ответственность PR-6 (usecases layer), не AI engine

- [ ] Integration tests: полный TurnInput → TurnDecision flow для всех fixture-существ из PR-4
- [ ] Test fixtures: `lich_fixture.go` (INT 20, Spellcasting), `skeleton_archer_fixture.go` (INT 6, ranged only)
- [ ] Property test: `DecideTurn()` никогда не паникует на любом валидном `TurnInput`

**Key files:**
- `internal/pkg/combatai/ai_engine.go`
- `internal/pkg/combatai/ai_engine_test.go`
- `internal/pkg/combatai/universal_actions.go`

**Acceptance criteria:**
- [ ] `DecideTurn()` returns valid `TurnDecision` для zombie, goblin, dragon, lich, skeleton_archer
- [ ] NPC при HP < 25% и нет хороших атак: `Action.ActionID == "dodge"`
- [ ] NPC при HP > 50% и есть атаки: Action — НЕ dodge
- [ ] Incapacitated NPC: returns `&TurnDecision{Action: nil, Reasoning: "Incapacitated — skip"}`
- [ ] Dead NPC: returns `nil, nil` (или error — решить при реализации)
- [ ] Reasoning содержит: роль, выбранное действие, цель, expected damage
- [ ] `DecideTurn()` не мутирует `TurnInput` (чистая функция)
- [ ] Intelligence 0.1 с одинаковым seed → иногда неоптимальный выбор; Intelligence 1.0 → всегда оптимальный

**Design reference:** Feature Plan, sections E.5 (action selection), E.9 (Dodge — решение + исполнение раздельно)

---

### PR-6: HTTP Endpoint `ai-turn` + Usecases Glue

**Status:** Done ✅ (PR #37)
**Dependencies:** PR-5, PR-0
**Branch:** `feature/combat-ai-endpoint`

**Контекст — паттерны кодовой базы:**

- **DI pattern** (см. `internal/pkg/actions/usecases/actions.go`): приватный struct + `NewXxxUsecases(...)` конструктор → возвращает interface.
- **Router** (см. `internal/pkg/server/delivery/routers/encounter.go`): `ServeXxxRouter(router *mux.Router, handler, middleware)` → subrouter с `Use(loginRequired)`.
- **Handler pattern** (см. любой handler): `func (h *Handler) Method(w, r)` → `json.Decode(r.Body)` → `h.usecases.DoSomething()` → `responses.SendOkResponse(w, result)`.
- **Encounter data** (см. `internal/pkg/actions/usecases/encounter_data.go`): encounter хранится как JSON blob в PostgreSQL. `ParseEncounterData(data)` парсит blob → `EncounterData{Participants: []ParticipantFull}`.
- **Ownership check:** encounter принадлежит пользователю → проверять `encounter.UserID == user.ID` (см. encounter handlers).

**Scope:**

- [ ] **`internal/pkg/combatai/interfaces.go`** (module-level interface):
  ```go
  type CombatAIUsecases interface {
      ExecuteAITurn(ctx context.Context, encounterID string, npcInstanceID string, userID int) (*AITurnResult, error)
  }
  ```

- [ ] **`internal/pkg/combatai/usecases/combat_ai_usecases.go`**:
  ```go
  type combatAIUsecases struct {
      ai          CombatAI                              // rule-based engine (из PR-5)
      encounters  encounterinterfaces.EncounterRepository
      bestiary    bestiaryinterfaces.BestiaryRepository  // для Creature template
      characters  characterinterfaces.CharacterBaseRepository // для PC CharacterBase
      actions     actionsinterfaces.ActionsUsecases       // ExecuteAction()
      table       tableinterfaces.TableManager            // BroadcastToEncounter()
  }
  func NewCombatAIUsecases(...) CombatAIUsecases { ... }
  ```

  - **`buildTurnInput()`** — алгоритм (полное описание — Feature Plan section F):
    1. Загрузить encounter из PostgreSQL: `encounters.GetEncounterByID(ctx, encounterID)`
    2. Распарсить participants: `ParseEncounterData(encounter.Data)` → `[]ParticipantFull`
    3. Найти active NPC по `npcInstanceID` в participants
    4. Загрузить Creature template: `bestiary.GetCreatureByID(ctx, activeNPC.CreatureID)` из MongoDB
    5. Для каждого participant → построить `CombatantStats`:
       - **NPC** (`IsPlayerCharacter == false`): загрузить Creature по `CreatureID` → `CombatantStats{AC: creature.ArmorClass, SaveBonuses: из creature.SavingThrows + Ability modifiers, Resistances: creature.DamageResistances, ...}`
       - **PC** (`IsPlayerCharacter == true`): загрузить CharacterBase по `CharacterRuntime.CharacterID` → `derived := compute.ComputeDerived(charBase)` → `CombatantStats{AC: derived.ArmorClass, SaveBonuses: derived.SaveBonuses, Resistances: derived.Resistances, ...}`
    6. `intelligence := ComputeIntelligence(creatureTemplate.Ability.Int, session.aiDifficultyMod)` (сигнатура из PR-2: `ComputeIntelligence(intScore int, difficultyMod float64)`)
    7. `PreviousTargetID`: из in-memory state или пустая строка (первый раунд)
    8. Walkability grid: nil (Phase 1 — нет движения)

  - **`executeTurn()`** — flow:
    1. `input := buildTurnInput(...)`
    2. `decision, err := ai.DecideTurn(input)`
    3. Если `decision.Action.ActionID == "dodge"` → **Dodge special path:**
       - Добавить `StatModifier{Name: "Dodge", Modifiers: [{Target: "ac", Op: "advantage"}], Duration: "until_turn"}` в `participant.RuntimeState.StatModifiers`
       - Сохранить обновлённый encounter data в PostgreSQL
       - Broadcast через WebSocket
       - Записать в audit log
       - Вернуть результат БЕЗ вызова action pipeline
    4. Если `decision.Action.MultiattackSteps != nil` → `executeMultiattack(decision.Action)`:
       - Для каждого step: `cmd := toActionCommand(decision.Action, &step)` → `actions.ExecuteAction(ctx, encounterID, &ActionRequest{CharacterID: npcInstanceID, Action: cmd}, systemUserID)`
       - Собрать все `ActionResponse` в массив
    5. Иначе (одиночное действие): `cmd := toActionCommand(decision.Action, nil)` → `actions.ExecuteAction(...)`
    6. Broadcast обновлённое состояние через `table.BroadcastToEncounter()`

  - **`toActionCommand()`** — конвертация (полный код — Feature Plan E.2):
    ```
    weapon_attack → cmd.WeaponID = ActionID, cmd.TargetID = TargetIDs[0]
    spell_cast    → cmd.SpellID  = ActionID, cmd.SlotLevel, cmd.TargetIDs
    use_feature   → cmd.FeatureID = ActionID, cmd.TargetID = TargetIDs[0]
    ```

- [ ] **`internal/pkg/combatai/delivery/combat_ai_handlers.go`**:
  ```go
  type CombatAIHandler struct {
      usecases CombatAIUsecases
      ctxUserKey interface{}
  }
  func NewCombatAIHandler(uc CombatAIUsecases, ctxUserKey interface{}) *CombatAIHandler
  ```
  - `POST /api/encounter/{encounterID}/ai-turn`:
    - Decode request: `{"npcID": "goblin-1"}`
    - Валидация: npcID непуст, participant найден, `IsPlayerCharacter == false`, HP > 0, `len(creature.StructuredActions) > 0`
    - `result, err := h.usecases.ExecuteAITurn(ctx, encounterID, npcID, userID)`
    - Error mapping: 400 (bad input), 403 (not owner), 404 (encounter not found), 422 (no StructuredActions)
    - Response: JSON с decision + actionResults (формат — Feature Plan section G)

- [ ] **Router registration** — новый файл `internal/pkg/server/delivery/routers/combatai.go`:
  ```go
  func ServeCombatAIRouter(router *mux.Router, handler *CombatAIHandler, loginRequired mux.MiddlewareFunc) {
      subrouter := router.PathPrefix("/encounter").Subrouter()
      subrouter.Use(loginRequired)
      subrouter.HandleFunc("/{encounterID}/ai-turn", handler.AITurn).Methods("POST")
  }
  ```
  - Зарегистрировать в `internal/pkg/server/delivery/routers/router.go` (основной файл)

- [ ] **DI wiring** — в `internal/pkg/server/app.go`:
  - Создать `combatAIEngine := combatai.NewRuleBasedAI()`
  - Создать `combatAIUsecases := combataiuc.NewCombatAIUsecases(combatAIEngine, encounterRepo, bestiaryRepo, characterRepo, actionsUsecases, tableManager)`
  - Создать `combatAIHandler := combataideliv.NewCombatAIHandler(combatAIUsecases, cfg.CtxUserKey)`
  - Вызвать `ServeCombatAIRouter(...)` в router builder

- [ ] Integration test: HTTP POST → AI decision → action execution → HP mutation → response

**Key files:**
- `internal/pkg/combatai/usecases/combat_ai_usecases.go` — **новый файл**
- `internal/pkg/combatai/delivery/combat_ai_handlers.go` — **новый файл**
- `internal/pkg/server/delivery/routers/combatai.go` — **новый файл**
- `internal/pkg/server/delivery/routers/router.go` — добавить вызов `ServeCombatAIRouter`
- `internal/pkg/server/app.go` — DI wiring
- `internal/pkg/actions/usecases/encounter_data.go` — `ParseEncounterData()` (переиспользовать)
- `internal/models/creature_runtime.go` — `ParticipantFull`, `StatModifier`
- `internal/pkg/compute/` — `ComputeDerived()` для PC stats

**Acceptance criteria:**
- [ ] `POST /api/encounter/{id}/ai-turn` с валидным NPC → decision + action result
- [ ] HP цели корректно мутируется через action pipeline (PR-0)
- [ ] Audit log записывает AI action
- [ ] WebSocket broadcast отправляет обновлённое состояние
- [ ] Error 400: npcID не найден / это PC / NPC мёртв
- [ ] Error 403: пользователь не владелец encounter
- [ ] Error 404: encounter не найден
- [ ] Error 422: у существа нет StructuredActions
- [ ] `buildTurnInput()` строит CombatantStats для NPC (из Creature.ArmorClass/SavingThrows) и PC (из DerivedStats)
- [ ] Dodge: StatModifier добавлен, pipeline НЕ вызывается, broadcast отправлен
- [ ] Multiattack: все шаги исполнены последовательно, все ActionResponse собраны

**Design reference:** Feature Plan, sections E.2 (toActionCommand, DistanceFt), E.9 (Dodge execution), F (folder structure, dependencies, buildTurnInput), G (API endpoint)

---

### PR-7: Auto-Play Mode + Turn Manager + `ai-round`

**Status:** Done ✅ (PR #38, merged 2026-02-23)
**Dependencies:** PR-6
**Branch:** `feature/ai-round-endpoint`

**Контекст — архитектура table/session:**

- **Session** (`internal/pkg/table/repository/session.go`): in-memory struct с `encounterData []byte`, `broadcast chan`, `participants map[int]*participant` (WebSocket connections). Одна goroutine на session (`go newSession.run(ctx)`) для broadcast loop.
- **CreateTableRequest** (`internal/models/table.go`): сейчас только `EncounterID string`. Нужно расширить.
- **BroadcastToEncounter** (`tableManager`): отправляет patch message из HTTP → WebSocket. Ищет session по `encounterID` через `encounterIndex map`. Используется inventory handlers для push-обновлений.
- **WS message types** (`internal/models/responses.go`): `BattleInfo`, `ParticipantsInfo`, `EncounterPatch`, `FogHistoryPatch` и т.д. **Нет** `your_turn` и `combat_end` — добавить.
- **WSResponse format**: `type WSResponse struct { Type WSMsgType; Data interface{} }` → marshal в JSON → send.

**Scope:**

Этот PR содержит две относительно независимые подзадачи: (A) `ai-round` endpoint и (B) auto-play turn loop.

#### A. `POST /api/encounter/{encounterID}/ai-round` endpoint

- [ ] Добавить handler `AIRound` в `combat_ai_handlers.go`:
  - Загрузить encounter, распарсить participants
  - Отсортировать participants по initiative (descending), NPC first on ties
  - Для каждого NPC participant (по порядку initiative):
    - Если dead (HP <= 0) или incapacitated → skip (`skipped: true`)
    - `processStartOfTurn()` → recharge rolls, reaction reset
    - `buildTurnInput()` → `DecideTurn()` → execute (переиспользовать `executeTurn()` из PR-6)
    - Собрать результат в массив
  - Проверить combat end conditions после каждого действия
  - Вернуть JSON response (формат — Feature Plan section G):
    ```json
    {"round": 3, "turns": [...], "combatEnded": false, "combatResult": ""}
    ```
- [ ] Зарегистрировать route: `subrouter.HandleFunc("/{encounterID}/ai-round", handler.AIRound).Methods("POST")` в `combatai.go` router
- [ ] `processStartOfTurn(participant, creatureTemplate)`:
  - **Recharge roll:** для каждого `StructuredAction` с `Recharge != nil` где `RechargeReady[actionID] == false` → roll d6 → если `roll >= Recharge.MinRoll` → `RechargeReady[actionID] = true`
  - **Reaction reset:** `Resources.ReactionUsed = false`
  - **Condition duration:** для conditions с `EndsOnTurn == "start"` и `TurnEntityID == participant.InstanceID` → удалить condition. Для conditions с `Duration == "rounds"` → `RoundsLeft--`, если 0 → удалить
  - **Legendary actions restore:** `Resources.LegendaryActions = creature.LegendaryActions.Count` (если есть)

#### B. Auto-Play Turn Loop

- [ ] **Расширить `CreateTableRequest`** (`internal/models/table.go`):
  ```go
  type CreateTableRequest struct {
      EncounterID     string  `json:"encounterID"`
      AIAutoPlay      bool    `json:"aiAutoPlay,omitempty"`
      AIDifficultyMod float64 `json:"aiDifficultyMod,omitempty"` // -0.5..+0.5
  }
  ```

- [ ] **Расширить session struct** (`internal/pkg/table/repository/session.go`):
  - Добавить поля: `aiAutoPlay bool`, `aiDifficultyMod float64`
  - Передать через `CreateSession()` (расширить сигнатуру или добавить options struct)

- [ ] **Новые WS message types** (`internal/models/responses.go`):
  ```go
  YourTurn    WSMsgType = "your_turn"
  CombatEnd   WSMsgType = "combat_end"
  AITurnResult WSMsgType = "ai_turn_result"
  ```

- [ ] **Turn manager** (`internal/pkg/combatai/usecases/turn_manager.go`):
  - `initTurnOrder(participants []ParticipantFull) []ParticipantFull` — `sort.SliceStable` по initiative desc, NPC first on ties
  - `advanceTurn()` — цикл (pseudocode — Feature Plan E.11):
    ```
    loop:
      currentTurnIndex = (currentTurnIndex + 1) % len(participants)
      if currentTurnIndex == 0: currentRound++
      if checkCombatEnd(): return
      p = participants[currentTurnIndex]
      if p dead or incapacitated: continue
      processStartOfTurn(p)
      if p.IsPlayerCharacter:
          broadcastYourTurn(p)    // → send "your_turn" WS message
          return                  // WAIT for player action
      else:
          decision := ai.DecideTurn(buildTurnInput(...))
          executeTurn(decision)
          broadcastAIResult(decision)
          continue               // next participant
    ```
  - **Модель ожидания PC-хода:**
    - Auto-play loop запускается как горутина
    - При PC-ходе: loop блокируется на channel `pcActionDone chan struct{}`
    - Когда PC выполняет действие (через existing action endpoint или WS) → signal в channel → loop продолжается
    - **Альтернатива (проще для Phase 1):** auto-play loop NOT a goroutine. Вместо этого:
      - `advanceTurn()` вызывается после каждого действия (NPC — автоматически, PC — по событию)
      - Frontend или WS handler после PC-действия вызывает `ContinueAutoPlay()` → runs `advanceTurn()` до следующего PC
    - **Решение при реализации** — оба подхода валидны, выбрать при кодировании
  - `checkCombatEnd() (ended bool, result string)`:
    - TPK: все PC dead → `"defeat"`
    - Victory: все NPC dead → `"victory"`
    - Timeout: `currentRound > maxRounds` (default 100) → `"timeout"`
    - Deadlock: все живые NPC не могут атаковать → skip + warning (не hard end)

- [ ] **WebSocket messages:**
  ```json
  // your_turn
  {"type": "your_turn", "data": {"participantID": "fighter-1", "round": 3, "turnIndex": 5}}

  // combat_end
  {"type": "combat_end", "data": {"result": "victory", "round": 5, "summary": "All enemies defeated"}}

  // ai_turn_result (broadcast после каждого NPC хода в auto-play)
  {"type": "ai_turn_result", "data": {"npcID": "goblin-1", "decision": {...}, "actionResults": [...]}}
  ```

- [ ] **Broadcast:** использовать `table.BroadcastToEncounter(ctx, encounterID, systemUserID, marshaledMsg)` — systemUserID = 0 или special admin ID (broadcast не фильтрует по senderID для patch messages, но нужно проверить — если relayPatchMessage исключает sender, использовать senderID=0)

- [ ] Integration tests: full auto-play round, combat end conditions

**Key files:**
- `internal/pkg/combatai/usecases/turn_manager.go` — **новый файл**
- `internal/pkg/combatai/delivery/combat_ai_handlers.go` — добавить `AIRound` handler
- `internal/models/table.go` — расширить `CreateTableRequest`
- `internal/models/responses.go` — добавить WS message types
- `internal/pkg/table/repository/session.go` — расширить session struct
- `internal/pkg/table/usecases/table.go` — передать aiAutoPlay/aiDifficultyMod в CreateSession
- `internal/pkg/table/interfaces.go` — расширить CreateSession signature (если нужно)
- `internal/pkg/server/delivery/routers/combatai.go` — добавить ai-round route

**Acceptance criteria:**
- [ ] `ai-round` выполняет ходы всех NPC последовательно, возвращает массив решений
- [ ] Dead NPC пропущен: `skipped: true, skipReason: "Dead (HP <= 0)"`
- [ ] Incapacitated NPC пропущен
- [ ] `processStartOfTurn()` бросает d6 для recharge-способностей, сбрасывает ReactionUsed
- [ ] Initiative sorting: stable, descending, NPC first on ties
- [ ] Auto-play: NPC ходят автоматически, PC получает `your_turn` через WebSocket
- [ ] Combat end: `combat_end` message при TPK / Victory / Timeout
- [ ] `aiDifficultyMod: -0.3` передаётся в `ComputeIntelligence` для всех NPC
- [ ] `ai-round` проверяет combat end после каждого NPC-хода
- [ ] Auto-play loop корректно останавливается при PC-ходе и возобновляется после

**Design reference:** Feature Plan, sections E.11 (auto-play, turn management, processStartOfTurn, checkCombatEnd), G (ai-round endpoint, WS messages)

---

## Phase 2 Tasks (after Phase 1 stabilizes)

| Task | Description | Status |
|------|-------------|--------|
| PR-8 | Threat assessment (intelligence-gated, `ThreatScore` formula) | Done ✅ (PR #39, 2026-02-23) |
| PR-9 | Spell slot management / resource manager (round-based economy) | Done ✅ (PR #40, 2026-02-23) |
| PR-10 | AoE target counting (geometry on grid, `AreaOfEffect`) | Done ✅ (PR #41, 2026-02-23) |
| PR-11 | Bonus actions + legendary actions between turns | Done ✅ (PR #42, 2026-02-23) |
| PR-12 | Opportunity attacks (reaction hook on movement) | Done ✅ (PR #43, 2026-02-23) |

## Phase 3 Tasks (after Phase 2)

| Task | Description | Status |
|------|-------------|--------|
| PR-13 | A* pathfinding on walkability grid | Done ✅ (PR #44, 2026-02-23) |
| PR-14 | Movement planner + Dash / Disengage universal actions | Planned |
| PR-15 | ReactionRule engine (Shield, Counterspell, Parry, etc.) | Planned |
| PR-16 | Focus fire / team tactics (intelligence-gated) | Planned |

---

## Test Fixtures (created across PRs)

| Fixture | Created in | Purpose |
|---------|-----------|---------|
| `zombie_fixture.go` | PR-4 | Low INT (3), melee, testing dumb behavior |
| `goblin_fixture.go` | PR-4 | Medium INT (10), melee + ranged, basic tactics |
| `dragon_fixture.go` | PR-4 | High INT (16), multiattack + recharge breath |
| `lich_fixture.go` | PR-5 | Max INT (20), full spellcaster, optimal play |
| `skeleton_archer_fixture.go` | PR-5 | Low INT (6), pure ranged |
