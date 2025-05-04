package model

import (
	"testing"
	"time"
)

func TestLifetime_Validate(t *testing.T) {
	validInputs := []Lifetime{
		"+100 days",
		"45 minutes",
		"-3 hours",
		"10 seconds",
		"   +1 day ",
	}
	invalidInputs := []Lifetime{
		"",
		"3",
		"100 lightyears",
		"day 100",
		"++ 2 days",
		"3days",
	}

	for _, input := range validInputs {
		if err := input.Validate(); err != nil {
			t.Errorf("expected valid, got error: %v", err)
		}
	}

	for _, input := range invalidInputs {
		if err := input.Validate(); err == nil {
			t.Errorf("expected error for %q, got none", input)
		}
	}
}

func TestLifetime_Duration(t *testing.T) {
	tests := []struct {
		in       Lifetime
		expected time.Duration
	}{
		{"1 day", 24 * time.Hour},
		{"+2 hours", 2 * time.Hour},
		{"-15 minutes", -15 * time.Minute},
		{"30 seconds", 30 * time.Second},
	}

	for _, tt := range tests {
		got, err := tt.in.Duration()
		if err != nil {
			t.Errorf("unexpected error for %q: %v", tt.in, err)
			continue
		}
		if got != tt.expected {
			t.Errorf("wrong duration for %q: got %v, want %v", tt.in, got, tt.expected)
		}
	}
}

func TestLifetime_Duration_Invalid(t *testing.T) {
	_, err := Lifetime("foo bar").Duration()
	if err == nil {
		t.Error("expected error for invalid input, got nil")
	}
}
