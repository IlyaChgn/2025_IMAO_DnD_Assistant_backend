package combatai

// ComputeIntelligence maps a creature's INT score (1-20) and a difficulty modifier
// to a float64 in the range [0.05, 1.0]. This value gates tactical decision quality:
//
//	INT 1-3   → 0.05-0.15 (attack nearest, sticky targeting)
//	INT 4-7   → 0.15-0.35 (basic tactics: finish wounded)
//	INT 8-11  → 0.35-0.55 (normal tactics: best action/target)
//	INT 12-15 → 0.55-0.75 (smart: focus fire, break concentration)
//	INT 16-20 → 0.75-1.0  (full tactics: AoE optimization, resource management)
func ComputeIntelligence(intScore int, difficultyMod float64) float64 {
	base := (float64(intScore) - 1.0) / 19.0
	return clamp(base+difficultyMod, 0.05, 1.0)
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
