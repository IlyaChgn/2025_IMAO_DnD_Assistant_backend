package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Name struct {
	Rus string `json:"rus" bson:"rus"`
	Eng string `json:"eng" bson:"eng"`
}

type Size struct {
	Rus  string `json:"rus" bson:"rus"`
	Eng  string `json:"eng" bson:"eng"`
	Cell string `json:"cell" bson:"cell"`
}

type Type struct {
	Name string   `json:"name" bson:"name"`
	Tags []string `json:"tags" bson:"tags"`
}

type SourceGroup struct {
	Name      string `json:"name" bson:"name"`
	ShortName string `json:"shortName" bson:"shortName"`
}

type Source struct {
	ShortName string      `json:"shortName" bson:"shortName"`
	Name      string      `json:"name" bson:"name"`
	Group     SourceGroup `json:"group" bson:"group"`
}

type Hits struct {
	Average int    `json:"average" bson:"average"`
	Formula string `json:"formula" bson:"formula"`
}

// Deprecated: Speed uses interface{} for Value. Use CreatureMovement instead for automation.
type Speed struct {
	Value      interface{} `json:"value" bson:"value"`
	Name       string      `json:"name,omitempty" bson:"name,omitempty"`
	Additional string      `json:"additional,omitempty" bson:"additional,omitempty"`
}

// CreatureMovement provides structured movement speeds in feet for pathfinding automation.
type CreatureMovement struct {
	Walk   int  `json:"walk" bson:"walk"`
	Fly    int  `json:"fly,omitempty" bson:"fly,omitempty"`
	Swim   int  `json:"swim,omitempty" bson:"swim,omitempty"`
	Climb  int  `json:"climb,omitempty" bson:"climb,omitempty"`
	Burrow int  `json:"burrow,omitempty" bson:"burrow,omitempty"`
	Hover  bool `json:"hover,omitempty" bson:"hover,omitempty"` // fly without falling when speed=0
}

type Ability struct {
	Str int `json:"str" bson:"str"`
	Dex int `json:"dex" bson:"dex"`
	Con int `json:"con" bson:"con"`
	Int int `json:"int" bson:"int"`
	Wiz int `json:"wis" bson:"wiz"`
	Cha int `json:"cha" bson:"cha"`
}

type Skill struct {
	Name  string `json:"name" bson:"name"`
	Value int    `json:"value" bson:"value"`
}

// Deprecated: Senses uses string for PassivePerception and unstructured Sense slice.
// Use CreatureVision for automation. PassivePerception can be computed from skills.
type Senses struct {
	PassivePerception string  `json:"passivePerception" bson:"passivePerception"`
	Sense             []Sense `json:"senses,omitempty" bson:"senses,omitempty"`
}

type Sense struct {
	Name       string `json:"name" bson:"name"`
	Value      int    `json:"value" bson:"value"`
	Additional string `json:"additional,omitempty" bson:"additional,omitempty"`
}

// CreatureVision provides structured vision ranges in feet for fog-of-war/lighting automation.
type CreatureVision struct {
	Darkvision  int `json:"darkvision,omitempty" bson:"darkvision,omitempty"` // 0 = no darkvision
	Blindsight  int `json:"blindsight,omitempty" bson:"blindsight,omitempty"`
	Truesight   int `json:"truesight,omitempty" bson:"truesight,omitempty"`
	Tremorsense int `json:"tremorsense,omitempty" bson:"tremorsense,omitempty"`
}

// Deprecated: Action stores text only. Use StructuredAction for automation.
type Action struct {
	Name  string `json:"name" bson:"name"`
	Value string `json:"value" bson:"value"`
}

// Deprecated: BonusAction stores text only. Use StructuredAction with Category=bonus_action.
type BonusAction struct {
	Name  string `json:"name" bson:"name"`
	Value string `json:"value" bson:"value"`
}

// Deprecated: Reaction stores text only. Use StructuredAction with Category=reaction.
type Reaction struct {
	Name  string `json:"name" bson:"name"`
	Value string `json:"value" bson:"value"`
}

type Tag struct {
	Name        string `json:"name" bson:"name"`
	Description string `json:"description" bson:"description"`
}

type SavingThrow struct {
	Name      string      `json:"name" bson:"name"`
	ShortName string      `json:"shortName" bson:"shortName"`
	Value     interface{} `json:"value" bson:"value"`
}

type Feat struct {
	Name  string      `json:"name" bson:"name"`
	Value interface{} `json:"value" bson:"value"`
}

type Legendary struct {
	List  []LegendaryAction `json:"list" bson:"list"`
	Count interface{}       `json:"count" bson:"count"`
}

type LegendaryAction struct {
	Name  string      `json:"name" bson:"name"`
	Value interface{} `json:"value" bson:"value"`
}

type Armor struct {
	Name string      `json:"name" bson:"name"`
	Type string      `json:"type" bson:"type"`
	Url  interface{} `json:"url" bson:"url"`
}

type Creature struct {
	ID                    primitive.ObjectID  `bson:"_id,omitempty" json:"_id"`
	Name                  Name                `json:"name" bson:"name"`
	Size                  Size                `json:"size" bson:"size"`
	Type                  Type                `json:"type" bson:"type"`
	ChallengeRating       string              `json:"challengeRating" bson:"challengeRating"`
	URL                   string              `json:"url" bson:"url"`
	Source                Source              `json:"source" bson:"source"`
	IDNum                 int                 `json:"id" bson:"id"`
	Experience            int                 `json:"experience,omitempty" bson:"experience,omitempty"`
	ProficiencyBonus      string              `json:"proficiencyBonus" bson:"proficiencyBonus"`
	Alignment             string              `json:"alignment" bson:"alignment"`
	ArmorClass            int                 `json:"armorClass" bson:"armorClass"`
	ArmorText             string              `json:"armorText,omitempty" bson:"armorText,omitempty"`
	Armors                []Armor             `json:"armors,omitempty" bson:"armors,omitempty"`
	Hits                  Hits                `json:"hits" bson:"hits"`
	Speed                 []Speed             `json:"speed" bson:"speed"`
	Movement              CreatureMovement    `json:"movement,omitempty" bson:"movement,omitempty"` // structured speeds for automation
	Ability               Ability             `json:"ability" bson:"ability"`
	SavingThrows          []SavingThrow       `json:"savingThrows,omitempty" bson:"savingThrows,omitempty"`
	Skills                []Skill             `json:"skills" bson:"skills"`
	DamageVulnerabilities []string            `json:"damageVulnerabilities,omitempty" bson:"damageVulnerabilities,omitempty"`
	DamageResistances     []string            `json:"damageResistances,omitempty" bson:"damageResistances,omitempty"`
	ConditionImmunities   []string            `json:"conditionImmunities,omitempty" bson:"conditionImmunities,omitempty"`
	DamageImmunities      []string            `json:"damageImmunities,omitempty" bson:"damageImmunities,omitempty"`
	Senses                Senses              `json:"senses" bson:"senses"`
	Vision                CreatureVision      `json:"vision,omitempty" bson:"vision,omitempty"` // structured vision for fog/lighting
	Languages             []string            `json:"languages" bson:"languages"`
	Feats                 []Feat              `json:"feats,omitempty" bson:"feats,omitempty"`
	Actions               []Action            `json:"actions" bson:"actions"`
	BonusActions          []BonusAction       `json:"bonusActions,omitempty" bson:"bonusActions,omitempty"`
	Legendary             Legendary           `json:"legendary,omitempty" bson:"legendary,omitempty"`
	Reactions             []Reaction          `json:"reactions,omitempty" bson:"reactions,omitempty"`
	Description           string              `json:"description" bson:"description"`
	Tags                  []Tag               `json:"tags" bson:"tags"`
	Images                []string            `json:"images" bson:"images"`
	Environment           []string            `json:"environment,omitempty" bson:"environment,omitempty"`
	LLMParsedAttack       []AttackLLM         `bson:"llm_parsed_attack,omitempty" json:"attacksLLM,omitempty"`
	StructuredActions     []StructuredAction  `json:"structuredActions,omitempty" bson:"structuredActions,omitempty"`   // machine-readable actions for automation
	Spellcasting          *Spellcasting       `json:"spellcasting,omitempty" bson:"spellcasting,omitempty"`             // regular spellcasting (spell slots)
	InnateSpellcasting    *InnateSpellcasting `json:"innateSpellcasting,omitempty" bson:"innateSpellcasting,omitempty"` // at-will and X/day spells
	UserID                string              `bson:"userID,omitempty" json:"userID,omitempty"`
}

type CreatureInput struct {
	ID                string `json:"_id"`
	ImageBase64       string `json:"imageBase64"`
	ImageBase64Circle string `json:"imageBase64Circle"`
	Creature
}
