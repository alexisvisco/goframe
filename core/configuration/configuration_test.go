package configuration

import (
	"os"
	"testing"
)

func TestReplaceVariables(t *testing.T) {
	// Set up test environment variables
	os.Setenv("TEST_VAR", "test_value")
	os.Setenv("HOME", "/home/user")
	os.Setenv("EMPTY_VAR", "")
	os.Setenv("NUMBER_VAR", "123")
	os.Setenv("SPECIAL_CHARS", "hello@world.com")
	defer func() {
		os.Unsetenv("TEST_VAR")
		os.Unsetenv("HOME")
		os.Unsetenv("EMPTY_VAR")
		os.Unsetenv("NUMBER_VAR")
		os.Unsetenv("SPECIAL_CHARS")
	}()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Basic ${VAR} format tests
		{
			name:     "simple variable replacement",
			input:    "Hello ${TEST_VAR}!",
			expected: "Hello test_value!",
		},
		{
			name:     "multiple variables",
			input:    "${TEST_VAR} in ${HOME}",
			expected: "test_value in /home/user",
		},
		{
			name:     "variable at start",
			input:    "${TEST_VAR} world",
			expected: "test_value world",
		},
		{
			name:     "variable at end",
			input:    "Hello ${TEST_VAR}",
			expected: "Hello test_value",
		},
		{
			name:     "only variable",
			input:    "${TEST_VAR}",
			expected: "test_value",
		},

		// ${VAR:default} format tests
		{
			name:     "variable with default - variable exists",
			input:    "${TEST_VAR:default_value}",
			expected: "test_value",
		},
		{
			name:     "variable with default - variable doesn't exist",
			input:    "${NONEXISTENT:default_value}",
			expected: "default_value",
		},
		{
			name:     "variable with default - empty variable uses default",
			input:    "${EMPTY_VAR:default_value}",
			expected: "default_value",
		},
		{
			name:     "variable with empty default",
			input:    "${NONEXISTENT:}",
			expected: "",
		},
		{
			name:     "variable with spaces in default",
			input:    "${NONEXISTENT:hello world}",
			expected: "hello world",
		},
		{
			name:     "variable with colon in default",
			input:    "${NONEXISTENT:http://example.com}",
			expected: "http://example.com",
		},

		// $VAR format tests
		{
			name:     "simple dollar variable",
			input:    "Hello $TEST_VAR!",
			expected: "Hello test_value!",
		},
		{
			name:     "dollar variable with underscore",
			input:    "Value: $NUMBER_VAR",
			expected: "Value: 123",
		},
		{
			name:     "dollar variable nonexistent",
			input:    "Hello $NONEXISTENT!",
			expected: "Hello $NONEXISTENT!",
		},
		{
			name:     "dollar variable at end",
			input:    "Directory: $HOME",
			expected: "Directory: /home/user",
		},

		// Mixed format tests
		{
			name:     "mixed formats",
			input:    "${TEST_VAR} and $HOME and ${NUMBER_VAR:999}",
			expected: "test_value and /home/user and 123",
		},

		// Edge cases
		{
			name:     "no variables",
			input:    "Hello world!",
			expected: "Hello world!",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "just dollar sign",
			input:    "$",
			expected: "$",
		},
		{
			name:     "dollar with non-variable chars",
			input:    "$123abc",
			expected: "$123abc",
		},
		{
			name:     "unclosed brace",
			input:    "${TEST_VAR",
			expected: "${TEST_VAR",
		},
		{
			name:     "empty braces",
			input:    "${}",
			expected: "${}",
		},
		{
			name:     "nested braces (invalid)",
			input:    "${TEST_${VAR}}",
			expected: "${TEST_${VAR}}",
		},
		{
			name:     "multiple dollar signs",
			input:    "$$TEST_VAR",
			expected: "$$TEST_VAR",
		},
		{
			name:     "dollar at end",
			input:    "Hello$",
			expected: "Hello$",
		},

		// Special characters in variable values
		{
			name:     "special chars in value",
			input:    "Email: ${SPECIAL_CHARS}",
			expected: "Email: hello@world.com",
		},

		// Whitespace handling
		{
			name:     "spaces around variable name with default",
			input:    "${ TEST_VAR : default }",
			expected: "test_value",
		},
		{
			name:     "spaces around nonexistent variable with default",
			input:    "${ NONEXISTENT : spaced default }",
			expected: "spaced default",
		},

		// Complex scenarios
		{
			name:     "multiple replacements in path",
			input:    "${HOME}/config/${TEST_VAR}.conf",
			expected: "/home/user/config/test_value.conf",
		},
		{
			name:     "variable in quotes",
			input:    `"${TEST_VAR}"`,
			expected: `"test_value"`,
		},
		{
			name:     "variable with brackets around",
			input:    "[${TEST_VAR}]",
			expected: "[test_value]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := replaceVariables([]byte(tt.input))
			if string(result) != tt.expected {
				t.Errorf("replaceVariables(%q) = %q, want %q", tt.input, string(result), tt.expected)
			}
		})
	}
}

func TestReplaceVariablesNilInput(t *testing.T) {
	result := replaceVariables(nil)
	if result != nil {
		t.Errorf("replaceVariables(nil) = %v, want nil", result)
	}
}

// Benchmark tests
func BenchmarkReplaceVariables(b *testing.B) {
	os.Setenv("BENCH_VAR", "benchmark_value")
	defer os.Unsetenv("BENCH_VAR")

	input := []byte("Hello ${BENCH_VAR}, welcome to ${HOME:default_home}!")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		replaceVariables(input)
	}
}
