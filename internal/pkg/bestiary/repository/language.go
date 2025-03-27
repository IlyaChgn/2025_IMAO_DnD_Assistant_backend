package repository

import "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"

func detectLanguageField(value string) (string, error) {
	hasRussian, hasEnglish := false, false

	for _, r := range value {
		switch {
		case (r >= 'а' && r <= 'я') || (r >= 'А' && r <= 'Я') || r == 'ё' || r == 'Ё':
			hasRussian = true
		case (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z'):
			hasEnglish = true
		}

		if hasRussian && hasEnglish {
			return "", apperrors.MixedLangsError
		}
	}

	if hasEnglish {
		return "name.eng", nil
	}

	return "name.rus", nil
}
