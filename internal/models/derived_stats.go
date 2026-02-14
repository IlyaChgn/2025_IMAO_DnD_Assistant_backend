package models

// DerivedStats holds all computed values derived from a CharacterBase.
// Produced by compute.ComputeDerived — never stored in the database.
type DerivedStats struct {
	AbilityModifiers     map[string]int            `json:"abilityModifiers"`
	ProficiencyBonus     int                       `json:"proficiencyBonus"`
	SkillBonuses         map[string]BonusBreakdown `json:"skillBonuses"`
	SaveBonuses          map[string]BonusBreakdown `json:"saveBonuses"`
	ArmorClass           int                       `json:"armorClass"`
	MaxHp                int                       `json:"maxHp"`
	Speed                SpeedDerived              `json:"speed"`
	Initiative           InitiativeDerived         `json:"initiative"`
	PassivePerception    int                       `json:"passivePerception"`
	PassiveInvestigation int                       `json:"passiveInvestigation"`
	PassiveInsight       int                       `json:"passiveInsight"`
	Resistances          []string                  `json:"resistances"`
	Immunities           []string                  `json:"immunities"`
	Vulnerabilities      []string                  `json:"vulnerabilities"`
	Spellcasting         *SpellcastingDerived       `json:"spellcasting,omitempty"`
}

// BonusBreakdown shows how a skill or save bonus is computed.
type BonusBreakdown struct {
	Total       int `json:"total"`
	AbilityMod  int `json:"abilityMod"`
	Proficiency int `json:"proficiency"`
	Expertise   int `json:"expertise"`
	Other       int `json:"other"`
}

// SpeedDerived holds computed speed values.
type SpeedDerived struct {
	Walk int `json:"walk"`
}

// InitiativeDerived holds computed initiative values.
type InitiativeDerived struct {
	Bonus int `json:"bonus"`
}

// SpellcastingDerived holds computed spellcasting stats.
type SpellcastingDerived struct {
	SpellSaveDC          int               `json:"spellSaveDC"`
	SpellAttackBonus     int               `json:"spellAttackBonus"`
	SpellSaveDCBreakdown string            `json:"spellSaveDCBreakdown"`
	SpellAttackBreakdown string            `json:"spellAttackBreakdown"`
	MaxSpellSlots        map[int]int       `json:"maxSpellSlots"`
	PactMagic            *PactMagicDerived `json:"pactMagic,omitempty"`
	MaxPreparedSpells    *int              `json:"maxPreparedSpells,omitempty"`
	MaxCantripsKnown     int               `json:"maxCantripsKnown"`
}

// PactMagicDerived holds warlock pact magic slot info.
type PactMagicDerived struct {
	MaxSlots  int `json:"maxSlots"`
	SlotLevel int `json:"slotLevel"`
}

// ComputedCharacterResponse is the wire format for GET /characters/:id/computed.
type ComputedCharacterResponse struct {
	Base    *CharacterBase `json:"base"`
	Derived *DerivedStats  `json:"derived"`
}
