// Inspect actual name structure for missing SRD creatures

// Check Swarm candidates (have matching docs but name.eng is empty)
print("=== Swarm of Beetles candidates (raw name field) ===");
db.creatures.find(
  { "name": { $regex: "Swarm", $options: "i" } }
).limit(5).forEach(c => {
  print(JSON.stringify({ name: c.name, url: c.url }));
});

print("\n=== Axe Beak candidates ===");
db.creatures.find(
  { $or: [
    { "name": { $regex: "Axe", $options: "i" } },
    { "name.rus": { $regex: "секач", $options: "i" } },
    { "url": { $regex: "axe", $options: "i" } }
  ] }
).limit(5).forEach(c => {
  print(JSON.stringify({ name: c.name, url: c.url }));
});

print("\n=== Deep Gnome candidates ===");
db.creatures.find(
  { $or: [
    { "name": { $regex: "Deep", $options: "i" } },
    { "name": { $regex: "Gnome", $options: "i" } },
    { "url": { $regex: "gnome", $options: "i" } }
  ] }
).limit(5).forEach(c => {
  print(JSON.stringify({ name: c.name, url: c.url }));
});

print("\n=== Will-o-Wisp / Cloaker / Succubus search ===");
db.creatures.find(
  { $or: [
    { "url": { $regex: "will", $options: "i" } },
    { "url": { $regex: "wisp", $options: "i" } },
    { "url": { $regex: "cloaker", $options: "i" } },
    { "url": { $regex: "succubus", $options: "i" } },
    { "url": { $regex: "incubus", $options: "i" } },
  ] }
).limit(10).forEach(c => {
  print(JSON.stringify({ name: c.name, url: c.url }));
});

print("\n=== Giant Rat (Diseased) candidates ===");
db.creatures.find(
  { $or: [
    { "url": { $regex: "rat", $options: "i" } },
    { "name": { $regex: "rat", $options: "i" } }
  ] }
).limit(10).forEach(c => {
  print(JSON.stringify({ name: c.name, url: c.url }));
});
