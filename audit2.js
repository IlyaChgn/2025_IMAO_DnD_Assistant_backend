const fs = require('fs');
const path = require('path');

const SEED_DIR = path.join(__dirname, 'internal', 'pkg', 'spells', 'seed');

const FILES = [
  'srd_spells_level0.json',
  'srd_spells_level1.json',
  'srd_spells_level2.json',
  'srd_spells_level3.json',
  'srd_spells_level4.json',
  'srd_spells_level5.json',
  'srd_spells_level6plus.json',
];

// Load all spells
const allSpells = [];
for (const file of FILES) {
  const spells = JSON.parse(fs.readFileSync(path.join(SEED_DIR, file), 'utf8'));
  for (const s of spells) {
    s._file = file;
    allSpells.push(s);
  }
}

// Helper to find spell
function find(engName) {
  return allSpells.find(s => s.engName === engName);
}

// --- Spot-checks ---

console.log('=== SPOT-CHECK: Key Spells ===\n');

// Fireball: should be level 3, 8d6 fire, evocation, classes: sorcerer/wizard
const fireball = find('fireball');
if (fireball) {
  console.log('FIREBALL:');
  console.log(`  Level: ${fireball.level} (expect 3)`);
  console.log(`  School: ${fireball.school} (expect evocation)`);
  console.log(`  Classes: ${JSON.stringify(fireball.classes)} (expect sorcerer, wizard)`);
  const dmg = fireball.effects?.[0]?.damage?.base;
  if (dmg) {
    console.log(`  Damage: ${dmg.diceCount}${dmg.diceType} ${dmg.damageType} (expect 8d6 fire)`);
  } else {
    console.log(`  Damage: NOT FOUND in effects[0].damage.base`);
  }
  console.log(`  Range: ${JSON.stringify(fireball.range)}`);
  console.log(`  Enabled: ${fireball.enabled}`);
} else {
  console.log('FIREBALL: NOT FOUND');
}

console.log('');

// Cure Wounds: level 1, 1d8 healing, evocation, classes: bard/cleric/druid/paladin/ranger
const cureWounds = find('cure-wounds');
if (cureWounds) {
  console.log('CURE WOUNDS:');
  console.log(`  Level: ${cureWounds.level} (expect 1)`);
  console.log(`  School: ${cureWounds.school} (expect evocation)`);
  console.log(`  Classes: ${JSON.stringify(cureWounds.classes)}`);
  const healing = cureWounds.effects?.[0]?.healing?.base;
  if (healing) {
    console.log(`  Healing: ${healing.diceCount}${healing.diceType} (expect 1d8)`);
  } else {
    console.log(`  Healing: NOT FOUND in effects[0].healing.base`);
  }
  console.log(`  Enabled: ${cureWounds.enabled}`);
} else {
  console.log('CURE WOUNDS: NOT FOUND');
}

console.log('');

// Shield: level 1, reaction, abjuration
const shield = find('shield');
if (shield) {
  console.log('SHIELD:');
  console.log(`  Level: ${shield.level} (expect 1)`);
  console.log(`  School: ${shield.school} (expect abjuration)`);
  console.log(`  Casting time: ${JSON.stringify(shield.castingTime)} (expect reaction)`);
  console.log(`  Enabled: ${shield.enabled}`);
} else {
  console.log('SHIELD: NOT FOUND');
}

console.log('');

// Magic Missile: level 1, 3 x 1d4+1 force, evocation
const magicMissile = find('magic-missile');
if (magicMissile) {
  console.log('MAGIC MISSILE:');
  console.log(`  Level: ${magicMissile.level} (expect 1)`);
  console.log(`  School: ${magicMissile.school} (expect evocation)`);
  const dmg = magicMissile.effects?.[0]?.damage?.base;
  if (dmg) {
    console.log(`  Damage: ${dmg.diceCount}${dmg.diceType}+${dmg.bonus || 0} ${dmg.damageType} (expect 1d4+1 force, x3)`);
  }
  console.log(`  Enabled: ${magicMissile.enabled}`);
} else {
  console.log('MAGIC MISSILE: NOT FOUND');
}

console.log('');

// Counterspell: level 3, abjuration, reaction
const counterspell = find('counterspell');
if (counterspell) {
  console.log('COUNTERSPELL:');
  console.log(`  Level: ${counterspell.level} (expect 3)`);
  console.log(`  School: ${counterspell.school} (expect abjuration)`);
  console.log(`  Casting time: ${JSON.stringify(counterspell.castingTime)} (expect reaction)`);
  console.log(`  Classes: ${JSON.stringify(counterspell.classes)}`);
  console.log(`  Enabled: ${counterspell.enabled}`);
} else {
  console.log('COUNTERSPELL: NOT FOUND');
}

console.log('');

// Wish: level 9, conjuration (it's actually conjuration per some sources, but most say transmutation — let's just report)
const wish = find('wish');
if (wish) {
  console.log('WISH:');
  console.log(`  Level: ${wish.level} (expect 9)`);
  console.log(`  School: ${wish.school} (expect conjuration)`);
  console.log(`  Classes: ${JSON.stringify(wish.classes)}`);
  console.log(`  Enabled: ${wish.enabled}`);
} else {
  console.log('WISH: NOT FOUND');
}

console.log('');

// Eldritch Blast: cantrip, evocation, warlock
const eldritchBlast = find('eldritch-blast');
if (eldritchBlast) {
  console.log('ELDRITCH BLAST:');
  console.log(`  Level: ${eldritchBlast.level} (expect 0)`);
  console.log(`  School: ${eldritchBlast.school} (expect evocation)`);
  console.log(`  Classes: ${JSON.stringify(eldritchBlast.classes)} (expect warlock)`);
  const dmg = eldritchBlast.effects?.[0]?.damage?.base;
  if (dmg) {
    console.log(`  Damage: ${dmg.diceCount}${dmg.diceType} ${dmg.damageType} (expect 1d10 force)`);
  }
  console.log(`  Enabled: ${eldritchBlast.enabled}`);
} else {
  console.log('ELDRITCH BLAST: NOT FOUND');
}

console.log('');

// Healing Word: level 1, bonus action, 1d4 healing
const healingWord = find('healing-word');
if (healingWord) {
  console.log('HEALING WORD:');
  console.log(`  Level: ${healingWord.level} (expect 1)`);
  console.log(`  School: ${healingWord.school} (expect evocation)`);
  console.log(`  Casting time: ${JSON.stringify(healingWord.castingTime)} (expect bonus_action)`);
  const healing = healingWord.effects?.[0]?.healing?.base;
  if (healing) {
    console.log(`  Healing: ${healing.diceCount}${healing.diceType} (expect 1d4)`);
  }
  console.log(`  Range: ${JSON.stringify(healingWord.range)} (expect 60ft)`);
  console.log(`  Enabled: ${healingWord.enabled}`);
} else {
  console.log('HEALING WORD: NOT FOUND');
}

console.log('');

// --- List all disabled spells ---
console.log('=== DISABLED SPELLS (enabled: false) ===\n');
const disabled = allSpells.filter(s => s.enabled === false);
for (const s of disabled) {
  console.log(`  [${s._file}] ${s.engName} (level ${s.level}, ${s.school})`);
}

console.log(`\nTotal disabled: ${disabled.length}`);

// --- List all schools used ---
console.log('\n=== SCHOOLS USED ===');
const schools = new Set(allSpells.map(s => s.school));
console.log([...schools].sort().join(', '));

// --- List all classes used ---
console.log('\n=== CLASSES USED ===');
const classes = new Set(allSpells.flatMap(s => s.classes || []));
console.log([...classes].sort().join(', '));

// --- Check for empty descriptions ---
console.log('\n=== EMPTY DESCRIPTIONS CHECK ===');
const emptyDescs = allSpells.filter(s => {
  if (!s.description) return true;
  if (typeof s.description === 'object') {
    return !s.description.eng || !s.description.rus;
  }
  return !s.description;
});
if (emptyDescs.length > 0) {
  for (const s of emptyDescs) {
    console.log(`  EMPTY: ${s.engName} in ${s._file}`);
  }
} else {
  console.log('  All spells have non-empty eng+rus descriptions.');
}

// --- Check for empty classes arrays ---
console.log('\n=== EMPTY CLASSES CHECK ===');
const emptyClasses = allSpells.filter(s => !s.classes || s.classes.length === 0);
if (emptyClasses.length > 0) {
  for (const s of emptyClasses) {
    console.log(`  EMPTY CLASSES: ${s.engName} in ${s._file}`);
  }
} else {
  console.log('  All spells have at least one class.');
}

// --- Check components structure ---
console.log('\n=== COMPONENTS STRUCTURE CHECK ===');
const badComponents = allSpells.filter(s => {
  if (!s.components) return true;
  return typeof s.components.verbal !== 'boolean' || typeof s.components.somatic !== 'boolean' || typeof s.components.material !== 'boolean';
});
if (badComponents.length > 0) {
  for (const s of badComponents) {
    console.log(`  BAD COMPONENTS: ${s.engName} — ${JSON.stringify(s.components)}`);
  }
} else {
  console.log('  All spells have valid components (V/S/M booleans).');
}

// --- Level distribution in level6plus ---
console.log('\n=== LEVEL DISTRIBUTION IN level6plus.json ===');
const l6plus = allSpells.filter(s => s._file === 'srd_spells_level6plus.json');
const levelDist = {};
for (const s of l6plus) {
  levelDist[s.level] = (levelDist[s.level] || 0) + 1;
}
for (const [lvl, count] of Object.entries(levelDist).sort()) {
  console.log(`  Level ${lvl}: ${count} spells`);
}
