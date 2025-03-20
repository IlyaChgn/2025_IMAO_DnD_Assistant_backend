package utils

import (
	"strings"
	"unicode"
)

// normalizeText удаляет специальные символы и приводит текст к нижнему регистру
func NormalizeText(text string) string {
	// Удаляем специальные символы
	var normalized strings.Builder
	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSpace(r) {
			normalized.WriteRune(r)
		}
	}
	// Приводим к нижнему регистру
	return strings.ToLower(normalized.String())
}
