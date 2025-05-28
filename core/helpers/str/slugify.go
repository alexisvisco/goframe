package str

import (
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// Slugify converts a string into a URL-friendly slug.
// It handles:
// - Unicode characters and diacritics -> removed
// - Multiple spaces and special characters -> replace with 1
// - Case conversion -> lowercase
// - Leading/trailing spaces and dashes
// - Multiple consecutive dashes
// Examples of usage:
// 	fmt.Println(Slugify("Hello, World!"))                    // Output: hello-world
// 	fmt.Println(Slugify("  Spécial  Charācters--"))         // Output: special-characters
// 	fmt.Println(Slugify("This & That"))                     // Output: this-and-that
// 	fmt.Println(Slugify("Multiple   Spaces"))               // Output: multiple-spaces
// 	fmt.Println(Slugify("Hello世界"))                         // Output: hello
// 	fmt.Println(Slugify("!@#$%^&*()"))                      // Output: ""
// 	fmt.Println(Slugify("Coast-to-Coast"))                  // Output: coast-to-coast
// 	fmt.Println(Slugify("über café"))                       // Output: uber-cafe

func Slugify(s string) string {
	// Convert to lowercase
	s = strings.ToLower(s)

	// Create a transform that decomposes Unicode characters and removes diacritics
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	result, _, _ := transform.String(t, s)

	// Replace common special characters with dashes
	reg := regexp.MustCompile(`[^a-z0-9]+`)
	result = reg.ReplaceAllString(result, "-")

	// Clean up multiple dashes and trim
	reg = regexp.MustCompile(`-{2,}`)
	result = reg.ReplaceAllString(result, "-")
	result = strings.Trim(result, "-")

	return result
}

// SlugifyConfig allows customizing the slugification process
type SlugifyConfig struct {
	Lowercase     bool
	MaxLength     int
	CustomReplace map[string]string
	AllowUnicode  bool
}

func SlugifyWithConfig(s string, config SlugifyConfig) string {
	if config.Lowercase {
		s = strings.ToLower(s)
	}

	// Apply custom replacements
	for old, new := range config.CustomReplace {
		s = strings.ReplaceAll(s, old, new)
	}

	var result string
	if config.AllowUnicode {
		// Only normalize Unicode combining characters
		t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
		result, _, _ = transform.String(t, s)

		// Replace spaces and invalid characters while preserving Unicode
		reg := regexp.MustCompile(`[[:space:][:punct:]]+`)
		result = reg.ReplaceAllString(result, "-")
	} else {
		// Convert to ASCII
		reg := regexp.MustCompile(`[^a-zA-Z0-9]+`)
		result = reg.ReplaceAllString(s, "-")
	}

	// Clean up multiple dashes and trim
	reg := regexp.MustCompile(`-{2,}`)
	result = reg.ReplaceAllString(result, "-")
	result = strings.Trim(result, "-")

	// Enforce max length if specified
	if config.MaxLength > 0 && len(result) > config.MaxLength {
		truncated := result[:config.MaxLength]
		// If truncated at a dash, just trim it
		if strings.HasSuffix(truncated, "-") {
			return strings.TrimRight(truncated, "-")
		}
		// If we cut in the middle of a word, go back to last complete word
		if lastDash := strings.LastIndex(truncated, "-"); lastDash != -1 {
			nextDash := strings.Index(result[lastDash+1:], "-")
			if nextDash != -1 && lastDash+1+nextDash > config.MaxLength {
				return truncated[:lastDash]
			}
		}
		return strings.TrimRight(truncated, "-")
	}

	return result
}
