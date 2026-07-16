package main

import (
	"errors"
	"strings"
	"testing"

	"mira-tp4/internal/apiclient"
	"mira-tp4/internal/core"
)

func TestNormalizeLimit(t *testing.T) {
	tests := []struct {
		name     string
		limit    int
		def, max int
		want     int
		wantErr  bool
	}{
		{"absent falls back to default", 0, 10, 50, 10, false},
		{"within range is kept as is", 25, 10, 50, 25, false},
		{"above max is capped", 100, 10, 50, 50, false},
		{"exactly max is kept", 50, 10, 50, 50, false},
		{"negative is a validation error", -1, 10, 50, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeLimit(tt.limit, tt.def, tt.max)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("normalizeLimit(%d, %d, %d) error = nil, want error", tt.limit, tt.def, tt.max)
				}
				return
			}
			if err != nil {
				t.Fatalf("normalizeLimit(%d, %d, %d) unexpected error: %v", tt.limit, tt.def, tt.max, err)
			}
			if got != tt.want {
				t.Fatalf("normalizeLimit(%d, %d, %d) = %d, want %d", tt.limit, tt.def, tt.max, got, tt.want)
			}
		})
	}
}

func TestDescribeAPIError(t *testing.T) {
	t.Run("APIError surfaces the API's own message", func(t *testing.T) {
		err := describeAPIError(&apiclient.APIError{Status: 404, Code: "not_found", Message: "note not found"})
		if !strings.Contains(err.Error(), "note not found") {
			t.Fatalf("describeAPIError() = %q, want it to contain the API message", err.Error())
		}
	})

	t.Run("other errors get a generic connectivity message", func(t *testing.T) {
		err := describeAPIError(errors.New("connection refused"))
		if strings.Contains(err.Error(), "connection refused") {
			t.Fatalf("describeAPIError() = %q, should not leak the raw underlying error", err.Error())
		}
		if err.Error() == "" {
			t.Fatalf("describeAPIError() returned an empty message")
		}
	})
}

func TestFormatNote(t *testing.T) {
	score := 0.75
	note := core.Note{
		ID:               "note-1",
		Title:            "Test",
		Content:          "body",
		Status:           core.StatusActive,
		Tags:             []string{"a", "b"},
		EnrichmentStatus: core.EnrichmentDone,
		Summary:          "a short summary",
		Score:            &score,
	}

	out := formatNote(note)

	for _, want := range []string{"Test", "note-1", "active", "done", "a, b", "a short summary", "0.750", "body"} {
		if !strings.Contains(out, want) {
			t.Errorf("formatNote() missing %q in output:\n%s", want, out)
		}
	}
}

func TestFormatNotesList(t *testing.T) {
	t.Run("empty list", func(t *testing.T) {
		out := formatNotesList(nil)
		if !strings.Contains(out, "Aucune note") {
			t.Fatalf("formatNotesList(nil) = %q, want an empty-list message", out)
		}
	})

	t.Run("non-empty list mentions every note", func(t *testing.T) {
		notes := []core.Note{
			{ID: "1", Title: "First", Status: core.StatusActive, EnrichmentStatus: core.EnrichmentDone},
			{ID: "2", Title: "Second", Status: core.StatusActive, EnrichmentStatus: core.EnrichmentPending},
		}
		out := formatNotesList(notes)
		for _, want := range []string{"First", "Second", "2 note"} {
			if !strings.Contains(out, want) {
				t.Errorf("formatNotesList() missing %q in output:\n%s", want, out)
			}
		}
	})
}
