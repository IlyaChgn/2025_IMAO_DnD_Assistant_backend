// Fix the 10 missing SRD creatures using their actual names as stored in the DB.
// Run after mark_srd_creatures.js.

// 1. Known name mismatches — mark directly by actual stored name
const directFixes = [
  "Axe Baek",               // typo for "Axe Beak"
  "Gnome, Deep (Svirfneblin)", // stored differently from "Deep Gnome (Svirfneblin)"
  "Will-o-wisp",            // vs "Will-o'-Wisp"
  "Incubus",                // stored as Incubus only, SRD has "Succubus/Incubus"
];

print("=== Direct name fixes ===");
for (const name of directFixes) {
  const r = db.creatures.updateMany(
    { "name.eng": name },
    { $set: { isSrdSupported: true } },
    { collation: { locale: "en", strength: 2 } }
  );
  print(`  "${name}": matched=${r.matchedCount} modified=${r.modifiedCount}`);
}

// 2. Swarms — search by name.eng regex
print("\n=== Swarm searches ===");
const swarmKeywords = ["Beetles", "Centipedes", "Spiders", "Wasps"];
for (const kw of swarmKeywords) {
  const candidates = db.creatures.find(
    { "name.eng": { $regex: kw, $options: "i" } }
  ).toArray();
  if (candidates.length === 0) {
    print(`  "Swarm of ${kw}": NOT FOUND in DB`);
  } else {
    for (const c of candidates) {
      print(`  "Swarm of ${kw}" → found: "${c.name.eng}" — marking as SRD`);
      db.creatures.updateOne(
        { _id: c._id },
        { $set: { isSrdSupported: true } }
      );
    }
  }
}

// 3. Cloaker
print("\n=== Cloaker ===");
const cloakers = db.creatures.find({ "name.eng": { $regex: "cloaker", $options: "i" } }).toArray();
if (cloakers.length === 0) {
  print('  "Cloaker": NOT FOUND in DB');
} else {
  for (const c of cloakers) {
    print(`  Found: "${c.name.eng}" — marking as SRD`);
    db.creatures.updateOne({ _id: c._id }, { $set: { isSrdSupported: true } });
  }
}

// 4. Giant Rat (Diseased)
print("\n=== Giant Rat (Diseased) ===");
const diseased = db.creatures.find({
  "name.eng": { $regex: "giant rat", $options: "i" }
}).toArray();
for (const c of diseased) {
  print(`  Found: "${c.name.eng}"`);
}
if (diseased.length === 0) {
  print('  No "Giant Rat" variants found at all');
}

// Final count
print(`\n=== isSrdSupported total ===`);
print(`  Total marked: ${db.creatures.countDocuments({ isSrdSupported: true })}`);
