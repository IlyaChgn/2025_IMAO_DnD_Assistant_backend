package repository

func detectLanguageField(value string) (string, bool) {
	var hasRussian, hasEnglish bool

	for _, r := range value {
		switch {
		case (r >= 'а' && r <= 'я') || (r >= 'А' && r <= 'Я') || r == 'ё' || r == 'Ё':
			hasRussian = true
		case (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z'):
			hasEnglish = true
		}

		if hasRussian && hasEnglish {
			return "", false
		}
	}

	if hasEnglish {
		return "name.eng", true
	}

	return "name.rus", true
}
