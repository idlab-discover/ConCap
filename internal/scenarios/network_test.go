package scenarios

import (
	"strings"
	"testing"
)

func TestGetTCCommandWithoutShapingShowsInitialQdisc(t *testing.T) {
	command := (&Network{}).GetTCCommand()

	if strings.Contains(command, "qdisc after") {
		t.Fatalf("GetTCCommand() unexpectedly logged post-shaping qdisc: %q", command)
	}

	expected := "printf 'qdisc before:\\n' && tc qdisc show dev eth0"
	if command != expected {
		t.Fatalf("GetTCCommand() = %q, want %q", command, expected)
	}
}

func TestGetTCCommandWithShapingShowsUpdatedQdisc(t *testing.T) {
	command := (&Network{
		Bandwidth:    "100mbit",
		QueueSize:    "100ms",
		Limit:        "10000",
		Delay:        "100ms",
		Jitter:       "20ms",
		Distribution: "normal",
		Seed:         "7",
	}).GetTCCommand()

	checks := []string{
		"printf 'qdisc before:\\n'",
		"tc qdisc show dev eth0",
		"tc qdisc replace dev eth0 root handle 1: tbf rate 100mbit burst 62500 latency 100ms",
		"tc qdisc replace dev eth0 parent 1:1 netem limit 10000 delay 100ms 20ms distribution normal seed 7",
		"printf 'qdisc after:\\n'",
	}

	for _, check := range checks {
		if !strings.Contains(command, check) {
			t.Fatalf("GetTCCommand() missing %q in %q", check, command)
		}
	}

	if strings.Count(command, "tc qdisc show dev eth0") != 2 {
		t.Fatalf("GetTCCommand() should show qdisc twice, got %q", command)
	}
}
