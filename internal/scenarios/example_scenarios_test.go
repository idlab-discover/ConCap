package scenarios

import (
	"path/filepath"
	"testing"
)

func TestExampleScenariosParse(t *testing.T) {
	paths, err := filepath.Glob(filepath.Join("..", "..", "example", "scenarios", "*.yaml"))
	if err != nil {
		t.Fatalf("glob example scenarios: %v", err)
	}
	if len(paths) == 0 {
		t.Fatal("no example scenarios found")
	}

	for _, path := range paths {
		path := path
		t.Run(filepath.Base(path), func(t *testing.T) {
			t.Parallel()
			if _, err := CreateScenario(path); err != nil {
				t.Fatalf("parse example scenario %s: %v", path, err)
			}
		})
	}
}
