package scenarios

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteAnalysisLogPersistsProcessorProvenanceStdoutAndStderr(t *testing.T) {
	pod := &ProcessingPod{
		Name:           "rustiflow",
		ContainerImage: "ghcr.io/idlab-discover/rustiflow:slim",
		Command:        "rustiflow -f rustiflow\n  --header",
	}
	logPath := filepath.Join(t.TempDir(), "analysis.log")
	if err := writeAnalysisLog(logPath, pod, "hello", "warning"); err != nil {
		t.Fatalf("writeAnalysisLog returned error: %v", err)
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read analysis log: %v", err)
	}

	want := "processor:\n  name: rustiflow\n  image: ghcr.io/idlab-discover/rustiflow:slim\n  command: |-\n    rustiflow -f rustiflow\n      --header\nstdout:\nhello\nstderr:\nwarning\n"
	if got := string(data); got != want {
		t.Fatalf("analysis log = %q, want %q", got, want)
	}
}
