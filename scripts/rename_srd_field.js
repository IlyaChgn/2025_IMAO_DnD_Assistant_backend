// Rename isSrdSupported -> isSrd and set false for non-SRD creatures.

// 1. Rename field for all docs that have it
const renameResult = db.creatures.updateMany(
  { isSrdSupported: { $exists: true } },
  { $rename: { "isSrdSupported": "isSrd" } }
);
print(`Renamed isSrdSupported -> isSrd: ${renameResult.modifiedCount} docs`);

// 2. Set isSrd: false for all creatures that don't have isSrd yet
const falseResult = db.creatures.updateMany(
  { isSrd: { $exists: false } },
  { $set: { isSrd: false } }
);
print(`Set isSrd: false for ${falseResult.modifiedCount} remaining creatures`);

// 3. Summary
const totalSrd = db.creatures.countDocuments({ isSrd: true });
const totalNonSrd = db.creatures.countDocuments({ isSrd: false });
print(`\nTotal: ${totalSrd} SRD + ${totalNonSrd} non-SRD = ${totalSrd + totalNonSrd} creatures`);
