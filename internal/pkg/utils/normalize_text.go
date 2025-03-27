package utils

import (
	"strings"
	"unicode"
)

// NormalizeText удаляет специальные символы и приводит текст к нижнему регистру
func NormalizeText(text string) string {
	var normalized strings.Builder

	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSpace(r) {
			normalized.WriteRune(r)
		}
	}

	return strings.ToLower(normalized.String())
}

func RemoveBackslashes(input string) string {
	return strings.ReplaceAll(input, "\\", "")
}
