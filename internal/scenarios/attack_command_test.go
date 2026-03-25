package scenarios

import "testing"

func TestWithAttackLoggingTrimsTrailingWhitespace(t *testing.T) {
	got := withAttackLogging("echo hi\n")
	want := "(echo hi) 2>&1 | tee -a /logs/attacker.log | tee -a /proc/1/fd/1"
	if got != want {
		t.Fatalf("withAttackLogging() = %q, want %q", got, want)
	}
}

func TestWithAttackLoggingWrapsMultilineCommand(t *testing.T) {
	cmd := "printf 'a\\n'\nprintf 'b\\n'\n"
	got := withAttackLogging(cmd)
	want := "(printf 'a\\n'\nprintf 'b\\n') 2>&1 | tee -a /logs/attacker.log | tee -a /proc/1/fd/1"
	if got != want {
		t.Fatalf("withAttackLogging() = %q, want %q", got, want)
	}
}

func TestWithAttackLoggingEmptyCommand(t *testing.T) {
	if got := withAttackLogging(" \n\t "); got != "" {
		t.Fatalf("withAttackLogging() = %q, want empty string", got)
	}
}
