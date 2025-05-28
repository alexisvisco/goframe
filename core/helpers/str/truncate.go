package str

import (
	"strings"
	"unicode"
)

// TruncateByWords truncates text to a maximum number of words,
// appending ellipsis characters if truncation occurs.
// If maxWords <= 0, returns an empty string.
// If text is empty, returns an empty string.
func TruncateByWords(text string, maxWords int, ellipsisChars string) string {
	if maxWords <= 0 || text == "" {
		return ""
	}

	words := 0
	lastIndex := 0

	// Iterate through runes to properly handle UTF-8
	for i, r := range text {
		// Count word boundaries
		if i > 0 && unicode.IsSpace(r) && !unicode.IsSpace(rune(text[i-1])) {
			words++
			if words >= maxWords {
				lastIndex = i
				break
			}
		}
	}

	// Handle case where word count wasn't reached
	if words < maxWords {
		return text
	}

	// Trim trailing spaces before adding ellipsis
	return strings.TrimRightFunc(text[:lastIndex], unicode.IsSpace) + ellipsisChars
}

// TruncateByCharLength truncates text to a maximum length in characters,
// appending ellipsis characters if truncation occurs.
// If maxLen <= 0, returns an empty string.
// If text is empty, returns an empty string.
func TruncateByCharLength(text string, maxLen int, ellipsisChars string) string {
	if maxLen <= 0 || text == "" {
		return ""
	}

	textRunes := []rune(text)
	if len(textRunes) <= maxLen {
		return text
	}

	// Account for ellipsis in max length
	ellipsisLen := len([]rune(ellipsisChars))
	if maxLen <= ellipsisLen {
		return string(textRunes[:maxLen])
	}

	return string(textRunes[:maxLen-ellipsisLen]) + ellipsisChars
}
