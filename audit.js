const fs = require('fs');
const path = require('path');

const SEED_DIR = path.join(__dirname, 'internal', 'pkg', 'spells', 'seed');

const FILES = [
  { file: 'srd_spells_level0.json', expectedLevels: [0] },
  { file: 'srd_spells_level1.json', expectedLevels: [1] },
  { file: 'srd_spells_level2.json', expectedLevels: [2] },
  { file: 'srd_spells_level3.json', expectedLevels: [3] },
  { file: 'srd_spells_level4.json', expectedLevels: [4] },
  { file: 'srd_spells_level5.json', expectedLevels: [5] },
  { file: 'srd_spells_level6plus.json', expectedLevels: [6, 7, 8, 9] },
];

const REQUIRED_FIELDS = ['engName', 'name', 'level', 'school', 'enabled', 'description', 'components', 'classes'];

const allEngNames = new Map(); // engName -> file
const results = [];

for (const { file, expectedLevels } of FILES) {
  const filePath = path.join(SEED_DIR, file);
  const result = { file, issues: [] };

  // 1. Parse JSON
  let spells;
  try {
    const raw = fs.readFileSync(filePath, 'utf8');
    spells = JSON.parse(raw);
    if (!Array.isArray(spells)) {
      result.issues.push('NOT an array at top level');
      results.push(result);
      continue;
    }
  } catch (e) {
    result.issues.push(`INVALID JSON: ${e.message}`);
    results.push(result);
    continue;
  }

  // 2. Count
  result.totalSpells = spells.length;

  // 3. Enabled counts
  const enabledTrue = spells.filter(s => s.enabled === true).length;
  const enabledFalse = spells.filter(s => s.enabled === false).length;
  const enabledOther = spells.length - enabledTrue - enabledFalse;
  result.enabled = { true: enabledTrue, false: enabledFalse };
  if (enabledOther > 0) {
    result.issues.push(`${enabledOther} spells have non-boolean 'enabled' or missing 'enabled'`);
  }

  // 4. Level bucket check
  const levelMismatches = [];
  for (const spell of spells) {
    if (!expectedLevels.includes(spell.level)) {
      levelMismatches.push({ engName: spell.engName, actualLevel: spell.level });
    }
  }
  if (levelMismatches.length > 0) {
    result.issues.push(`Level mismatches: ${JSON.stringify(levelMismatches)}`);
  }

  // 5. Required fields
  const missingFields = [];
  for (const spell of spells) {
    const missing = REQUIRED_FIELDS.filter(f => spell[f] === undefined || spell[f] === null);
    if (missing.length > 0) {
      missingFields.push({ engName: spell.engName || '(no engName)', missing });
    }
  }
  if (missingFields.length > 0) {
    result.issues.push(`Missing required fields: ${JSON.stringify(missingFields)}`);
  }

  // 6. Duplicate check (cross-file)
  for (const spell of spells) {
    if (spell.engName) {
      if (allEngNames.has(spell.engName)) {
        result.issues.push(`DUPLICATE engName "${spell.engName}" (also in ${allEngNames.get(spell.engName)})`);
      } else {
        allEngNames.set(spell.engName, file);
      }
    }
  }

  results.push(result);
}

// Print report
console.log('=== SPELL SEED FILES AUDIT ===\n');

let grandTotal = 0;
let grandEnabled = 0;
let grandDisabled = 0;

for (const r of results) {
  console.log(`--- ${r.file} ---`);
  if (r.totalSpells !== undefined) {
    console.log(`  Total spells: ${r.totalSpells}`);
    grandTotal += r.totalSpells;
  }
  if (r.enabled) {
    console.log(`  Enabled: ${r.enabled.true}  |  Disabled: ${r.enabled.false}`);
    grandEnabled += r.enabled.true;
    grandDisabled += r.enabled.false;
  }
  if (r.issues.length === 0) {
    console.log(`  Issues: NONE`);
  } else {
    for (const issue of r.issues) {
      console.log(`  ISSUE: ${issue}`);
    }
  }
  console.log('');
}

console.log('=== GRAND TOTALS ===');
console.log(`Total spells across all files: ${grandTotal}`);
console.log(`Enabled: ${grandEnabled}  |  Disabled: ${grandDisabled}`);
console.log(`Unique engName count: ${allEngNames.size}`);
if (allEngNames.size !== grandTotal) {
  console.log(`WARNING: ${grandTotal - allEngNames.size} duplicate or missing engNames detected`);
}
