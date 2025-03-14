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

type Speed struct {
	Value int `json:"value" bson:"value"`
}

type Ability struct {
	Str int `json:"str" bson:"str"`
	Dex int `json:"dex" bson:"dex"`
	Con int `json:"con" bson:"con"`
	Int int `json:"int" bson:"int"`
	Wiz int `json:"wiz" bson:"wiz"`
	Cha int `json:"cha" bson:"cha"`
}

type Skill struct {
	Name  string `json:"name" bson:"name"`
	Value int    `json:"value" bson:"value"`
}

type Senses struct {
	PassivePerception string `json:"passivePerception" bson:"passivePerception"`
}

type Action struct {
	Name  string `json:"name" bson:"name"`
	Value string `json:"value" bson:"value"`
}

type Reaction struct {
	Name  string `json:"name" bson:"name"`
	Value string `json:"value" bson:"value"`
}

type Tag struct {
	Name        string `json:"name" bson:"name"`
	Description string `json:"description" bson:"description"`
}

type Creature struct {
	ID               primitive.ObjectID `bson:"_id,omitempty" json:"_id"`
	Name             Name               `json:"name" bson:"name"`
	Size             Size               `json:"size" bson:"size"`
	Type             Type               `json:"type" bson:"type"`
	ChallengeRating  string             `json:"challengeRating" bson:"challengeRating"`
	URL              string             `json:"url" bson:"url"`
	Source           Source             `json:"source" bson:"source"`
	IDNum            int                `json:"id" bson:"id"`
	ProficiencyBonus string             `json:"proficiencyBonus" bson:"proficiencyBonus"`
	Alignment        string             `json:"alignment" bson:"alignment"`
	ArmorClass       int                `json:"armorClass" bson:"armorClass"`
	Hits             Hits               `json:"hits" bson:"hits"`
	Speed            []Speed            `json:"speed" bson:"speed"`
	Ability          Ability            `json:"ability" bson:"ability"`
	Skills           []Skill            `json:"skills" bson:"skills"`
	Senses           Senses             `json:"senses" bson:"senses"`
	Languages        []string           `json:"languages" bson:"languages"`
	Actions          []Action           `json:"actions" bson:"actions"`
	Reactions        []Reaction         `json:"reactions" bson:"reactions"`
	Description      string             `json:"description" bson:"description"`
	Tags             []Tag              `json:"tags" bson:"tags"`
	Images           []string           `json:"images" bson:"images"`
}
