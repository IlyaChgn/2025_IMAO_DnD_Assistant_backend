package compute

// skillAbilities maps each D&D skill (lowercase, underscored) to its governing ability.
var skillAbilities = map[string]string{
	"acrobatics":      "dex",
	"animal_handling": "wis",
	"arcana":          "int",
	"athletics":       "str",
	"deception":       "cha",
	"history":         "int",
	"insight":         "wis",
	"intimidation":    "cha",
	"investigation":   "int",
	"medicine":        "wis",
	"nature":          "int",
	"perception":      "wis",
	"performance":     "cha",
	"persuasion":      "cha",
	"religion":        "int",
	"sleight_of_hand": "dex",
	"stealth":         "dex",
	"survival":        "wis",
}
