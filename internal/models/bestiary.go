package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type BestiaryReq struct {
	Start  int          `json:"start"`
	Size   int          `json:"size"`
	Search SearchParams `json:"search"`
	Order  []Order      `json:"order"`
	Filter FilterParams `json:"filter"`
}

type FilterParams struct {
	Book                []string `json:"book"`
	Npc                 []string `json:"npc"`
	ChallengeRating     []string `json:"challengeRating"`
	Type                []string `json:"type"`
	Size                []string `json:"size"`
	Tag                 []string `json:"tag"`
	Moving              []string `json:"moving"`
	Senses              []string `json:"senses"`
	VulnerabilityDamage []string `json:"vulnerabilityDamage"`
	ResistanceDamage    []string `json:"resistanceDamage"`
	ImmunityDamage      []string `json:"immunityDamage"`
	ImmunityCondition   []string `json:"immunityCondition"`
	Features            []string `json:"features"`
	Environment         []string `json:"environment"`
}

type BestiaryCreature struct {
	ID              primitive.ObjectID `bson:"_id" json:"id"`
	Name            Name               `bson:"name" json:"name"`
	Type            TypeName           `bson:"type" json:"type"`
	ChallengeRating string             `bson:"challengeRating" json:"challengeRating"`
	URL             string             `bson:"url" json:"url"`
	Source          Source             `bson:"source" json:"source"`
	Images          []string           `bson:"images" json:"images"`
}

type TypeName struct {
	Name string `bson:"name" json:"name"`
}
