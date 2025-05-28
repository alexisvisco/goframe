package str

import (
	"testing"
)

func TestSlugify(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "basic conversion",
			input:    "Hello, World!",
			expected: "hello-world",
		},
		{
			name:     "special characters and spaces",
			input:    "  Spécial  Charācters--",
			expected: "special-characters",
		},
		{
			name:     "ampersand handling",
			input:    "This & That",
			expected: "this-that",
		},
		{
			name:     "multiple spaces",
			input:    "Multiple   Spaces",
			expected: "multiple-spaces",
		},
		{
			name:     "unicode characters",
			input:    "Hello世界",
			expected: "hello",
		},
		{
			name:     "only special characters",
			input:    "!@#$%^&*()",
			expected: "",
		},
		{
			name:     "existing hyphens",
			input:    "Coast-to-Coast",
			expected: "coast-to-coast",
		},
		{
			name:     "diacritics",
			input:    "über café",
			expected: "uber-cafe",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "numbers",
			input:    "2024 Year",
			expected: "2024-year",
		},
		{
			name:     "multiple dashes cleanup",
			input:    "multiple---dashes",
			expected: "multiple-dashes",
		},
		{
			name:     "leading and trailing spaces",
			input:    "  trimmed  ",
			expected: "trimmed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Slugify(tt.input)
			if got != tt.expected {
				t.Errorf("Slugify() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestSlugifyWithConfig(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		config   SlugifyConfig
		expected string
	}{
		{
			name:  "basic with custom replacements",
			input: "Hello & World @ 2024!",
			config: SlugifyConfig{
				Lowercase: true,
				CustomReplace: map[string]string{
					"&": "and",
					"@": "at",
				},
			},
			expected: "hello-and-world-at-2024",
		},
		{
			name:  "max length enforcement",
			input: "This is a very long title that should be truncated",
			config: SlugifyConfig{
				Lowercase: true,
				MaxLength: 20,
			},
			expected: "this-is-a-very-long",
		},
		{
			name:  "allow unicode",
			input: "Hello 世界",
			config: SlugifyConfig{
				Lowercase:    true,
				AllowUnicode: true,
			},
			expected: "hello-世界",
		},
		{
			name:  "case preservation",
			input: "Hello World",
			config: SlugifyConfig{
				Lowercase: false,
			},
			expected: "Hello-World",
		},
		{
			name:  "multiple config options",
			input: "Hello & World @ 2024! 世界",
			config: SlugifyConfig{
				Lowercase: true,
				MaxLength: 15,
				CustomReplace: map[string]string{
					"&": "and",
					"@": "at",
				},
				AllowUnicode: false,
			},
			expected: "hello-and-world",
		},
		{
			name:     "empty config",
			input:    "Hello World!",
			config:   SlugifyConfig{},
			expected: "Hello-World",
		},
		{
			name:  "custom replacements only",
			input: "Contact @ email & phone",
			config: SlugifyConfig{
				CustomReplace: map[string]string{
					"@": "at",
					"&": "and",
				},
			},
			expected: "Contact-at-email-and-phone",
		},
		{
			name:  "max length with exact word boundary",
			input: "one-two-three-four-five",
			config: SlugifyConfig{
				MaxLength: 11,
			},
			expected: "one-two",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SlugifyWithConfig(tt.input, tt.config)
			if got != tt.expected {
				t.Errorf("SlugifyWithConfig() = %v, want %v", got, tt.expected)
			}
		})
	}
}
