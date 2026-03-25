package scenarios

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAddColumnsToCSVAppendsHeadersAndValues(t *testing.T) {
	pod := &ProcessingPod{}
	csvPath := filepath.Join(t.TempDir(), "flows.csv")
	if err := os.WriteFile(csvPath, []byte("flow_id\n1\n2\n"), 0644); err != nil {
		t.Fatalf("write csv fixture: %v", err)
	}

	if err := pod.AddColumnsToCSV(csvPath, []string{"label", "target"}, []string{"benign", "svc-a"}, true); err != nil {
		t.Fatalf("AddColumnsToCSV returned error: %v", err)
	}

	data, err := os.ReadFile(csvPath)
	if err != nil {
		t.Fatalf("read updated csv: %v", err)
	}

	want := "flow_id,label,target\n1,benign,svc-a\n2,benign,svc-a\n"
	if got := string(data); got != want {
		t.Fatalf("updated csv = %q, want %q", got, want)
	}
}

func TestAddColumnsToCSVRejectsEmptyFiles(t *testing.T) {
	pod := &ProcessingPod{}
	csvPath := filepath.Join(t.TempDir(), "empty.csv")
	if err := os.WriteFile(csvPath, nil, 0644); err != nil {
		t.Fatalf("write empty csv fixture: %v", err)
	}

	err := pod.AddColumnsToCSV(csvPath, []string{"label"}, []string{"benign"}, true)
	if err == nil {
		t.Fatal("AddColumnsToCSV returned nil error for empty CSV")
	}
	if !strings.Contains(err.Error(), "csv file is empty") {
		t.Fatalf("AddColumnsToCSV error = %v, want csv file is empty", err)
	}
}

func TestWriteAnalysisLogPersistsStdoutAndStderr(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "analysis.log")
	if err := writeAnalysisLog(logPath, "hello", "warning"); err != nil {
		t.Fatalf("writeAnalysisLog returned error: %v", err)
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read analysis log: %v", err)
	}

	want := "stdout:\nhello\nstderr:\nwarning\n"
	if got := string(data); got != want {
		t.Fatalf("analysis log = %q, want %q", got, want)
	}
}
