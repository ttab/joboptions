package parser_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/ttab/joboptions/parser"
)

func TestScanner(t *testing.T) {
	validFiles, err := filepath.Glob(filepath.Join(
		"..", "testdata", "valid", "*.joboptions"))
	must(t, err, "glob for valid files")

	for _, p := range validFiles {
		t.Run(p, func(t *testing.T) {
			data, err := os.ReadFile(p)
			must(t, err, "read simple options")

			s := parser.NewScanner(data)

			for s.Scan() {
			}

			must(t, s.Err(), "parse without errors")
		})
	}
}

func must(t *testing.T, err error, format string, a ...any) {
	t.Helper()

	if err != nil {
		message := fmt.Sprintf(format, a...)
		t.Errorf("%s: %v", message, err)
	}
}
