package str

import "testing"

func TestTruncateByWords(t *testing.T) {
	tests := []struct {
		name         string
		text         string
		maxWords     int
		ellipsisChar string
		want         string
	}{
		{
			name:         "empty string",
			text:         "",
			maxWords:     5,
			ellipsisChar: "...",
			want:         "",
		},
		{
			name:         "zero max words",
			text:         "hello world",
			maxWords:     0,
			ellipsisChar: "...",
			want:         "",
		},
		{
			name:         "negative max words",
			text:         "hello world",
			maxWords:     -1,
			ellipsisChar: "...",
			want:         "",
		},
		{
			name:         "single word",
			text:         "hello",
			maxWords:     1,
			ellipsisChar: "...",
			want:         "hello",
		},
		{
			name:         "exact word count",
			text:         "hello world",
			maxWords:     2,
			ellipsisChar: "...",
			want:         "hello world",
		},
		{
			name:         "truncate middle",
			text:         "hello world goodbye everyone",
			maxWords:     2,
			ellipsisChar: "...",
			want:         "hello world...",
		},
		{
			name:         "multiple spaces",
			text:         "hello    world    goodbye",
			maxWords:     2,
			ellipsisChar: "...",
			want:         "hello    world...",
		},
		{
			name:         "unicode text",
			text:         "こんにちは 世界 さようなら みなさん",
			maxWords:     2,
			ellipsisChar: "...",
			want:         "こんにちは 世界...",
		},
		{
			name:         "custom ellipsis",
			text:         "hello world goodbye",
			maxWords:     2,
			ellipsisChar: "…",
			want:         "hello world…",
		},
		{
			name:         "trailing spaces",
			text:         "hello world   ",
			maxWords:     1,
			ellipsisChar: "...",
			want:         "hello...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TruncateByWords(tt.text, tt.maxWords, tt.ellipsisChar)
			if got != tt.want {
				t.Errorf("TruncateByWords() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTruncateByCharLength(t *testing.T) {
	tests := []struct {
		name         string
		text         string
		maxLen       int
		ellipsisChar string
		want         string
	}{
		{
			name:         "empty string",
			text:         "",
			maxLen:       5,
			ellipsisChar: "...",
			want:         "",
		},
		{
			name:         "zero max length",
			text:         "hello",
			maxLen:       0,
			ellipsisChar: "...",
			want:         "",
		},
		{
			name:         "negative max length",
			text:         "hello",
			maxLen:       -1,
			ellipsisChar: "...",
			want:         "",
		},
		{
			name:         "exact length",
			text:         "hello",
			maxLen:       5,
			ellipsisChar: "...",
			want:         "hello",
		},
		{
			name:         "truncate with ellipsis",
			text:         "hello world",
			maxLen:       8,
			ellipsisChar: "...",
			want:         "hello...",
		},
		{
			name:         "max length less than ellipsis",
			text:         "hello",
			maxLen:       2,
			ellipsisChar: "...",
			want:         "he",
		},
		{
			name:         "unicode text",
			text:         "こんにちは世界",
			maxLen:       4,
			ellipsisChar: "...",
			want:         "こ...",
		},
		{
			name:         "custom ellipsis",
			text:         "hello world",
			maxLen:       6,
			ellipsisChar: "…",
			want:         "hello…",
		},
		{
			name:         "long ellipsis",
			text:         "hello world",
			maxLen:       10,
			ellipsisChar: ".....",
			want:         "hello.....",
		},
		{
			name:         "spaces at truncation point",
			text:         "hello     world",
			maxLen:       8,
			ellipsisChar: "...",
			want:         "hello...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TruncateByCharLength(tt.text, tt.maxLen, tt.ellipsisChar)
			if got != tt.want {
				t.Errorf("TruncateByCharLength() = %q, want %q", got, tt.want)
			}
		})
	}
}
