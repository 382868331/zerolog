package zerolog_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/rs/zerolog"
)

// TestTaskZER01ConsoleWriterNoDuplicateFields is a hidden regression test for
// task bugfix_zerolog_001: ConsoleWriter renders parts via PartsOrder first and
// then walks all event fields again in writeFields. When PartsOrder contains
// custom (non-standard) part names, those names are rendered twice, producing
// duplicated keys in the console output.
//
// Hidden verification direction: custom PartsOrder containing extra fields;
// assert each field appears exactly once.
func TestTaskZER01ConsoleWriterNoDuplicateFields(t *testing.T) {
	buf := &bytes.Buffer{}
	w := zerolog.ConsoleWriter{
		Out:     buf,
		NoColor: true,
		PartsOrder: []string{
			zerolog.LevelFieldName,
			"pkg",
			"fn",
			zerolog.MessageFieldName,
		},
	}

	evt := `{"level": "info", "message": "This is a test.", "pkg": "main", "fn": "TestFunc"}`
	if _, err := w.Write([]byte(evt)); err != nil {
		t.Fatalf("Unexpected error when writing output: %s", err)
	}

	out := buf.String()

	// Custom parts must not be re-printed as regular key=value fields.
	if strings.Contains(out, "pkg=") || strings.Contains(out, "fn=") {
		t.Errorf("duplicated custom part fields in output: %q", out)
	}

	// Each custom part value must appear exactly once (only in its part slot).
	if got := strings.Count(out, "main"); got != 1 {
		t.Errorf("pkg value appears %d times in output (want 1): %q", got, out)
	}
	if got := strings.Count(out, "TestFunc"); got != 1 {
		t.Errorf("fn value appears %d times in output (want 1): %q", got, out)
	}
}
