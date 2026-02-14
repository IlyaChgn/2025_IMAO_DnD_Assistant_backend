package dice

import (
	"fmt"
	"math/rand/v2"
	"regexp"
	"strconv"
)

var diceRegex = regexp.MustCompile(`^(\d+)d(\d+)([+-]\d+)?$`)

// RollResult holds the outcome of a parsed dice expression.
type RollResult struct {
	Rolls    []int `json:"rolls"`
	Modifier int   `json:"modifier"`
	Total    int   `json:"total"`
}

// Parse breaks a dice expression like "2d6+3" into its components.
func Parse(expr string) (count, sides, modifier int, err error) {
	matches := diceRegex.FindStringSubmatch(expr)
	if matches == nil {
		return 0, 0, 0, fmt.Errorf("invalid dice expression: %s", expr)
	}

	count, _ = strconv.Atoi(matches[1])
	sides, _ = strconv.Atoi(matches[2])

	if matches[3] != "" {
		modifier, _ = strconv.Atoi(matches[3])
	}

	if count < 1 || sides < 1 {
		return 0, 0, 0, fmt.Errorf("invalid dice expression: %s", expr)
	}

	return count, sides, modifier, nil
}

// RollDice rolls count dice of the given number of sides.
func RollDice(count, sides int) (rolls []int, total int) {
	rolls = make([]int, count)
	for i := range count {
		r := rand.IntN(sides) + 1
		rolls[i] = r
		total += r
	}

	return rolls, total
}

// Roll parses a dice expression and rolls it.
func Roll(expr string) (RollResult, error) {
	count, sides, modifier, err := Parse(expr)
	if err != nil {
		return RollResult{}, err
	}

	rolls, total := RollDice(count, sides)

	return RollResult{
		Rolls:    rolls,
		Modifier: modifier,
		Total:    total + modifier,
	}, nil
}

// RollD20 rolls a d20 with a modifier, handling advantage and disadvantage.
// Returns the natural roll, the total (natural + mod), and all individual rolls.
func RollD20(mod int, adv, disadv bool) (natural, total int, rolls []int) {
	if adv && disadv {
		// Advantage and disadvantage cancel out
		adv = false
		disadv = false
	}

	r1 := rand.IntN(20) + 1

	if !adv && !disadv {
		return r1, r1 + mod, []int{r1}
	}

	r2 := rand.IntN(20) + 1
	rolls = []int{r1, r2}

	if adv {
		natural = max(r1, r2)
	} else {
		natural = min(r1, r2)
	}

	return natural, natural + mod, rolls
}
