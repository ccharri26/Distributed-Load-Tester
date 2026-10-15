package loadgen

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestOutputResults_WritesFileWithCorrectContent(t *testing.T) {
	// Arrange: use a temp dir so we don't pollute the working directory.
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })

	s := &Summary{
		ScheduledRequests:      100,
		StartedRequests:        100,
		CompletedRequests:      95,
		LateCompletedRequests:  3,
		SuccessfulRequests:     90,
		FailedRequests:         5,
		MinRequestDuration:     10 * time.Millisecond,
		MaxRequestDuration:     500 * time.Millisecond,
		AverageRequestDuration: 150 * time.Millisecond,
		P99:                    490000000,
		P95:                    450000000,
		P50:                    140000000,
	}

	// Act
	s.OutputResults()

	// Assert: the file was created and contains the expected values.
	data, err := os.ReadFile("TestResults.txt")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	content := string(data)

	expectations := []struct {
		label string
		want  string
	}{
		{"header", "Load Testing Results:"},
		{"scheduled", "Scheduled Requests: 100"},
		{"started", "Started Requests: 100"},
		{"completed", "Completed Requests: 95"},
		{"late", "Requests Completed After Deadline: 3"},
		{"successful", "Successful Requests: 90"},
		{"failed", "Failed Requests: 5"},
		{"fastest", "Fastest Request: 10ms"},
		{"slowest", "Slowest Request: 500ms"},
		{"average", "Average Request Time: 150ms"},
	}

	for _, tc := range expectations {
		if !strings.Contains(content, tc.want) {
			t.Errorf("output missing %s: want substring %q in:\n%s", tc.label, tc.want, content)
		}
	}
}

func TestOutputResults_ZeroSummary(t *testing.T) {
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })

	s := &Summary{}

	// Act: a zero-valued summary should still write a file without panicking.
	s.OutputResults()

	data, err := os.ReadFile("TestResults.txt")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "Scheduled Requests: 0") {
		t.Errorf("zero summary should contain 'Scheduled Requests: 0', got:\n%s", content)
	}
}

func TestOutputResults_OverwritesExistingFile(t *testing.T) {
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })

	// Write an initial file.
	first := &Summary{ScheduledRequests: 1}
	first.OutputResults()

	// Overwrite with a different summary.
	second := &Summary{ScheduledRequests: 999}
	second.OutputResults()

	data, err := os.ReadFile("TestResults.txt")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "Scheduled Requests: 999") {
		t.Errorf("file should reflect the second write, got:\n%s", content)
	}
	if strings.Contains(content, "Scheduled Requests: 1 ") {
		t.Errorf("file still contains the first write")
	}
}
