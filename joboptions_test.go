package joboptions_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ttab/joboptions"
)

func TestParse_Valid(t *testing.T) {
	regenerate := os.Getenv("REGENERATE") == "true"

	validFiles, err := filepath.Glob("testdata/valid/*.joboptions")
	must(t, err, "glob for valid files")

	for _, p := range validFiles {
		t.Run(p, func(t *testing.T) {
			data, err := os.ReadFile(p)
			must(t, err, "read input file")

			got, err := joboptions.Parse(data)
			must(t, err, "parse input file")

			goldenPath := p + ".json"

			if regenerate {
				data, err := json.MarshalIndent(got, "", "  ")
				must(t, err, "marshal golden file data")

				err = os.WriteFile(goldenPath, data, 0o600)
				must(t, err, "write golden file")
			}

			wantData, err := os.ReadFile(goldenPath)
			must(t, err, "read golden file")

			var want joboptions.Parameters

			err = json.Unmarshal(wantData, &want)
			must(t, err, "unmarshal golden file")

			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("Parse() mismatch (-want +got):\n%s", diff)
			}
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
