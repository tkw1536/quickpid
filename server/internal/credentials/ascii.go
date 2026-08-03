//spellchecker:words credentials
package credentials

// the different between uppercase and lowercase ASCII characters.
const asciiDifference = 'a' - 'A'

// equalFoldAscii is like [strings.EqualFold] but only performs ASCII case folding.
//
// Any two non-ASCII characters are only considered equal if they are the exact same rune.
func equalFoldAscii(a, b string) bool {
	runesA := []rune(a)
	runesB := []rune(b)
	if len(runesA) != len(runesB) {
		return false
	}

	for i := range runesA {
		c1, c2 := runesA[i], runesB[i]
		if 'A' <= c1 && c1 <= 'Z' {
			c1 += asciiDifference
		}
		if 'A' <= c2 && c2 <= 'Z' {
			c2 += asciiDifference
		}
		if c1 != c2 {
			return false
		}
	}

	return true
}

// hasPrefixAsciiFold is like [strings.HasPrefix] but checks the prefix under [equalFoldAscii].
func hasPrefixAsciiFold(s, prefix string) bool {
	if len(s) < len(prefix) {
		return false
	}
	return equalFoldAscii(s[:len(prefix)], prefix)
}

// trimPrefixAsciiFold is like [strings.TrimPrefix] but uses [hasPrefixAsciiFold] internally.
func trimPrefixAsciiFold(s, prefix string) string {
	if !hasPrefixAsciiFold(s, prefix) {
		return s
	}
	return s[len(prefix):]
}
