# Homebrew Material Components

36 leveled action-cast spells that were originally V/S-only in 5e SRD received homebrew material components. These spells were V/S-only even in older editions (AD&D 1e/2e, 3.5e), so components are designed thematically by school and effect.

Cantrips remain free. Bonus action and reaction spells (Shield, Counterspell, Healing Word, Misty Step, etc.) are excluded — they stay V/S-only.

## Spells with Homebrew Components

### Level 1

| Spell | School | Component | Reagent engName | Reused? |
|-------|--------|-----------|-----------------|---------|
| Burning Hands | evocation | a pinch of sulfur | `sulfur` | yes |
| Command | enchantment | a brass whistle | `brass-whistle` | NEW |
| Cure Wounds | evocation | a sprig of dried yarrow | `dried-yarrow` | NEW |
| Entangle | conjuration | a twist of vine | `vine-twist` | NEW |
| Faerie Fire | evocation | a pinch of phosphorus | `phosphorus` | yes |
| Fog Cloud | conjuration | a bit of sponge soaked in water | `wet-sponge` | NEW |
| Guiding Bolt | evocation | a shard of quartz | `quartz-shard` | NEW |
| Heroism | enchantment | a lion's whisker | `lion-whisker` | NEW |
| Inflict Wounds | necromancy | a sliver of bone | `bone-sliver` | NEW |
| Magic Missile | evocation | a chip of quartz | `quartz-shard` | NEW (shared) |
| Thunderwave | evocation | a small brass bell | `brass-bell` | NEW |

### Level 2

| Spell | School | Component | Reagent engName | Reused? |
|-------|--------|-----------|-----------------|---------|
| Blindness/Deafness | necromancy | a pinch of soot | `soot` | NEW |
| Blur | illusion | a smear of translucent grease | `translucent-grease` | NEW |
| Lesser Restoration | abjuration | a pinch of powdered silver | `powdered-silver` | yes |
| Mirror Image | illusion | a sliver of mirror glass | `mirror-sliver` | NEW |
| Ray of Enfeeblement | necromancy | a withered twig | `withered-twig` | NEW |
| Scorching Ray | evocation | a piece of flint | `flint` | NEW |
| Silence | illusion | a drop of thick wax | `wax-drop` | NEW |

### Level 3

| Spell | School | Component | Reagent engName(s) | Reused? |
|-------|--------|-----------|-------------------|---------|
| Beacon of Hope | abjuration | a chip of sunstone | `sunstone-chip` | NEW |
| Bestow Curse | necromancy | a dried raven's claw | `raven-claw` | NEW |
| Blink | transmutation | a pinch of silver dust | `powdered-silver` | yes |
| Call Lightning | conjuration | a bit of fur and an iron nail | `fur` + `iron-nail` | fur=yes, nail=NEW |
| Dispel Magic | abjuration | a pinch of powdered iron | `powdered-iron` | yes |
| Protection from Energy | abjuration | a strip of elemental-touched cloth | `elemental-cloth` | NEW |
| Remove Curse | abjuration | a sprig of dried sage | `dried-sage` | NEW |
| Vampiric Touch | necromancy | a drop of blood | `blood-drop` | yes |

### Level 4

| Spell | School | Component | Reagent engName(s) | Reused? |
|-------|--------|-----------|-------------------|---------|
| Blight | necromancy | a pinch of ash from a dead tree | `dead-ash` | NEW |
| Death Ward | abjuration | a sliver of bone wrapped in white cloth | `bone-sliver` + `cloth-strip` | both=yes |
| Dimension Door | conjuration | a copper key | `copper-key` | NEW |
| Greater Invisibility | illusion | a pinch of powdered glass | `powdered-glass` | NEW |
| Guardian of Faith | conjuration | a holy symbol | `holy-symbol` | yes |

### Level 5

| Spell | School | Component | Reagent engName(s) | Reused? |
|-------|--------|-----------|-------------------|---------|
| Antilife Shell | abjuration | a ring of iron filings | `powdered-iron` | yes |
| Cloudkill | conjuration | a dried toadstool | `dried-toadstool` | NEW |
| Contagion | necromancy | a pinch of rot grub dust | `rot-grub-dust` | NEW |
| Mass Cure Wounds | evocation | dried yarrow and a drop of holy water | `dried-yarrow` + `holy-water` | both=yes |
| Telekinesis | transmutation | a copper wire bent into a spiral | `copper-wire` | NEW |

## New Reagent Items (25)

These were added to `internal/pkg/items/seed/srd_reagents.json`:

| engName | Name (EN) | Name (RU) | Subcategory |
|---------|-----------|-----------|-------------|
| `bone-sliver` | Bone Sliver | Костяная щепка | animal |
| `brass-bell` | Small Brass Bell | Маленький латунный колокольчик | mundane |
| `brass-whistle` | Brass Whistle | Латунный свисток | mundane |
| `copper-key` | Copper Key | Медный ключ | mundane |
| `copper-wire` | Copper Wire Spiral | Спираль из медной проволоки | mundane |
| `dead-ash` | Ash of Dead Wood | Пепел мёртвого дерева | plant |
| `dried-sage` | Dried Sage | Сушёный шалфей | plant |
| `dried-toadstool` | Dried Toadstool | Сушёная поганка | plant |
| `dried-yarrow` | Dried Yarrow | Сушёный тысячелистник | plant |
| `elemental-cloth` | Elemental-Touched Cloth | Стихийная ткань | arcane |
| `flint` | Piece of Flint | Кусочек кремня | mineral |
| `iron-nail` | Iron Nail | Железный гвоздь | mineral |
| `lion-whisker` | Lion's Whisker | Ус льва | animal |
| `mirror-sliver` | Mirror Sliver | Осколок зеркала | arcane |
| `powdered-glass` | Powdered Glass | Стеклянный порошок | arcane |
| `quartz-shard` | Quartz Shard | Осколок кварца | mineral |
| `raven-claw` | Dried Raven's Claw | Сушёный коготь ворона | animal |
| `rot-grub-dust` | Rot Grub Dust | Порошок гнильца | animal |
| `soot` | Pinch of Soot | Щепотка сажи | mundane |
| `sunstone-chip` | Sunstone Chip | Осколок солнечного камня | mineral |
| `translucent-grease` | Translucent Grease | Прозрачная мазь | liquid |
| `vine-twist` | Vine Twist | Витая лоза | plant |
| `wax-drop` | Drop of Wax | Капля воска | mundane |
| `wet-sponge` | Wet Sponge | Мокрая губка | mundane |
| `withered-twig` | Withered Twig | Засохшая веточка | plant |

## Coverage Summary

| Category | Count |
|----------|-------|
| Enabled leveled spells | 102 |
| With material components | 90 (88%) |
| — reagentFormula | 86 |
| — gemCost | 4 |
| Without M (bonus action / reaction) | 12 |
| Cantrips (free) | 14 |
| Total unique reagents | 114 |
