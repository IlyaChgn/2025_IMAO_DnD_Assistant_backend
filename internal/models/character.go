package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type CharacterFilterParams struct{}

type CharacterName struct {
	Value string `json:"value" bson:"value"`
}

type CharClass struct {
	Name  string `json:"name" bson:"name"`
	Label string `json:"label" bson:"label"`
	Value string `json:"value" bson:"value"`
}

type CharacterLevel struct {
	Name  string `json:"name" bson:"name"`
	Label string `json:"label" bson:"label"`
	Value int    `json:"value" bson:"value"`
}

type CharacterRace struct {
	Name  string `json:"name" bson:"name"`
	Label string `json:"label" bson:"label"`
	Value string `json:"value" bson:"value"`
}

type CharacterAvatar struct {
	Jpeg string `json:"jpeg" bson:"jpeg"`
	Webp string `json:"webp" bson:"webp"`
}

type CharacterData struct {
	IsDefault bool          `json:"isDefault" bson:"isDefault"`
	JsonType  string        `json:"jsonType" bson:"jsonType"`
	Template  string        `json:"template" bson:"template"`
	Name      CharacterName `json:"name" bson:"name"`
	Info      struct {
		CharClass  CharClass      `json:"charClass" bson:"charClass"`
		Level      CharacterLevel `json:"level" bson:"level"`
		Background struct {
			Name  string `json:"name" bson:"name"`
			Label string `json:"label" bson:"label"`
			Value string `json:"value" bson:"value"`
		} `json:"background" bson:"background"`
		PlayerName struct {
			Name  string `json:"name" bson:"name"`
			Label string `json:"label" bson:"label"`
			Value string `json:"value" bson:"value"`
		} `json:"playerName" bson:"playerName"`
		Race      CharacterRace `json:"race" bson:"race"`
		Alignment struct {
			Name  string `json:"name" bson:"name"`
			Label string `json:"label" bson:"label"`
			Value string `json:"value" bson:"value"`
		} `json:"alignment" bson:"alignment"`
		Experience struct {
			Name  string      `json:"name" bson:"name"`
			Label string      `json:"label" bson:"label"`
			Value interface{} `json:"value" bson:"value"`
		} `json:"experience" bson:"experience"`
	} `json:"info" bson:"info"`
	SubInfo struct {
		Age struct {
			Name  string      `json:"name" bson:"name"`
			Label string      `json:"label" bson:"label"`
			Value interface{} `json:"value" bson:"value"`
		} `json:"age" bson:"age"`
		Height struct {
			Name  string      `json:"name" bson:"name"`
			Label string      `json:"label" bson:"label"`
			Value interface{} `json:"value" bson:"value"`
		} `json:"height" bson:"height"`
		Weight struct {
			Name  string      `json:"name" bson:"name"`
			Label string      `json:"label" bson:"label"`
			Value interface{} `json:"value" bson:"value"`
		} `json:"weight" bson:"weight"`
		Eyes struct {
			Name  string      `json:"name" bson:"name"`
			Label string      `json:"label" bson:"label"`
			Value interface{} `json:"value" bson:"value"`
		} `json:"eyes" bson:"eyes"`
		Skin struct {
			Name  string      `json:"name" bson:"name"`
			Label string      `json:"label" bson:"label"`
			Value interface{} `json:"value" bson:"value"`
		} `json:"skin" bson:"skin"`
		Hair struct {
			Name  string      `json:"name" bson:"name"`
			Label string      `json:"label" bson:"label"`
			Value interface{} `json:"value" bson:"value"`
		} `json:"hair" bson:"hair"`
	} `json:"subInfo" bson:"subInfo"`
	SpellsInfo struct {
		Base struct {
			Name  string `json:"name" bson:"name"`
			Label string `json:"label" bson:"label"`
			Value string `json:"value" bson:"value"`
		} `json:"base" bson:"base"`
		Save struct {
			Name  string `json:"name" bson:"name"`
			Label string `json:"label" bson:"label"`
			Value string `json:"value" bson:"value"`
		} `json:"save" bson:"save"`
		Mod struct {
			Name  string `json:"name" bson:"name"`
			Label string `json:"label" bson:"label"`
			Value string `json:"value" bson:"value"`
		} `json:"mod" bson:"mod"`
	} `json:"spellsInfo" bson:"spellsInfo"`
	Spells struct {
		Slots1 struct {
			Value interface{} `json:"value" bson:"value"`
		} `json:"slots-1" bson:"slots-1"`
		Slots2 struct {
			Value interface{} `json:"value" bson:"value"`
		} `json:"slots-2" bson:"slots-2"`
		Slots3 struct {
			Value interface{} `json:"value" bson:"value"`
		} `json:"slots-3" bson:"slots-3"`
		Slots4 struct {
			Value interface{} `json:"value" bson:"value"`
		} `json:"slots-4" bson:"slots-4"`
		Slots5 struct {
			Value interface{} `json:"value" bson:"value"`
		} `json:"slots-5" bson:"slots-5"`
	} `json:"spells" bson:"spells"`
	SpellsPact  map[string]interface{} `json:"spellsPact" bson:"spellsPact"`
	Proficiency int                    `json:"proficiency" bson:"proficiency"`
	Stats       struct {
		Str struct {
			Name     string `json:"name" bson:"name"`
			Label    string `json:"label" bson:"label"`
			Score    int    `json:"score" bson:"score"`
			Modifier int    `json:"modifier" bson:"modifier"`
		} `json:"str" bson:"str"`
		Dex struct {
			Name     string `json:"name" bson:"name"`
			Label    string `json:"label" bson:"label"`
			Score    int    `json:"score" bson:"score"`
			Modifier int    `json:"modifier" bson:"modifier"`
		} `json:"dex" bson:"dex"`
		Con struct {
			Name     string `json:"name" bson:"name"`
			Label    string `json:"label" bson:"label"`
			Score    int    `json:"score" bson:"score"`
			Modifier int    `json:"modifier" bson:"modifier"`
		} `json:"con" bson:"con"`
		Int struct {
			Name     string `json:"name" bson:"name"`
			Label    string `json:"label" bson:"label"`
			Score    int    `json:"score" bson:"score"`
			Modifier int    `json:"modifier" bson:"modifier"`
		} `json:"int" bson:"int"`
		Wis struct {
			Name     string `json:"name" bson:"name"`
			Label    string `json:"label" bson:"label"`
			Score    int    `json:"score" bson:"score"`
			Modifier int    `json:"modifier" bson:"modifier"`
		} `json:"wis" bson:"wis"`
		Cha struct {
			Name     string `json:"name" bson:"name"`
			Label    string `json:"label" bson:"label"`
			Score    int    `json:"score" bson:"score"`
			Modifier int    `json:"modifier" bson:"modifier"`
		} `json:"cha" bson:"cha"`
	} `json:"stats" bson:"stats"`
	Saves struct {
		Str struct {
			Name   string `json:"name" bson:"name"`
			IsProf bool   `json:"isProf" bson:"isProf"`
		} `json:"str" bson:"str"`
		Dex struct {
			Name   string `json:"name" bson:"name"`
			IsProf bool   `json:"isProf" bson:"isProf"`
		} `json:"dex" bson:"dex"`
		Con struct {
			Name   string `json:"name" bson:"name"`
			IsProf bool   `json:"isProf" bson:"isProf"`
		} `json:"con" bson:"con"`
		Int struct {
			Name   string `json:"name" bson:"name"`
			IsProf bool   `json:"isProf" bson:"isProf"`
		} `json:"int" bson:"int"`
		Wis struct {
			Name   string `json:"name" bson:"name"`
			IsProf bool   `json:"isProf" bson:"isProf"`
		} `json:"wis" bson:"wis"`
		Cha struct {
			Name   string `json:"name" bson:"name"`
			IsProf bool   `json:"isProf" bson:"isProf"`
		} `json:"cha" bson:"cha"`
	} `json:"saves" bson:"saves"`
	Skills map[string]struct {
		BaseStat string `json:"baseStat" bson:"baseStat"`
		Name     string `json:"name" bson:"name"`
		Label    string `json:"label" bson:"label"`
		IsProf   int    `json:"isProf" bson:"isProf"`
	} `json:"skills" bson:"skills"`
	Vitality struct {
		HpDiceCurrent struct {
			Value int `json:"value" bson:"value"`
		} `json:"hp-dice-current" bson:"hp-dice-current"`
		HpDiceMulti map[string]interface{} `json:"hp-dice-multi" bson:"hp-dice-multi"`
		HitDie      struct {
			Value interface{} `json:"value" bson:"value"`
		} `json:"hit-die" bson:"hit-die"`
		HpCurrent struct {
			Value interface{} `json:"value" bson:"value"`
		} `json:"hp-current" bson:"hp-current"`
		Speed struct {
			Value interface{} `json:"value" bson:"value"`
		} `json:"speed" bson:"speed"`
		HpMax struct {
			Value interface{} `json:"value" bson:"value"`
		} `json:"hp-max" bson:"hp-max"`
		Ac struct {
			Value interface{} `json:"value" bson:"value"`
		} `json:"ac" bson:"ac"`
		IsDying        bool `json:"isDying" bson:"isDying"`
		DeathFails     int  `json:"deathFails" bson:"deathFails"`
		DeathSuccesses int  `json:"deathSuccesses" bson:"deathSuccesses"`
	} `json:"vitality" bson:"vitality"`
	WeaponsList []struct {
		Id   string `json:"id" bson:"id"`
		Name struct {
			Value string `json:"value" bson:"value"`
		} `json:"name" bson:"name"`
		Mod struct {
			Value string `json:"value" bson:"value"`
		} `json:"mod" bson:"mod"`
		Dmg struct {
			Value string `json:"value" bson:"value"`
		} `json:"dmg" bson:"dmg"`
		IsProf   bool `json:"isProf" bson:"isProf"`
		ModBonus struct {
			Value int `json:"value" bson:"value"`
		} `json:"modBonus" bson:"modBonus"`
	} `json:"weaponsList" bson:"weaponsList"`
	Weapons struct {
		WeaponName0 struct {
			Value string `json:"value" bson:"value"`
		} `json:"weaponName-0" bson:"weaponName-0"`
		WeaponMod0 struct {
			Value string `json:"value" bson:"value"`
		} `json:"weaponMod-0" bson:"weaponMod-0"`
		WeaponDmg0 struct {
			Value string `json:"value" bson:"value"`
		} `json:"weaponDmg-0" bson:"weaponDmg-0"`
		WeaponName1 struct {
			Value string `json:"value" bson:"value"`
		} `json:"weaponName-1" bson:"weaponName-1"`
		WeaponMod1 struct {
			Value string `json:"value" bson:"value"`
		} `json:"weaponMod-1" bson:"weaponMod-1"`
		WeaponDmg1 struct {
			Value string `json:"value" bson:"value"`
		} `json:"weaponDmg-1" bson:"weaponDmg-1"`
	} `json:"weapons" bson:"weapons"`
	Text struct {
		Attacks struct {
			Value struct {
				Data interface{} `json:"data" bson:"data"`
			} `json:"value" bson:"value"`
		} `json:"attacks" bson:"attacks"`
		Equipment struct {
			Value struct {
				Data struct {
					Type    string `json:"type" bson:"type"`
					Content []struct {
						Type    string `json:"type" bson:"type"`
						Content []struct {
							Type string `json:"type" bson:"type"`
							Text string `json:"text" bson:"text"`
						} `json:"content" bson:"content"`
					} `json:"content" bson:"content"`
				} `json:"data" bson:"data"`
			} `json:"value" bson:"value"`
		} `json:"equipment" bson:"equipment"`
		Prof struct {
			Value struct {
				Data struct {
					Type    string `json:"type" bson:"type"`
					Content []struct {
						Type    string `json:"type" bson:"type"`
						Content []struct {
							Type  string `json:"type" bson:"type"`
							Text  string `json:"text" bson:"text"`
							Marks []struct {
								Type string `json:"type" bson:"type"`
							} `json:"marks" bson:"marks"`
						} `json:"content" bson:"content"`
					} `json:"content" bson:"content"`
				} `json:"data" bson:"data"`
			} `json:"value" bson:"value"`
		} `json:"prof" bson:"prof"`
		Traits struct {
			Value struct {
				Data struct {
					Type    string `json:"type" bson:"type"`
					Content []struct {
						Type    string `json:"type" bson:"type"`
						Content []struct {
							Type  string `json:"type" bson:"type"`
							Text  string `json:"text" bson:"text"`
							Marks []struct {
								Type string `json:"type" bson:"type"`
							} `json:"marks" bson:"marks"`
						} `json:"content" bson:"content"`
					} `json:"content" bson:"content"`
				} `json:"data" bson:"data"`
			} `json:"value" bson:"value"`
		} `json:"traits" bson:"traits"`
		Allies struct {
			Value struct {
				Data interface{} `json:"data" bson:"data"`
			} `json:"value" bson:"value"`
		} `json:"allies" bson:"allies"`
		Features struct {
			Value struct {
				Data interface{} `json:"data" bson:"data"`
			} `json:"value" bson:"value"`
		} `json:"features" bson:"features"`
		Personality struct {
			Value struct {
				Data struct {
					Type    string `json:"type" bson:"type"`
					Content []struct {
						Type string `json:"type" bson:"type"`
					} `json:"content" bson:"content"`
				} `json:"data" bson:"data"`
			} `json:"value" bson:"value"`
			Size int `json:"size" bson:"size"`
		} `json:"personality" bson:"personality"`
		Ideals struct {
			Value struct {
				Data struct {
					Type    string `json:"type" bson:"type"`
					Content []struct {
						Type string `json:"type" bson:"type"`
					} `json:"content" bson:"content"`
				} `json:"data" bson:"data"`
			} `json:"value" bson:"value"`
		} `json:"ideals" bson:"ideals"`
		Bonds struct {
			Value struct {
				Data struct {
					Type    string `json:"type" bson:"type"`
					Content []struct {
						Type string `json:"type" bson:"type"`
					} `json:"content" bson:"content"`
				} `json:"data" bson:"data"`
			} `json:"value" bson:"value"`
		} `json:"bonds" bson:"bonds"`
		Flaws struct {
			Value struct {
				Data struct {
					Type    string `json:"type" bson:"type"`
					Content []struct {
						Type string `json:"type" bson:"type"`
					} `json:"content" bson:"content"`
				} `json:"data" bson:"data"`
			} `json:"value" bson:"value"`
		} `json:"flaws" bson:"flaws"`
		Background struct {
			Value struct {
				Data interface{} `json:"data" bson:"data"`
			} `json:"value" bson:"value"`
		} `json:"background" bson:"background"`
		Quests struct {
			Value struct {
				Data interface{} `json:"data" bson:"data"`
			} `json:"value" bson:"value"`
			Size int `json:"size" bson:"size"`
		} `json:"quests" bson:"quests"`
		SpellsLevel0 struct {
			Value struct {
				Data interface{} `json:"data" bson:"data"`
			} `json:"value" bson:"value"`
		} `json:"spells-level-0" bson:"spells-level-0"`
		SpellsLevel1 struct {
			Value struct {
				Data interface{} `json:"data" bson:"data"`
			} `json:"value" bson:"value"`
		} `json:"spells-level-1" bson:"spells-level-1"`
		SpellsLevel2 struct {
			Value struct {
				Data interface{} `json:"data" bson:"data"`
			} `json:"value" bson:"value"`
		} `json:"spells-level-2" bson:"spells-level-2"`
		SpellsLevel3 struct {
			Value struct {
				Data interface{} `json:"data" bson:"data"`
			} `json:"value" bson:"value"`
		} `json:"spells-level-3" bson:"spells-level-3"`
		SpellsLevel4 struct {
			Value struct {
				Data interface{} `json:"data" bson:"data"`
			} `json:"value" bson:"value"`
		} `json:"spells-level-4" bson:"spells-level-4"`
		SpellsLevel5 struct {
			Value struct {
				Data interface{} `json:"data" bson:"data"`
			} `json:"value" bson:"value"`
		} `json:"spells-level-5" bson:"spells-level-5"`
		SpellsLevel6 struct {
			Value struct {
				Data interface{} `json:"data" bson:"data"`
			} `json:"value" bson:"value"`
		} `json:"spells-level-6" bson:"spells-level-6"`
		SpellsLevel7 struct {
			Value struct {
				Data interface{} `json:"data" bson:"data"`
			} `json:"value" bson:"value"`
		} `json:"spells-level-7" bson:"spells-level-7"`
		SpellsLevel8 struct {
			Value struct {
				Data interface{} `json:"data" bson:"data"`
			} `json:"value" bson:"value"`
		} `json:"spells-level-8" bson:"spells-level-8"`
		SpellsLevel9 struct {
			Value struct {
				Data interface{} `json:"data" bson:"data"`
			} `json:"value" bson:"value"`
		} `json:"spells-level-9" bson:"spells-level-9"`
	} `json:"text" bson:"text"`
	Coins struct {
		Gp struct {
			Value interface{} `json:"value" bson:"value"`
		} `json:"gp" bson:"gp"`
		Total struct {
			Value interface{} `json:"value" bson:"value"`
		} `json:"total" bson:"total"`
		Sp struct {
			Value interface{} `json:"value" bson:"value"`
		} `json:"sp" bson:"sp"`
		Cp struct {
			Value interface{} `json:"value" bson:"value"`
		} `json:"cp" bson:"cp"`
		Pp struct {
			Value interface{} `json:"value" bson:"value"`
		} `json:"pp" bson:"pp"`
		Ep struct {
			Value interface{} `json:"value" bson:"value"`
		} `json:"ep" bson:"ep"`
	} `json:"coins" bson:"coins"`
	Resources     map[string]interface{} `json:"resources" bson:"resources"`
	BonusesSkills map[string]interface{} `json:"bonusesSkills" bson:"bonusesSkills"`
	BonusesStats  map[string]interface{} `json:"bonusesStats" bson:"bonusesStats"`
	Conditions    interface{}            `json:"conditions" bson:"conditions"`
	HiddenName    string                 `json:"hiddenName" bson:"hiddenName"`
	CasterClass   struct {
		Value string `json:"value" bson:"value"`
	} `json:"casterClass" bson:"casterClass"`
	Avatar    CharacterAvatar `json:"avatar" bson:"avatar"`
	CreatedAt string          `json:"createdAt" bson:"createdAt"`
}

type CharacterRaw struct {
	Tags           []string               `json:"tags" bson:"tags"`
	DisabledBlocks map[string]interface{} `json:"disabledBlocks" bson:"disabledBlocks"`
	Spells         struct {
		Mode     string   `json:"mode" bson:"mode"`
		Prepared []string `json:"prepared" bson:"prepared"`
		Book     []string `json:"book" bson:"book"`
	} `json:"spells" bson:"spells"`
	Data     string `json:"data" bson:"data"`
	JsonType string `json:"jsonType" bson:"jsonType"`
	Version  string `json:"version" bson:"version"`
}

type Character struct {
	ID             primitive.ObjectID     `bson:"_id" json:"id"`
	Tags           []string               `json:"tags" bson:"tags"`
	DisabledBlocks map[string]interface{} `json:"disabledBlocks" bson:"disabledBlocks"`
	Spells         struct {
		Mode     string   `json:"mode" bson:"mode"`
		Prepared []string `json:"prepared" bson:"prepared"`
		Book     []string `json:"book" bson:"book"`
	} `json:"spells" bson:"spells"`
	Data     CharacterData `json:"data" bson:"data"`
	JsonType string        `json:"jsonType" bson:"jsonType"`
	Version  string        `json:"version" bson:"version"`
}

type CharacterShort struct {
	ID             primitive.ObjectID `bson:"_id" json:"id"`
	CharClass      CharClass          `json:"charClass" bson:"charClass"`
	CharacterLevel CharacterLevel     `json:"level" bson:"level"`
	CharacterName  CharacterName      `json:"name" bson:"name"`
	CharacterRace  CharacterRace      `json:"race" bson:"race"`
	Avatar         CharacterAvatar    `json:"avatar" bson:"avatar"`
}

type CharacterReq struct {
	Start  int                   `json:"start"`
	Size   int                   `json:"size"`
	Search SearchParams          `json:"search"`
	Order  []Order               `json:"order"`
	Filter CharacterFilterParams `json:"filter"`
}
