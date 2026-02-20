# Combat AI Design Plan

> **Type:** Feature Plan
> **Status:** Draft
> **Branch:** TBD (`feature/combat-ai-mvp`)
> **Last updated:** 2026-02-20

---

## A. Goal

Реализовать rule-based AI-модуль, который управляет NPC в бою: выбирает действия, цели и перемещение за монстров, позволяя одиночному игроку проходить энкаунтеры без живого DM.

---

## B. Target State

- **Полностью автономный AI** — игрок может проходить энкаунтеры один, без живого DM
- Два режима: **manual** (DM нажимает "AI Turn") и **auto-play** (бэкенд автоматически ходит за всех NPC)
- AI учитывает: `StructuredActions` + `Multiattacks`, `RuntimeState` (HP, ресурсы, перезарядки, слоты), позиции, resistances/immunities, координаты на сетке
- **Intelligence** каждого NPC привязан к INT существа — зомби (INT 3) тупой, лич (INT 20) играет оптимально
- Реакции NPC (opportunity attacks, Shield) обрабатываются автоматически
- Модуль реализован как Go-пакет внутри монолита, но за чистым интерфейсом `CombatAI`, пригодным для извлечения в gRPC-сервис или замены на RL
- Три фазы развития: MVP + Auto-Play → Тактика + Реакции → Перемещение + Командная тактика

---

## C. Scope

### In scope

- **Фаза 1 (MVP):** Классификация роли, выбор действия/цели, intelligence система, multiattack, Dodge, manual + auto-play режимы, turn management на бэкенде
- **Фаза 2:** Threat assessment, управление ресурсами кастеров, AoE-оптимизация, opportunity attacks (реакции как хук при перемещении), legendary actions
- **Фаза 3:** Перемещение на сетке (A* pathfinding), Dash/Disengage, ReactionRule engine, командная тактика, focus fire

### Out of scope

- Reinforcement Learning (будущее, после накопления данных боёв — см. секцию N)
- AI для игровых персонажей (PC) — только NPC/монстры
- Генерация encounter'ов (подбор монстров, балансировка CR)
- Автоматическая инициатива (initiative rolls) — задаётся при создании encounter'а

---

## D. Current State

### Что уже есть

| Компонент | Расположение | Что делает |
|-----------|-------------|------------|
| Action execution pipeline | `internal/pkg/actions/usecases/` | Исполняет `weapon_attack`, `spell_cast`, `use_feature`, `ability_check`, `saving_throw`, `custom_roll` — бросает кубики, считает урон, мутирует HP, записывает audit log |
| StructuredActions | `internal/models/action_structured.go` | Машиночитаемые действия существ: `AttackRollData`, `SavingThrowData`, `DamageRoll`, `ActionEffect`, `RechargeData`, `UsesData` |
| CreatureRuntimeState | `internal/models/creature_runtime.go` | Боевое состояние: HP, conditions, spell slots, ability uses, legendary actions, recharges, concentration |
| ParticipantFull | `internal/models/creature_runtime.go` | Полная модель участника боя: template ref + runtime state + позиция на сетке + initiative |
| Creature template | `internal/models/creature.go` | Полный stat block D&D 5e: abilities, AC, HP, movement, vision, resistances/immunities, spellcasting |
| Spellcasting model | `internal/models/spellcasting.go` | `Spellcasting` (slots) + `InnateSpellcasting` (at-will/per-day) + `SpellDefinition` с resolution/effects |
| Trigger engine | `internal/pkg/triggers/engine.go` | Обработка trigger effects на оружии (on_hit, on_critical и т.д.) |
| WebSocket table | `internal/pkg/table/` | In-memory сессия боя, broadcast state изменений всем клиентам |
| Audit log | `internal/pkg/actions/repository/` | MongoDB collection `action_log` с 30-day TTL |

### Чего не хватает

1. **Модуль принятия решений** — «мозг», который по текущему состоянию боя выбирает действие, цель и движение
2. **Классификация ролей** — определение тактического поведения по stat block'у существа
3. **Threat assessment** — оценка приоритетности целей
4. **Movement planning** — перемещение по сетке с учётом walkability

### Точки интеграции

```
=== MANUAL MODE ===                     === AUTO-PLAY MODE ===

[Frontend: DM нажимает "AI Turn"]       [Turn Manager (бэкенд)]
        │                                       │
        ▼                                       ▼
POST /api/encounter/{id}/ai-turn        Автоматически для каждого NPC
        │                                       │
        └──────────────┬────────────────────────┘
                       ▼
              ┌─────────────────────┐
              │   CombatAI module   │  ← НОВЫЙ МОДУЛЬ
              │  (DecideTurn)       │
              │  intelligence-gated │
              └─────────┬───────────┘
                        │ TurnDecision
                        ▼
              ┌─────────────────────────┐
              │ Action Execution Engine  │  ← УЖЕ ЕСТЬ
              │ (ExecuteAction)          │
              │ (× N для multiattack)   │
              └─────────┬───────────────┘
                        │ ActionResponse
                        ▼
              ┌─────────────────────────┐
              │ WebSocket broadcast      │  ← УЖЕ ЕСТЬ
              │ battleInfo + your_turn   │
              └─────────────────────────┘
```

---

## E. Design

### E.1 Service-Boundary-Ready Interface

Главный принцип: модуль AI зависит **только от моделей** (`internal/models`), не от конкретных репозиториев или usecases. Все данные о бое передаются в AI как готовая структура — это позволяет в будущем:
1. Вынести модуль в отдельный gRPC-сервис без изменения логики
2. Заменить rule-based движок на RL-модель за тем же интерфейсом

```go
// internal/pkg/combatai/interfaces.go

package combatai

import "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"

// CombatAI принимает снимок состояния боя и возвращает решение для одного NPC.
// Не имеет side effects — чистая функция принятия решений.
// Пригоден для извлечения в gRPC-сервис: вход и выход полностью сериализуемы.
type CombatAI interface {
    DecideTurn(input *TurnInput) (*TurnDecision, error)
}
```

### E.2 Core Data Structures

```go
// TurnInput — всё, что AI нужно знать для принятия решения.
// Полностью самодостаточная структура (no DB access needed).
type TurnInput struct {
    // Кто ходит
    ActiveNPC        models.ParticipantFull   // NPC, чей ход
    CreatureTemplate models.Creature          // полный stat block активного NPC из MongoDB

    // Состояние боя
    Participants     []models.ParticipantFull // все участники (включая ActiveNPC)
    CurrentRound     int

    // === Данные о целях (для расчёта expected damage) ===
    //
    // ParticipantFull содержит только runtime state (HP, conditions),
    // но не template data (AC, resistances). AI нужен доступ к AC,
    // saving throw бонусам, resistances/immunities/vulnerabilities целей.
    //
    // NPC и PC имеют РАЗНЫЕ модели данных:
    // - NPC: models.Creature (ArmorClass, SavingThrows, DamageResistances, Ability)
    // - PC:  models.DerivedStats (ArmorClass, SaveBonuses, Resistances — вычисляются
    //        из CharacterBase через compute.ComputeDerived())
    //
    // Для унификации доступа используем CombatantStats — упрощённый срез данных,
    // нужный AI для расчётов. Построение этой карты — ответственность usecases layer.
    CombatantStats   map[string]CombatantStats // key = ParticipantFull.InstanceID

    // Intelligence (0.05-1.0) — предвычислено из INT существа + aiDifficultyMod
    Intelligence     float64

    // Контекст предыдущего хода (для sticky targeting при intelligence < 0.15).
    // Заполняется usecases layer из audit log или in-memory state.
    // Пустая строка = нет предыдущей цели (первый раунд или цель мертва).
    PreviousTargetID string

    // Карта (для movement, Phase 3)
    MapWidth         int                      // ширина сетки в клетках
    MapHeight        int                      // высота сетки в клетках
    WalkabilityGrid  [][]bool                 // true = проходимо (nil = movement отключён)
}

// CombatantStats — унифицированный срез боевых характеристик участника.
// Абстрагирует разницу между NPC (Creature) и PC (CharacterBase + DerivedStats).
// Строится в usecases layer при формировании TurnInput.
type CombatantStats struct {
    AC               int               // ArmorClass (с учётом StatModifiers, если есть)
    SaveBonuses      map[string]int    // ability type ("STR","DEX",...) → total save bonus
    Resistances      []string          // damage types: "fire", "cold", etc.
    Immunities       []string          // damage type immunities
    Vulnerabilities  []string          // damage type vulnerabilities
    IsPC             bool              // true для PC (влияет на приоритизацию)
}

// TurnDecision — результат решения AI.
type TurnDecision struct {
    // Перемещение (nil = не двигаться)
    Movement    *MovementDecision

    // Основное действие (nil = пропуск хода, напр. если incapacitated)
    Action      *ActionDecision

    // Бонусное действие (nil = нет подходящего)
    BonusAction *ActionDecision

    // Правило для реакции (сохраняется до следующего хода NPC)
    // Phase 2+
    Reaction    *ReactionRule

    // Человекочитаемое объяснение решения (для DM лога)
    Reasoning   string
}

// ActionDecision — конкретное действие или multiattack последовательность,
// готовое к преобразованию в один или несколько ActionRequest.
type ActionDecision struct {
    // Одиночное действие (если MultiattackSteps пуст)
    ActionType     models.ActionType  // weapon_attack, spell_cast, use_feature, "" для multiattack
    ActionID       string             // ID из StructuredActions или SpellID
    TargetIDs      []string           // instance IDs целей
    SlotLevel      int                // для spell_cast: уровень слота (upcast)

    // Multiattack: если NPC использует multiattack, это массив шагов.
    // Каждый шаг — отдельная атака со своей целью.
    // Если MultiattackSteps не пуст, поля выше (ActionType/ActionID/TargetIDs) игнорируются.
    MultiattackGroupID string              // ID из MultiattackGroup (для лога)
    MultiattackSteps   []MultiattackStep   // nil = одиночное действие

    // Метаданные для лога
    ActionName     string
    ExpectedDamage float64            // оценка ожидаемого урона (суммарный для multiattack)
    Reasoning      string             // почему выбрано это действие
}

// MultiattackStep — один шаг внутри multiattack последовательности.
type MultiattackStep struct {
    ActionType models.ActionType  // weapon_attack
    ActionID   string             // ID конкретной атаки (напр. "bite", "claw")
    TargetIDs  []string           // цель этой атаки (может отличаться от других шагов)
}

// MovementDecision — куда двигаться.
type MovementDecision struct {
    TargetX   int
    TargetY   int
    Path      []models.CellsCoordinates  // путь по клеткам (для анимации)
    Reasoning string
}

// ReactionRule — условие для автоматического использования реакции.
type ReactionRule struct {
    ActionID  string             // ID реакции из StructuredActions
    Trigger   string             // "opportunity_attack", "shield_spell", etc.
    Condition string             // человекочитаемое условие
}
```

#### Расчёт дистанции

Дистанция между участниками считается по **Chebyshev distance** (D&D 5e standard grid):

```go
// DistanceFt возвращает дистанцию в футах между двумя участниками.
// Chebyshev: каждая клетка (включая диагональ) = 5 футов.
// Если у одного из участников нет координат (CellsCoords == nil) → MaxInt (бесконечность).
func DistanceFt(a, b *models.CellsCoordinates) int {
    if a == nil || b == nil { return math.MaxInt32 }
    dx := abs(a.CellsX - b.CellsX)
    dy := abs(a.CellsY - b.CellsY)
    return max(dx, dy) * 5  // 1 клетка = 5 футов
}
```

> Используется для: "ближайший враг", проверка reach/range, threat assessment distance penalty.

#### Конвертация ActionDecision → ActionCommand

AI выдаёт `ActionDecision`, но action execution pipeline принимает `models.ActionCommand`. Маппинг:

```go
// toActionCommand конвертирует решение AI в формат action pipeline.
// Для multiattack вызывается для каждого MultiattackStep отдельно.
func toActionCommand(d *ActionDecision, step *MultiattackStep) models.ActionCommand {
    // Если это шаг multiattack — берём данные из step
    actionType := d.ActionType
    actionID := d.ActionID
    targetIDs := d.TargetIDs
    if step != nil {
        actionType = step.ActionType
        actionID = step.ActionID
        targetIDs = step.TargetIDs
    }

    cmd := models.ActionCommand{Type: actionType}

    switch actionType {
    case models.ActionWeaponAttack:
        cmd.WeaponID = actionID            // ActionID = StructuredAction.ID
        if len(targetIDs) > 0 {
            cmd.TargetID = targetIDs[0]    // weapon_attack: single target
        }
    case models.ActionSpellCast:
        cmd.SpellID = actionID             // ActionID = SpellKnown.SpellID или Name
        cmd.SlotLevel = d.SlotLevel
        cmd.TargetIDs = targetIDs          // spell_cast: supports multi-target
    case models.ActionUseFeature:
        cmd.FeatureID = actionID
        if len(targetIDs) > 0 {
            cmd.TargetID = targetIDs[0]
        }
    }
    return cmd
}

// Для multiattack: исполнить все шаги последовательно
func executeMultiattack(d *ActionDecision, executor ActionsUsecases) []ActionResponse {
    var results []ActionResponse
    for _, step := range d.MultiattackSteps {
        cmd := toActionCommand(d, &step)
        req := models.ActionRequest{CharacterID: npcInstanceID, Action: cmd}
        resp, _ := executor.ExecuteAction(ctx, encounterID, req, systemUserID)
        results = append(results, resp)
    }
    return results
}
```

> **Примечание:** `ActionRequest.CharacterID` — это `ParticipantFull.InstanceID`, не CreatureID.
> `WeaponID` в existing pipeline ищет по `CharacterBase.Weapons[]` для PC. Для NPC-атак нужно расширить pipeline для поиска по `StructuredActions[].ID` — см. PR-6.

### E.3 Creature Role Classification

AI определяет тактическую роль существа по его stat block'у. Роль влияет на приоритеты при выборе действий и целей.

```go
type CreatureRole string

const (
    RoleBrute      CreatureRole = "brute"       // melee-ориентированный, высокий STR/HP
    RoleRanged     CreatureRole = "ranged"       // дальнобойные атаки, держит дистанцию
    RoleCaster     CreatureRole = "caster"       // есть Spellcasting, приоритет на заклинания
    RoleSkirmisher CreatureRole = "skirmisher"   // высокий DEX, hit-and-run
    RoleController CreatureRole = "controller"   // AoE/conditions, контроль поля
    RoleTank       CreatureRole = "tank"         // высокий AC/HP, защищает союзников
)
```

**Алгоритм классификации:**

```
1. Есть Spellcasting с attack/damage заклинаниями → Caster
2. Все атаки ranged и нет melee → Ranged
3. STR >= 16 И HP.Average >= parseCR(CR) * 15 → Brute
4. DEX >= STR + 4 И Movement.Walk >= 40 → Skirmisher
5. AC >= 18 ИЛИ (HP.Average >= parseCR(CR) * 20 И есть melee) → Tank
6. >= 2 StructuredActions с SavingThrow или ConditionEffect → Controller
7. Fallback: есть ranged → Ranged, иначе → Brute
```

> **Парсинг строковых полей:** несколько полей Creature хранятся как `string`, но используются как числа:
> - `ChallengeRating`: `"1/4"` → `0.25`, `"1/2"` → `0.5`, `"3"` → `3.0`. Функция `parseCR(cr string) float64` в `role_classifier.go`.
> - `ProficiencyBonus`: `"+2"` → `2`. Функция `parseProfBonus(pb string) int` в `role_classifier.go`.
> - `DamageRoll.DiceType`: `"d6"` → `6`, `"d8"` → `8`. Функция `parseDiceMax(dt string) int` в `expected_value.go` (для расчёта avg damage = diceCount × (diceMax+1)/2 + bonus).

Роль можно переопределить вручную (поле `aiRole` в creature template, опционально, Phase 2+).

### E.4 Target Selection

#### Phase 1: Intelligence-Gated Target Selection

Выбор цели зависит от `intelligence` NPC — тупые существа атакуют ближайшего, умные используют тактику:

```
if intelligence < 0.15:
    → Если PreviousTargetID != "" и цель жива → атаковать её (sticky targeting)
    → Иначе → атаковать ближайшего врага (IsPlayerCharacter == true)

elif intelligence < 0.35:
    → Атаковать ближайшего врага
    → Но: если враг в зоне досягаемости с HP < 25% → добить его

elif intelligence < 0.55:
    → Для melee NPC:
        1. Все враги в зоне досягаемости (reach)
        2. Из них: с наименьшим CurrentHP (добить)
        3. Если никого в зоне досягаемости: ближайший враг
    → Для ranged NPC:
        1. Все враги в зоне дальности атаки
        2. Из них: с наименьшим AC (легче попасть)
        3. Если никого в зоне дальности: ближайший враг

else (intelligence >= 0.55):
    → Использовать threat assessment (Phase 2, см. ниже)
    → Phase 1 fallback: как для 0.35-0.55, но бонус за low HP и концентрацию
```

> **Примечание:** Для всех уровней intelligence нужен доступ к AC и resistances целей — см. `CombatantStats` в `TurnInput` (секция E.2). Данные берутся по `InstanceID` участника.

#### Phase 2: Threat Assessment

```go
type ThreatScore struct {
    TargetID       string
    Score          float64
    Distance       int       // в клетках
    IsConcentrating bool
    HPPercent      float64
}
```

**Формула threat score:**

```
base = estimated_DPR(target)                    // оценка урона в раунд

// Бонусы
if target.Concentration != nil:
    base += 30                                   // сбить концентрацию = высокий приоритет
if target.HPPercent < 0.25:
    base += 25                                   // добить раненого
if target.HPPercent < 0.50:
    base += 10

// Штрафы
if distance > movement + reach:
    base -= 15                                   // далеко = менее приоритетно (для melee)

// Учёт resistances
if target_is_immune_to_our_main_damage_type:
    base -= 50
if target_is_resistant_to_our_main_damage_type:
    base -= 20
if target_is_vulnerable_to_our_main_damage_type:
    base += 20
```

### E.5 Action Selection

#### Phase 1: Expected Value Maximization

Для каждого доступного действия NPC вычислить **expected damage** и выбрать лучшее:

```
Для AttackRoll action (данные из StructuredAction.Attack):
  target_AC  = CombatantStats[targetID].AC
  hit_chance = (21 - (target_AC - Attack.Bonus)) / 20
  hit_chance = clamp(hit_chance, 0.05, 0.95)          // nat 1 always misses, nat 20 always hits
  avg_damage = sum(diceCount * (parseDiceMax(diceType) + 1) / 2 + bonus) for each Attack.Damage[]
  expected   = hit_chance * avg_damage + 0.05 * avg_damage   // crit adds extra dice roll

Для SavingThrow action (данные из StructuredAction.SavingThrow):
  target_save = CombatantStats[targetID].SaveBonuses[SavingThrow.Ability]
  fail_chance = (SavingThrow.DC - target_save - 1) / 20
  fail_chance = clamp(fail_chance, 0.05, 0.95)
  avg_damage  = sum(damage from SavingThrow.Damage[])
  half_on_success = (SavingThrow.OnSuccess == "half damage")
  expected = fail_chance * avg_damage + (1 - fail_chance) * (half_on_success ? avg_damage/2 : 0)
  // AoE бонус: expected *= number_of_targets_in_area
```

**Оценка заклинаний (Phase 1 — heuristic):**

`SpellKnown` в creature template содержит `Name` и `Level`, но **не содержит данных об уроне** — для точного расчёта нужна `SpellDefinition` из коллекции `spell_definitions` (MongoDB). Полная интеграция со spell database — Phase 2.

Phase 1 использует эвристику на основе `SpellQuickRef` (если заполнен) и уровня слота:

```
Для spell_attack заклинания (Spellcasting.SpellAttackBonus):
  hit_chance = (21 - (target_AC - spellAttackBonus)) / 20
  estimated_damage = spell_level * 3.5 + 3    // heuristic: примерно d6 per level + modifier
  expected = hit_chance * estimated_damage

Для save-or-damage заклинания:
  estimated_damage = spell_level * 4.5         // save spells обычно чуть мощнее
  fail_chance = (Spellcasting.SpellSaveDC - target_save_bonus - 1) / 20
  expected = fail_chance * estimated_damage + (1 - fail_chance) * estimated_damage / 2

Для cantrips (level 0):
  estimated_damage = 5.5                       // примерно 1d10 (Fire Bolt, Toll the Dead)
  Масштабирование: если CasterLevel >= 5 → ×2, >= 11 → ×3, >= 17 → ×4

Если SpellKnown.QuickRef != nil и QuickRef.Range != "Self":
  → заклинание считается damage-capable
Если QuickRef == nil:
  → пропустить (не можем оценить, fallback на weapon attacks)
```

> **Phase 2:** `buildTurnInput()` будет подгружать `SpellDefinition` для каждого spell в creature's Spellcasting и передавать точные данные об уроне, resolution type, AoE. Эвристика заменяется точным расчётом.

**Приоритизация (сверху вниз):**

```
1. Перезаряженная способность (Recharge ready) — всегда использовать,
   это самые мощные абилки (дыхание дракона и т.д.)

2. AoE действие, покрывающее >= 3 врагов
   expected_value = per_target_damage * num_targets

3. Действие с максимальным expected_value против лучшей цели

4. Если нет действий с уроном — Dodge (пропуск с бонусом к AC, Phase 1)
```

**Управление ресурсами (Phase 2):**

```
Spell slots:
  round <= 2 → можно тратить top-level слоты (AoE, control)
  round 3-5 → средние слоты
  round >= 6 → экономить, кантрипы + 1-2 уровень

  Если осталось < 30% HP → потратить лучший оставшийся слот

Limited uses (uses/day):
  Использовать если expected_value >= 150% лучшей обычной атаки

Legendary actions:
  Тратить каждый раунд (они восстанавливаются)
  Приоритет: damage > movement > detection
```

### E.6 Movement Planning

#### Phase 1: No Movement

MVP не включает перемещение — NPC атакует с текущей позиции. Если враг вне досягаемости, выбирается ranged-действие или пропуск хода.

#### Phase 3: Grid-Based Movement

```
Melee NPC:
  1. Если приоритетная цель в пределах reach → не двигаться
  2. Иначе → A* к ближайшей клетке в пределах reach от цели
  3. Ограничение: Movement.Walk / 5 клеток за ход

Ranged NPC:
  1. Если враг в melee range → отступить (Disengage если есть)
  2. Если цель вне дальности → приблизиться
  3. Оптимальная позиция: 60-80% от максимальной дальности

Caster:
  1. Максимизировать дистанцию до melee-врагов
  2. Позиционироваться для AoE (максимальное покрытие)
```

**A\* pathfinding:**
- Сетка: `WalkabilityGrid [][]bool`
- Стоимость: 1 за клетку (5 футов), diagonal = 1.5 (опционально)
- Сложность: O(V log V) где V = кол-во клеток. Для 100x100 = 10,000 клеток — микросекунды
- Occupied cells: участники блокируют клетки (кроме союзников, если используется правило сквозного прохода)

### E.7 Multiattack Model

> **Решение:** Отдельная структура `MultiattackGroup` на Creature — чистая модель, явно отражающая правила D&D.

Multiattack — мета-действие: "за одно Action сделай N конкретных атак в определённой комбинации" (напр. "3 атаки: 1 укус и 2 когтя"). Это не свойство отдельного `StructuredAction`, а отдельная сущность.

```go
// Новое поле на Creature (internal/models/creature.go):
type Creature struct {
    // ...existing fields...
    Multiattacks []MultiattackGroup `json:"multiattacks,omitempty" bson:"multiattacks,omitempty"`
}

// Определяет комбинацию атак за одно действие.
type MultiattackGroup struct {
    ID      string             `json:"id" bson:"id"`
    Name    string             `json:"name" bson:"name"`       // "Multiattack"
    Actions []MultiattackEntry `json:"actions" bson:"actions"`
}

// Ссылка на StructuredAction + количество повторений.
type MultiattackEntry struct {
    ActionID string `json:"actionId" bson:"actionId"` // -> StructuredAction.ID
    Count    int    `json:"count" bson:"count"`        // 2 для "2 когтя"
}
```

**AI логика:**
1. Есть `Multiattacks`? → Это приоритетное действие (вместо одиночных атак)
2. **Если несколько MultiattackGroup** (напр. "3 удара мечом ИЛИ 2 выстрела из лука"):
   → Вычислить expected value каждой группы = сумма EV компонентов
   → Выбрать группу с максимальным суммарным expected value
3. Вычислить expected value выбранной multiattack = сумма expected value каждого компонента
4. Сравнить с лучшим одиночным действием (напр. перезаряженное дыхание) — выбрать то, что выгоднее
5. Исполнить каждый компонент последовательно через action pipeline
6. Каждый компонент может иметь свою цель (bite → tank, claws → caster)

**Миграция:** ActionProcessorService (gRPC LLM) дополняется парсингом "Multiattack" из текстового поля `Actions[]` в `MultiattackGroup`.

### E.8 Intelligence System

> **Решение:** Intelligence привязан к INT существа + глобальный множитель на сессии. Определяет качество тактических решений AI.

```go
// Вычисление intelligence (0.0 - 1.0) из stat block:
func ComputeIntelligence(creature models.Creature, difficultyMod float64) float64 {
    base := clamp((float64(creature.Ability.Int) - 1.0) / 19.0, 0.05, 1.0)
    return clamp(base + difficultyMod, 0.05, 1.0)
}
```

**Что контролирует `intelligence`:**

| Intelligence | INT range | Поведение | Пример существа |
|-------------|-----------|-----------|-----------------|
| 0.05 - 0.15 | INT 1-3 | Атакует ближайшего, не переключает цели, не использует тактику | Gelatinous Cube (INT 1 → 0.05), Зомби (INT 3 → 0.11), Волк (INT 3 → 0.11) |
| 0.15 - 0.35 | INT 4-7 | Базовая тактика: добивает раненых, может убежать при низком HP | Огр (INT 5 → 0.21), Скелет (INT 6 → 0.26), Орк (INT 7 → 0.32) |
| 0.35 - 0.55 | INT 8-11 | Нормальная тактика: выбирает лучшее действие, переключает цели | Гноллы (INT 8 → 0.37), Гоблин (INT 10 → 0.47) |
| 0.55 - 0.75 | INT 12-15 | Умная тактика: focus fire, сбивает концентрацию, управляет ресурсами | Hobgoblin Captain (INT 12 → 0.58), Cambion (INT 15 → 0.74) |
| 0.75 - 1.0 | INT 16-20 | Полная тактика: AoE-оптимизация, resource management, командная работа | Дракон (INT 16 → 0.79), Вампир (INT 17 → 0.84), Beholder (INT 17 → 0.84), Лич (INT 20 → 1.0) |

**Механизм влияния на решения:**

```
При выборе действия:
  if rand() < intelligence:
      → выбрать оптимальное действие (maximum expected value)
  else:
      → выбрать случайное из допустимых действий

При выборе цели:
  if intelligence >= 0.55:
      → использовать threat assessment (концентрация, HP%, DPR)
  elif intelligence >= 0.35:
      → добивать раненых, иначе ближайший
  else:
      → ближайший враг

При управлении ресурсами (Phase 2):
  if intelligence >= 0.75:
      → полное управление spell slots (экономия по раундам)
  elif intelligence >= 0.55:
      → базовое управление (не тратить top-level слот в первый раунд)
  else:
      → тратить лучший доступный ресурс сразу
```

**Глобальный множитель** настраивается при создании сессии:

```json
POST /api/table/session
{
  "encounterID": "xxx",
  "aiAutoPlay": true,
  "aiDifficultyMod": 0.0    // -0.5..+0.5, корректирует intelligence всех NPC
}
```

### E.9 Universal Actions (Dodge / Dash / Disengage)

> **Решение:** Захардкодить в AI как правила игры, не в данные существ.

Dodge, Dash, Disengage — стандартные действия D&D, доступные **каждому** существу. Записывать их в `StructuredActions` = дублирование данных. Это правила, а не свойства.

```go
// В ai_engine.go — встроенные действия D&D:
var universalActions = []UniversalAction{
    {
        ID:       "dodge",
        Name:     "Dodge",
        Category: ActionCategoryAction,
        // Эффект: disadvantage на атаки по NPC до следующего хода.
        // AI использует когда: HP < 25% И нет хороших атак
        // Доступен: Phase 1+
    },
    {
        ID:       "dash",
        Name:     "Dash",
        Category: ActionCategoryAction,
        // Эффект: удвоить movement в этот ход.
        // AI использует когда: цель вне movement range, Dash достаточен для подхода
        // Доступен: Phase 3 (нужен movement)
    },
    {
        ID:       "disengage",
        Name:     "Disengage",
        Category: ActionCategoryAction,
        // Эффект: перемещение не провоцирует opportunity attacks.
        // AI использует когда: ranged NPC в melee range врага, хочет отступить
        // Доступен: Phase 3 (нужен movement + reactions)
    },
}
```

**Phase 1:** Только Dodge — если NPC при низком HP и нет выгодных атак, Dodge вместо бесполезного действия.

**Исполнение Dodge через pipeline:**

Existing `ActionType` не включает `dodge`. Решение: **не проходить через action pipeline** — Dodge не бросает кубики и не наносит урон. AI engine обрабатывает Dodge напрямую:

```go
// В executeDecision():
if decision.Action != nil && decision.Action.ActionID == "dodge" {
    // Добавить StatModifier: disadvantage на атаки по этому NPC до его следующего хода
    addStatModifier(npc, StatModifier{
        Name:      "Dodge",
        Modifiers: []ModifierEffect{{Target: ModTargetAC, Operation: ModOpAdvantage}},
        Duration:  DurationUntilTurn,
    })
    // Записать в audit log и broadcast
    broadcastDodge(npc)
    return // не вызывать ExecuteAction()
}
```

> **Примечание:** В D&D 5e Dodge даёт disadvantage на атаки **по NPC**, а не бонус к AC. Существующий `ModifierEffect` с `ModOpAdvantage` на `ModTargetAC` — приближение. Полная реализация Dodge может потребовать отдельного поля `DodgeActive bool` в `CreatureRuntimeState`, проверяемого в action pipeline при бросках атаки. Детали определяются при реализации PR-5.

**Phase 3:** Dash и Disengage — когда появится movement и opportunity attacks.

### E.10 Reaction System

> **Решение:** Поэтапное внедрение — Phase 2 opportunity attacks как хук, Phase 3 полный ReactionRule engine.

**Phase 2: Opportunity Attacks (хук при перемещении)**

Не полная event-driven система, а проверка при обработке перемещения:

```
Когда участник перемещается (обновление cellsCoords):
  → Для каждого NPC в reach от стартовой позиции перемещаемого:
    → Есть ли StructuredAction с category=reaction?
    → RuntimeState.Resources.ReactionUsed == false?
    → intelligence check: rand() < intelligence? (тупые монстры могут пропустить)
    → Если все проверки пройдены → автоматически исполнить opportunity attack
    → Установить ReactionUsed = true
```

**Phase 3: ReactionRule Engine**

AI в свой ход регистрирует правила реакций:

```go
type ReactionTrigger string

const (
    TriggerOnEnemyMove   ReactionTrigger = "enemy_leaves_reach"  // opportunity attack
    TriggerOnEnemyAttack ReactionTrigger = "enemy_attacks_self"  // Shield, Parry
    TriggerOnEnemyCast   ReactionTrigger = "enemy_casts_spell"   // Counterspell
    TriggerOnAllyDamaged ReactionTrigger = "ally_takes_damage"   // Protection
)
```

Каждое действие (атака, каст, перемещение) проверяет зарегистрированные правила всех NPC.

### E.11 Auto-Play Mode & Turn Management

> **Решение:** Два режима — manual (фронтенд управляет ходами) и auto-play (бэкенд управляет NPC-ходами автоматически).

**Manual mode (default):**
- Фронтенд авторитетен для `currentRound`/`currentTurnIndex`
- DM вызывает `POST /api/encounter/{id}/ai-turn` для каждого NPC
- Получает результат, сам продвигает turn index

**Auto-play mode:**
- Включается при создании WebSocket-сессии: `aiAutoPlay: true`
- Бэкенд управляет turn order для NPC-ходов
- Цикл:

```
Бэкенд проверяет initiative order:
  NPC? → CombatAI.DecideTurn() → ExecuteAction() → broadcast → next turn
  NPC? → CombatAI.DecideTurn() → ExecuteAction() → broadcast → next turn
  PC?  → broadcast "your_turn" → ждём action от игрока через WebSocket/HTTP
  NPC? → CombatAI.DecideTurn() → ...
  Все походили? → currentRound++ → начинаем заново
```

**Turn management логика (auto-play):**

> **Важно:** `s.participants` должен быть отсортирован по `Initiative` (по убыванию) при инициализации сессии. `ParticipantFull.Initiative` — это `int`, задаётся при создании encounter'а. Сортировка выполняется один раз при старте auto-play, а не на каждом ходу.

```go
// Вызывается один раз при старте auto-play сессии.
func (s *session) initTurnOrder() {
    sort.SliceStable(s.participants, func(i, j int) bool {
        // По убыванию initiative. При равенстве — NPC ходят первыми (convention).
        if s.participants[i].Initiative != s.participants[j].Initiative {
            return s.participants[i].Initiative > s.participants[j].Initiative
        }
        return !s.participants[i].IsPlayerCharacter
    })
    s.currentTurnIndex = -1  // advanceTurn() начнёт с 0
    s.currentRound = 0       // станет 1 при первом проходе через index 0
}

func (s *session) advanceTurn() {
    for {
        s.currentTurnIndex = (s.currentTurnIndex + 1) % len(s.participants)
        if s.currentTurnIndex == 0 {
            s.currentRound++     // новый раунд: 0→1 (первый), 1→2, ...
        }

        // Проверка завершения боя
        if s.checkCombatEnd() { return }

        p := s.participants[s.currentTurnIndex]

        // Пропуск мёртвых и incapacitated.
        // По правилам D&D 5e, ConditionIncapacitated включается в:
        // Stunned, Paralyzed, Petrified, Unconscious — поэтому одной
        // проверки ConditionIncapacitated достаточно для всех этих условий.
        if p.RuntimeState.CurrentHP <= 0 { continue }
        if hasCondition(p, ConditionIncapacitated) { continue }

        // Начало хода: обработка start-of-turn эффектов
        s.processStartOfTurn(p)

        if p.IsPlayerCharacter {
            s.broadcastYourTurn(p)  // ждём действие игрока
            return
        }

        // NPC → автоматический ход
        decision := s.combatAI.DecideTurn(buildTurnInput(s, p))
        s.executeDecision(decision)
        s.broadcastBattleInfo()
        // Цикл продолжается к следующему участнику
    }
}

// processStartOfTurn обрабатывает эффекты начала хода ПЕРЕД принятием решения AI.
// Критично для корректной работы AI — например, если Recharge не обновлён,
// AI не увидит готовое дыхание дракона.
func (s *session) processStartOfTurn(p *ParticipantFull) {
    // 1. Recharge roll: для каждого действия с Recharge, бросить d6
    for _, action := range creatureTemplate.StructuredActions {
        if action.Recharge == nil { continue }
        if p.RuntimeState.Resources.RechargeReady[action.ID] { continue } // уже заряжено
        roll := dice.Roll(1, 6)
        if roll >= action.Recharge.MinRoll {
            p.RuntimeState.Resources.RechargeReady[action.ID] = true
        }
    }

    // 2. Legendary actions: восстановить до максимума (D&D 5e: восстанавливаются в начале хода существа)
    // (Legendary actions восстанавливаются в начале хода существа, не в начале раунда)

    // 3. Condition duration: уменьшить roundsLeft для conditions с timing "start_of_turn"
    // 4. Ongoing damage: применить эффекты с trigger "start_of_turn"
    // 5. Сбросить ReactionUsed = false (реакция восстанавливается в начале хода)
    p.RuntimeState.Resources.ReactionUsed = false
}
```

**Условия завершения боя (auto-play):**

Auto-play должен детектировать конец боя и остановить turn loop:

```
Проверка после каждого действия:
  1. TPK (Total Party Kill): все PC мертвы (CurrentHP <= 0) → broadcast "combat_end" { result: "defeat" }
  2. Victory: все NPC мертвы (CurrentHP <= 0) → broadcast "combat_end" { result: "victory" }
  3. Timeout: currentRound > maxRounds (настраиваемо, default 100) → broadcast "combat_end" { result: "timeout" }
  4. Deadlock: все оставшиеся NPC не могут атаковать (нет действий, все враги вне зоны, Phase 1 без movement) → пропуск хода + warning
```

```json
{
  "type": "combat_end",
  "data": {
    "result": "victory",   // "victory" | "defeat" | "timeout"
    "round": 5,
    "summary": "All enemies defeated in 5 rounds"
  }
}
```

**WebSocket message для PC-хода (auto-play):**
```json
{
  "type": "your_turn",
  "data": {
    "participantID": "fighter-1",
    "round": 3,
    "turnIndex": 5
  }
}
```

---

## F. Implementation: Folder Structure

```
internal/pkg/combatai/
├── interfaces.go           # CombatAI interface, TurnInput, TurnDecision
├── models.go               # CreatureRole, ThreatScore, ActionCandidate, UniversalAction
├── role_classifier.go      # ClassifyRole(creature) → CreatureRole
├── intelligence.go         # ComputeIntelligence(creature, mod) → float64
├── target_selector.go      # SelectTarget(input, action, intelligence) → targetIDs
├── action_selector.go      # SelectAction(input, role, intelligence) → ActionDecision
├── multiattack.go          # EvaluateMultiattack(group, targets) → ActionDecision
├── expected_value.go       # ComputeExpectedDamage(action, target) → float64
├── universal_actions.go    # Dodge/Dash/Disengage logic
├── resource_manager.go     # ShouldSpendResource(slot/uses, round, hpPercent, intelligence) → bool  (Phase 2)
├── reaction_checker.go     # CheckOpportunityAttacks(mover, npcs) → []ActionDecision  (Phase 2)
├── movement_planner.go     # PlanMovement(input, role, targetPos) → MovementDecision  (Phase 3)
├── pathfinding.go          # AStar(grid, from, to) → path  (Phase 3)
├── ai_engine.go            # RuleBasedAI struct implementing CombatAI
└── ai_engine_test.go       # Unit tests
```

**Delivery layer (HTTP endpoint):**

```
internal/pkg/combatai/delivery/
└── combat_ai_handlers.go   # POST /api/encounter/{id}/ai-turn, ai-round
```

**Integration glue (собирает данные, управляет auto-play):**

```
internal/pkg/combatai/usecases/
├── combat_ai_usecases.go   # buildTurnInput(), executeTurn(), конвертация ActionDecision → ActionCommand
└── turn_manager.go         # Auto-play turn loop, advanceTurn(), processStartOfTurn(), broadcastYourTurn()
```

**Зависимости `combat_ai_usecases.go` (инжектируются через конструктор):**

```go
type CombatAIUsecases struct {
    ai               CombatAI                    // rule-based engine
    encounters       encounter.EncounterRepository // загрузить encounter JSON blob, распарсить participants
    bestiary         bestiary.BestiaryRepository   // загрузить Creature template по CreatureID (MongoDB)
    characters       character.CharacterRepository // загрузить CharacterBase для PC participants
    actions          actions.ActionsUsecases        // ExecuteAction() — исполнение через existing pipeline
    table            table.TableManager             // BroadcastToEncounter() — WebSocket broadcast
}
```

**`buildTurnInput()` — алгоритм:**

```
1. Загрузить encounter → распарсить participants из JSON blob
2. Для каждого participant:
   a. NPC (IsPlayerCharacter == false):
      → Загрузить Creature template по CreatureID из bestiary
      → CombatantStats: AC из creature.ArmorClass, saves из creature.SavingThrows + Ability, resistances/immunities
   b. PC (IsPlayerCharacter == true):
      → Загрузить CharacterBase по CharacterRuntime.CharacterID из characters
      → derived := compute.ComputeDerived(charBase)
      → CombatantStats: AC из derived.ArmorClass, saves из derived.SaveBonuses, resistances/immunities
3. ComputeIntelligence(activeNPC_creature, session.aiDifficultyMod)
4. PreviousTargetID: из in-memory session state (last target of this NPC)
5. Walkability grid: из encounter map data (Phase 3, nil в Phase 1)
```

---

## G. API Endpoint

### Session creation (extended)

`POST /api/table/session` — существующий endpoint, расширить request body:

```json
{
  "encounterID": "xxx",
  "aiAutoPlay": false,       // NEW: автоматические ходы NPC (default: false)
  "aiDifficultyMod": 0.0     // NEW: -0.5..+0.5, корректирует intelligence всех NPC (default: 0.0)
}
```

### `POST /api/encounter/{encounterID}/ai-turn`

**Авторизация:** LoginRequired + владелец encounter'а (DM)

**Request body:**
```json
{
  "npcID": "goblin-1"    // instanceID участника (ParticipantFull.InstanceID)
}
```

**Response (200 OK):**
```json
{
  "decision": {
    "movement": {
      "targetX": 5,
      "targetY": 3,
      "path": [{"cellsX": 4, "cellsY": 4}, {"cellsX": 5, "cellsY": 3}],
      "reasoning": "Move closer to Fighter"
    },
    "action": {
      "actionType": "weapon_attack",
      "actionID": "longsword",
      "actionName": "Longsword",
      "targetIDs": ["fighter-1"],
      "expectedDamage": 8.5,
      "reasoning": "Best melee attack against nearest enemy"
    },
    "bonusAction": null,
    "reasoning": "Brute role: attack nearest threat with highest-damage weapon"
  },
  "actionResult": {
    "rollResult": {"expression": "1d20+5", "rolls": [14], "modifier": 5, "total": 19},
    "damageRolls": [{"expression": "1d8+3", "rolls": [6], "modifier": 3, "total": 9, "damageType": "slashing"}],
    "stateChanges": [{"targetId": "fighter-1", "hpDelta": -9, "description": "Longsword hit: 9 slashing damage"}],
    "summary": "Goblin Boss attacks Fighter with Longsword: 19 vs AC 18 — HIT for 9 slashing damage",
    "hit": true
  }
}
```

**Error responses:**

| Code | Condition |
|------|-----------|
| 400 | `npcID` не указан или не найден среди participants |
| 400 | Участник — PC (`IsPlayerCharacter == true`) |
| 400 | NPC мёртв (HP <= 0) или incapacitated |
| 403 | Пользователь не владелец encounter'а |
| 404 | Encounter не найден |
| 422 | У существа нет `StructuredActions` (legacy text-only creature) |

### `POST /api/encounter/{encounterID}/ai-round` (Phase 1)

Выполняет автоматический ход **всех** NPC в порядке инициативы. Возвращает массив решений.

**Request body:** нет (все NPC ходят)

**Response (200 OK):**
```json
{
  "round": 3,
  "turns": [
    {
      "npcID": "goblin-1",
      "npcName": "Goblin",
      "decision": { "action": { "...same as ai-turn..." }, "reasoning": "..." },
      "actionResults": [
        { "rollResult": { "..." }, "damageRolls": [ "..." ], "summary": "...", "hit": true }
      ],
      "skipped": false,
      "skipReason": ""
    },
    {
      "npcID": "zombie-2",
      "npcName": "Zombie",
      "decision": null,
      "actionResults": null,
      "skipped": true,
      "skipReason": "Dead (HP <= 0)"
    }
  ],
  "combatEnded": false,
  "combatResult": ""
}
```

> **Примечание:** `actionResults` — массив, потому что multiattack порождает несколько `ActionResponse`. Для одиночного действия массив содержит 1 элемент. `skipped: true` — NPC пропущен (мёртв, incapacitated).

---

## H. Computational Complexity Analysis

### Обоснование: модуль внутри монолита, не отдельный сервис

| Операция | Сложность | Оценка времени |
|----------|-----------|----------------|
| Role classification | O(1) — проверка ~7 условий | < 1 мкс |
| Target selection (Phase 1) | O(P) — перебор участников | < 10 мкс для 20 участников |
| Threat assessment (Phase 2) | O(P) с несколькими множителями | < 20 мкс |
| Expected damage (1 action) | O(D) — сумма по damage rolls | < 1 мкс |
| Action selection | O(A * P) — действия * цели | < 50 мкс для 6 действий * 8 целей |
| A\* pathfinding (Phase 3) | O(V log V), V = cells | < 1 мс для 100x100 сетки |
| **Итого на 1 NPC** | | **< 0.1 мс (Phase 1-2), < 2 мс (Phase 3)** |
| **Полный раунд (20 NPC)** | | **< 2 мс (Phase 1-2), < 40 мс (Phase 3)** |

Для сравнения:
- Один запрос PostgreSQL: **1-5 мс**
- Один запрос MongoDB: **1-10 мс**
- JSON marshal/unmarshal encounter data: **0.1-1 мс**
- HTTP round-trip (если бы был отдельный сервис): **5-20 мс**

**Вывод:** вычислительная нагрузка AI пренебрежимо мала по сравнению с I/O. Выделение в отдельный сервис добавит latency, не решая реальной проблемы. Горизонтальное масштабирование не нужно.

**Извлекаемость:** интерфейс `CombatAI` не зависит от DB/HTTP/context — принимает `TurnInput`, возвращает `TurnDecision`. При необходимости:
1. Обернуть в gRPC-сервер (protobuf-сериализация `TurnInput`/`TurnDecision`)
2. В монолите заменить direct call на gRPC client
3. Логика AI не меняется

---

## I. Phase Breakdown

### Phase 1: MVP — Автономный AI с базовой тактикой

**Scope:**
- `ClassifyRole()` — определение роли по stat block
- `ComputeIntelligence()` — intelligence из INT существа + глобальный множитель
- `SelectTarget()` — выбор цели с учётом intelligence (ближайший для тупых, min AC/HP для умных)
- `SelectAction()` — выбор действия по expected value с intelligence gate
- `EvaluateMultiattack()` — поддержка multiattack (исполнение компонентов последовательно)
- `ComputeExpectedDamage()` — расчёт математического ожидания урона
- Dodge — universal action при низком HP и отсутствии хороших атак
- `POST /api/encounter/{id}/ai-turn` endpoint (manual mode)
- Auto-play mode: `aiAutoPlay` при создании сессии, turn management на бэкенде
- `POST /api/encounter/{id}/ai-round` — автоматический ход всех NPC за раунд
- `MultiattackGroup` модель + миграция существ
- Unit tests для каждого компонента

**Ограничения Phase 1:**
- Без перемещения (NPC бьёт с текущей позиции)
- Без реакций (opportunity attacks)
- Без бонусных действий
- Без управления spell slots (кастеры используют лучший доступный кантрип/слот)
- Без AoE-оптимизации (AoE действия оцениваются по single-target damage)

**Acceptance criteria:**
- [ ] Гоблин с мечом (INT 10) атакует ближайшего PC
- [ ] Зомби (INT 3) атакует случайного врага в зоне досягаемости
- [ ] Лич (INT 20) выбирает оптимальное действие и лучшую цель
- [ ] Дракон использует перезаряженное дыхание, если оно готово
- [ ] Дракон использует multiattack (bite + 2 claws) если дыхание не перезарядилось
- [ ] Скелет-лучник стреляет по цели с наименьшим AC
- [ ] NPC при HP < 25% без хороших атак использует Dodge
- [ ] Auto-play: NPC ходят автоматически, PC получает `your_turn` через WebSocket
- [ ] Manual: DM вызывает `ai-turn`, фронтенд продвигает ход
- [ ] `ai-round` выполняет ходы всех NPC последовательно
- [ ] `aiDifficultyMod: -0.3` делает всех NPC тупее
- [ ] NPC без StructuredActions возвращает ошибку 422
- [ ] Мёртвый NPC возвращает ошибку 400
- [ ] Результат действия корректно мутирует HP через action pipeline
- [ ] Audit log записывает действие AI
- [ ] WebSocket broadcast отправляет обновлённое состояние

### Phase 2: Тактический AI — Ресурсы и реакции

**Scope:**
- Threat assessment с приоритизацией (концентрация, HP%, DPR) — для NPC с intelligence >= 0.55
- Управление spell slots: экономия по раундам, upcast, выбор уровня слота — intelligence-gated
- AoE-оптимизация: подсчёт целей в зоне поражения по координатам
- Добивание: приоритет целей с < 25% HP — для intelligence >= 0.35
- Бонусные действия
- Legendary actions (тратить между ходами)
- Opportunity attacks как хук при перемещении (автоматические реакции NPC)

**Acceptance criteria:**
- [ ] Кастер (INT 18) экономит высокие слоты на первых раундах
- [ ] Кастер (INT 6) тратит лучший слот сразу
- [ ] AI с intelligence >= 0.55 приоритетно бьёт по цели с концентрацией
- [ ] Fire Breath используется когда >= 3 цели в конусе (intelligence >= 0.75)
- [ ] AI с intelligence >= 0.35 добивает цель с 5 HP
- [ ] Legendary actions тратятся каждый раунд
- [ ] При перемещении PC мимо NPC — автоматический opportunity attack
- [ ] NPC с intelligence < 0.15 может пропустить opportunity attack

### Phase 3: Полный AI — Перемещение и командная работа

**Scope:**
- A* pathfinding по walkability grid
- Movement decision: подход в melee, отступление для ranged, позиционирование для AoE
- Dash и Disengage как universal actions
- ReactionRule engine: Shield, Counterspell, Parry, Protection
- Opportunity attack avoidance (NPC учитывает зоны контроля врагов при перемещении)
- Командная тактика: focus fire (все бьют одну цель) — для intelligence >= 0.55
- Personality profiles: aggressive / cautious / cunning (опционально)

**Acceptance criteria:**
- [ ] Melee NPC подходит к врагу, если тот вне досягаемости
- [ ] Ranged NPC использует Disengage и отступает от melee-врага
- [ ] NPC обходит непроходимые клетки
- [ ] NPC с intelligence >= 0.55 не провоцирует opportunity attack без необходимости
- [ ] NPC с intelligence < 0.35 может пойти напрямик через зону контроля
- [ ] Несколько гоблинов с intelligence >= 0.55 фокусируют одного PC
- [ ] Лич с Shield автоматически блокирует атаку, если Shield подготовлен как reaction

---

## J. Test Strategy

### Unit tests (каждый компонент отдельно)

| Компонент | Тест-кейсы |
|-----------|------------|
| `ClassifyRole` | Brute (огр), Ranged (скелет-лучник), Caster (лич), Skirmisher (ассасин), Tank (animated armor), Controller (beholder) |
| `ComputeIntelligence` | INT 1 → 0.05, INT 3 → 0.11, INT 10 → 0.47, INT 20 → 1.0, difficultyMod -0.3 (INT 10 → 0.17), difficultyMod +0.5 (INT 3 → 0.61, clamped 0.61) |
| `BuildCombatantStats` | NPC creature → AC/saves/resistances из Creature, PC → AC/saves из DerivedStats, отсутствующий save → ability modifier only |
| `parseCR` / `parseDiceMax` | CR "1/4" → 0.25, CR "1/2" → 0.5, DiceType "d6" → 6, "d10" → 10 |
| `ComputeExpectedDamage` | Melee attack vs AC 15, Ranged attack vs AC 20, SavingThrow DC 15 vs +2 save, AoE vs 3 targets |
| `SelectTarget` | intelligence < 0.15 → случайный, >= 0.35 → добивает раненого, >= 0.55 → threat assessment |
| `SelectAction` | Recharge ready → recharge action, No recharge → highest expected, No actions → error |
| `SelectAction` (intelligence) | intelligence = 0.1 → может выбрать неоптимальное, intelligence = 1.0 → всегда оптимальное |
| `SelectAction` (caster) | Has slots → spell, No slots → cantrip, Innate at-will → use it |
| `EvaluateMultiattack` | Multiattack (bite + 2 claws) vs single best attack, пустой multiattack → fallback |
| `UniversalActions` | HP < 25% и нет хороших атак → Dodge, HP > 50% → не Dodge |

### Integration tests

| Сценарий | Проверки |
|----------|----------|
| Goblin vs Fighter | AI выбирает scimitar, правильный расчёт hit/damage |
| Zombie (INT 3) vs Party | Атакует случайного, не переключает цели |
| Dragon vs Party of 3 | Fire Breath (если recharged), иначе multiattack (bite + 2 claws) |
| Lich (INT 20) vs Party | Оптимальный выбор, Round 1: high-level spell |
| Auto-play round | NPC ходят автоматически, PC получает your_turn |
| Manual ai-turn | DM вызывает, получает результат |
| ai-round | Все NPC ходят, возвращается массив решений |
| Dead NPC | 400 error |
| PC target | 400 error |
| No StructuredActions | 422 error |

### Тестовые данные

Создать fixtures для типовых существ:
- `zombie_fixture.go` — тупой melee (INT 3), тестирование низкого intelligence
- `goblin_fixture.go` — простой melee + ranged (INT 10), базовая тактика
- `dragon_fixture.go` — multiattack + recharge breath (INT 16), высокий intelligence
- `lich_fixture.go` — full spellcaster (INT 20), максимальный intelligence
- `skeleton_archer_fixture.go` — pure ranged (INT 6)

---

## K. PR Plan

### Phase 1 PRs

| PR | Description | Dependencies | Key Files |
|----|-------------|--------------|-----------|
| PR-1 | MultiattackGroup model + Creature migration | None | `models/creature.go`, `models/multiattack.go`, migration script |
| PR-2 | Core interfaces, models, intelligence system | PR-1 | `combatai/interfaces.go`, `combatai/models.go`, `intelligence.go` |
| PR-3 | Role classifier + expected value calculator | PR-2 | `role_classifier.go`, `expected_value.go`, tests |
| PR-4 | Target selector + action selector + multiattack evaluator | PR-3 | `target_selector.go`, `action_selector.go`, `multiattack.go`, tests |
| PR-5 | AI engine + universal actions (Dodge) | PR-4 | `ai_engine.go`, `universal_actions.go`, integration tests |
| PR-6 | HTTP endpoint `ai-turn` + usecases glue | PR-5 | `delivery/`, `usecases/combat_ai_usecases.go`, router |
| PR-7 | Auto-play mode + turn manager + `ai-round` | PR-6 | `usecases/turn_manager.go`, session extension, WebSocket `your_turn` |

```
PR-1 → PR-2 → PR-3 → PR-4 → PR-5 → PR-6 → PR-7
```

### Phase 2 PRs (after Phase 1 stabilizes)

| PR | Description |
|----|-------------|
| PR-8 | Threat assessment (intelligence-gated) |
| PR-9 | Spell slot management / resource manager |
| PR-10 | AoE target counting (geometry on grid) |
| PR-11 | Bonus actions + legendary actions |
| PR-12 | Opportunity attacks (reaction hook on movement) |

### Phase 3 PRs (after Phase 2)

| PR | Description |
|----|-------------|
| PR-13 | A* pathfinding |
| PR-14 | Movement planner + Dash/Disengage |
| PR-15 | ReactionRule engine (Shield, Counterspell, etc.) |
| PR-16 | Focus fire / team tactics (intelligence-gated) |

---

## L. Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Существа без `StructuredActions` | AI не может работать с legacy text-only creatures | Возвращать 422 с понятным сообщением; приоритизировать парсинг через ActionProcessorService |
| Существа без `Multiattacks` | AI не сможет делать multiattack | Fallback: использовать лучшую одиночную атаку. Приоритизировать миграцию. |
| Некорректные `StructuredActions` (пустой damage, нулевой bonus) | AI выберет неоптимальное действие или запаникует | Валидация + fallback на действие с максимальным бонусом к атаке |
| Гонка: AI-ход + ручной ход одновременно | Lost update на encounter data (уже известная проблема, см. action pipeline) | Auto-play mode: mutex per session (бэкенд контролирует очерёдность). Manual mode: документировать ограничение. |
| Intelligence делает бой нечестным | Зомби слишком тупой → скучно; Лич слишком умный → unfun | `aiDifficultyMod` позволяет игроку настроить. Тестировать на реальных энкаунтерах. |
| Auto-play: бэкенд берёт turn management | Дублирование логики с фронтендом, рассинхрон | Auto-play = бэкенд авторитетен. Manual = фронтенд авторитетен. Чёткое разделение, не гибрид. |
| AoE geometry на гексагональной сетке | Расчёт покрытия зависит от типа сетки | Phase 1: не считать AoE. Phase 2: параметр grid type (square/hex) |
| Encounter JSON blob не содержит walkability | Movement невозможен | Phase 3: walkability передаётся как часть TurnInput; если нет — movement отключён |

---

## M. Resolved Decisions

Следующие вопросы были обсуждены и решены 2026-02-20:

### 1. Multiattack → Отдельная модель `MultiattackGroup` на Creature

**Решение:** Multiattack — мета-действие D&D ("за одно Action сделай N атак в определённой комбинации"). Это первоклассная сущность, не костыль внутри `StructuredAction`.

Новая структура `MultiattackGroup` + `MultiattackEntry` на модели `Creature`. Миграция данных: ActionProcessorService дополняется парсингом "Multiattack" из текстовых Action'ов.

**Обоснование:** Чистая модель лучше компромисса. Миграция данных допустима.

Подробности: секция E.7.

### 2. Auto mode → Оба режима: manual + auto-play

**Решение:** Manual (DM нажимает "AI Turn") + auto-play (бэкенд автоматически ходит за NPC). Оба режима с Phase 1.

Включается при создании сессии: `aiAutoPlay: true`. В auto-play бэкенд управляет turn order, в manual — фронтенд.

**Обоснование:** Цель — полностью автономный AI для solo play. Manual нужен для режима "DM + игроки".

Подробности: секция E.11.

### 3. Difficulty → Intelligence из INT существа + глобальный множитель

**Решение:** `intelligence = clamp((INT - 1) / 19, 0.05, 1.0) + aiDifficultyMod`. Зомби (INT 3) = 0.1, лич (INT 20) = 1.0.

Intelligence контролирует: вероятность оптимального решения, глубину target selection, управление ресурсами.

**Обоснование:** Без DM некому "поддаваться". Intelligence из INT — тематически правильно (тупые монстры тупые, умные — хитрые) и даёт естественную градацию сложности. Глобальный множитель для тюнинга.

Подробности: секция E.8.

### 4. Reactions → Поэтапно: Phase 2 opportunity attacks, Phase 3 ReactionRule engine

**Решение:**
- **Phase 2:** Opportunity attacks как хук при обработке перемещения. Не полный event bus, а проверка NPC в reach при изменении `cellsCoords`. Intelligence gate — тупые монстры могут пропустить.
- **Phase 3:** Полный ReactionRule engine — AI регистрирует правила реакций (Shield, Counterspell, Parry), каждое действие проверяет триггеры.

**Обоснование:** Для автономного AI реакции обязательны (opportunity attack — ключевая механика). Но полный event bus — Phase 3 объём. Хук при перемещении покрывает 80% случаев.

Подробности: секция E.10.

### 5. Disengage/Dash/Dodge → Universal actions в AI, не в данных

**Решение:** Dodge/Dash/Disengage захардкожены в AI как правила игры. Не записываются в `StructuredActions` существ.

- Dodge: Phase 1 (NPC при низком HP без хороших атак)
- Dash/Disengage: Phase 3 (нужен movement)

**Обоснование:** Это правила D&D, а не свойства существа. Каждое существо умеет это делать. Записывать в данные = дублирование.

Подробности: секция E.9.

### 6. Turn advancement → Manual: фронтенд. Auto-play: бэкенд.

**Решение:** Прямое следствие решения #2.

- Manual mode: фронтенд авторитетен для `currentRound`/`currentTurnIndex`. AI endpoint stateless.
- Auto-play mode: бэкенд управляет turn order. NPC ходят автоматически, PC получает `your_turn` через WebSocket.

**Обоснование:** Auto-play без turn management на бэкенде невозможен.

Подробности: секция E.11.

---

## N. Future: RL Transition Path

Когда накопится достаточно данных из audit log (тысячи боёв), можно рассмотреть RL:

1. **State space**: `TurnInput` уже содержит полное состояние — это observation для RL agent
2. **Action space**: `TurnDecision` — это action для RL agent
3. **Reward**: win/lose + damage dealt - damage taken + bonus за тактику
4. **Training data**: audit log содержит историю ходов, урона, state changes
5. **Interface**: `CombatAI` не меняется — RL-модель реализует тот же интерфейс
6. **Deployment**: RL inference через gRPC-сервис (Python + PyTorch), вызываемый из Go через тот же интерфейс

Минимальный датасет для обучения: **~10,000 боёв** (оценка). Rule-based AI может генерировать обучающие данные через self-play.
