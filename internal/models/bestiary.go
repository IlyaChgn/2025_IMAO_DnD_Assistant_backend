package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type BestiaryReq struct {
	Size  int `json:"size"`
	Start int `json:"start"`
}

type BestiaryCreature struct {
	ID              primitive.ObjectID `bson:"_id" json:"id"`
	Name            Name               `bson:"name" json:"name"`
	Type            TypeName           `bson:"type" json:"type"`
	ChallengeRating string             `bson:"challengeRating" json:"challengeRating"`
	URL             string             `bson:"url" json:"url"`
	Source          Source             `bson:"source" json:"source"`
}

type TypeName struct {
	Name string `bson:"name" json:"name"`
}
