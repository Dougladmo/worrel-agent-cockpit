package user

import "strings"

// accentFold mapeia caracteres acentuados comuns do PT-BR para ASCII.
var accentFold = map[rune]rune{
	'á': 'a', 'à': 'a', 'ã': 'a', 'â': 'a', 'ä': 'a',
	'é': 'e', 'ê': 'e', 'è': 'e', 'ë': 'e',
	'í': 'i', 'î': 'i', 'ì': 'i', 'ï': 'i',
	'ó': 'o', 'ô': 'o', 'õ': 'o', 'ò': 'o', 'ö': 'o',
	'ú': 'u', 'û': 'u', 'ù': 'u', 'ü': 'u',
	'ç': 'c', 'ñ': 'n',
}

// fold devolve s em minúsculas e sem acentos (ASCII-fold). Base para casar
// marcadores sem depender de \b com caracteres acentuados (que é frágil).
func fold(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if m, ok := accentFold[r]; ok {
			b.WriteRune(m)
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// norm normaliza um slot da chave canônica: minúsculas, sem acentos, só
// alfanumérico e espaços colapsados. "GET /users" -> "get users".
func norm(s string) string {
	folded := fold(s)
	var b strings.Builder
	prevSpace := false
	for _, r := range folded {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			prevSpace = false
		default:
			if !prevSpace {
				b.WriteRune(' ')
				prevSpace = true
			}
		}
	}
	return strings.TrimSpace(b.String())
}

// containsAny reporta se o texto (já folded) contém algum dos marcadores.
func containsAny(folded string, markers []string) bool {
	for _, m := range markers {
		if strings.Contains(folded, m) {
			return true
		}
	}
	return false
}
