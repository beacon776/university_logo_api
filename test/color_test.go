package test

import "testing"

func TestNormalizeColor(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"red", "#FF0000"},
		{"  blue  ", "#0000FF"},
		{"#f00", "#FF0000"},
		{"#f00f", "#FF0000"},     // 4位HEX
		{"#FF000080", "#FF0000"}, // 8位HEX
		{"rgb(255,0,0)", "#FF0000"},
		{"rgba(0,255,0,0.5)", "#00FF00"},
		{"transparent", ""},
		{"invalid", ""},
	}

	for _, tt := range tests {
		result := NormalizeColor(tt.input)
		if result != tt.expected {
			t.Errorf("input: %s, expected: %s, got: %s", tt.input, tt.expected, result)
		}
	}
}
