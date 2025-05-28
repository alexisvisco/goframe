package str

import (
	"strings"
	"sync"
)

// ToSnakeCase converts a string to snake_case
func ToSnakeCase(s string) string {
	return ToDelimitedCase(s, '_')
}

func ToSnakeCaseWithIgnore(s string, ignore string) string {
	return ToScreamingDelimitedCase(s, '_', ignore, false)
}

// ToScreamingSnakeCase converts a string to SCREAMING_SNAKE_CASE
func ToScreamingSnakeCase(s string) string {
	return ToScreamingDelimitedCase(s, '_', "", true)
}

// ToKebabCase converts a string to kebab-case
func ToKebabCase(s string) string {
	return ToDelimitedCase(s, '-')
}

// ToScreamingKebab converts a string to SCREAMING-KEBAB-CASE
func ToScreamingKebab(s string) string {
	return ToScreamingDelimitedCase(s, '-', "", true)
}

// ToDelimitedCase converts a string to delimited.snake.case
// (in this case `delimiter = '.'`)
func ToDelimitedCase(s string, delimiter uint8) string {
	return ToScreamingDelimitedCase(s, delimiter, "", false)
}

// ToScreamingDelimitedCase converts a string to SCREAMING.DELIMITED.SNAKE.CASE
// (in this case `delimiter = '.'; screaming = true`)
// or delimited.snake.case
// (in this case `delimiter = '.'; screaming = false`)
func ToScreamingDelimitedCase(s string, delimiter uint8, ignore string, screaming bool) string {
	s = strings.TrimSpace(s)
	n := strings.Builder{}
	n.Grow(len(s) + 2) // nominal 2 bytes of extra space for inserted delimiters
	for i, v := range []byte(s) {
		vIsCap := v >= 'A' && v <= 'Z'
		vIsLow := v >= 'a' && v <= 'z'
		if vIsLow && screaming {
			v += 'A'
			v -= 'a'
		} else if vIsCap && !screaming {
			v += 'a'
			v -= 'A'
		}

		// treat acronyms as words, eg for JSONData -> JSON is a whole word
		if i+1 < len(s) {
			next := s[i+1]
			vIsNum := v >= '0' && v <= '9'
			nextIsCap := next >= 'A' && next <= 'Z'
			nextIsLow := next >= 'a' && next <= 'z'
			nextIsNum := next >= '0' && next <= '9'
			// add underscore if next letter case type is changed
			if (vIsCap && (nextIsLow || nextIsNum)) || (vIsLow && (nextIsCap || nextIsNum)) || (vIsNum && (nextIsCap || nextIsLow)) {
				prevIgnore := ignore != "" && i > 0 && strings.ContainsAny(string(s[i-1]), ignore)
				if !prevIgnore {
					if vIsCap && nextIsLow {
						if prevIsCap := i > 0 && s[i-1] >= 'A' && s[i-1] <= 'Z'; prevIsCap {
							n.WriteByte(delimiter)
						}
					}
					n.WriteByte(v)
					if vIsLow || vIsNum || nextIsNum {
						n.WriteByte(delimiter)
					}
					continue
				}
			}
		}

		if (v == ' ' || v == '_' || v == '-' || v == '.') && !strings.ContainsAny(string(v), ignore) {
			// replace space/underscore/hyphen/dot with delimiter
			n.WriteByte(delimiter)
		} else {
			n.WriteByte(v)
		}
	}

	return n.String()
}

// Converts a string to CamelCase
func toCamelInitCase(s string, initCase bool) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	a, hasAcronym := uppercaseAcronym.Load(s)
	if hasAcronym {
		s = a.(string)
	}

	n := strings.Builder{}
	n.Grow(len(s))
	capNext := initCase
	prevIsCap := false
	for i, v := range []byte(s) {
		vIsCap := v >= 'A' && v <= 'Z'
		vIsLow := v >= 'a' && v <= 'z'
		if capNext {
			if vIsLow {
				v += 'A'
				v -= 'a'
			}
		} else if i == 0 {
			if vIsCap {
				v += 'a'
				v -= 'A'
			}
		} else if prevIsCap && vIsCap && !hasAcronym {
			v += 'a'
			v -= 'A'
		}
		prevIsCap = vIsCap

		if vIsCap || vIsLow {
			n.WriteByte(v)
			capNext = false
		} else if vIsNum := v >= '0' && v <= '9'; vIsNum {
			n.WriteByte(v)
			capNext = true
		} else {
			capNext = v == '_' || v == ' ' || v == '-' || v == '.'
		}
	}
	return n.String()
}

// ToPascalCase converts a string to CamelCase
func ToPascalCase(s string) string {
	return toCamelInitCase(s, true)
}

// ToCamelCase converts a string to lowerCamelCase
func ToCamelCase(s string) string {
	return toCamelInitCase(s, false)
}

var uppercaseAcronym = sync.Map{}

func init() {
	ConfigureAcronym("ID", "id")
	ConfigureAcronym("UUID", "uuid")
	ConfigureAcronym("URL", "url")
	ConfigureAcronym("API", "api")
	ConfigureAcronym("HTTP", "http")
	ConfigureAcronym("HTTPS", "https")
	ConfigureAcronym("JSON", "json")
	ConfigureAcronym("XML", "xml")
	ConfigureAcronym("HTML", "html")
	ConfigureAcronym("CSS", "css")
	ConfigureAcronym("JS", "js")
	ConfigureAcronym("JWT", "jwt")
	ConfigureAcronym("SHA", "sha")
	ConfigureAcronym("MD", "md")
	ConfigureAcronym("SQL", "sql")
	ConfigureAcronym("DB", "db")
	ConfigureAcronym("URI", "uri")
}

// ConfigureAcronym allows you to add additional words which will be considered acronyms
func ConfigureAcronym(key, val string) {
	uppercaseAcronym.Store(key, val)
}
