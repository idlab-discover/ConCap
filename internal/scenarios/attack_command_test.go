package scenarios

import "testing"

func TestWithAttackLoggingTrimsTrailingWhitespace(t *testing.T) {
	got := withAttackLogging("echo hi\n")
	want := "pipe=/tmp/attacker.log.pipe; rm -f \"$pipe\" && mkfifo \"$pipe\" || exit 1; tee -a /logs/attacker.log /proc/1/fd/1 < \"$pipe\" & tee_pid=$!; (echo hi) > \"$pipe\" 2>&1; status=$?; wait \"$tee_pid\"; rm -f \"$pipe\"; exit \"$status\""
	if got != want {
		t.Fatalf("withAttackLogging() = %q, want %q", got, want)
	}
}

func TestWithAttackLoggingWrapsMultilineCommand(t *testing.T) {
	cmd := "printf 'a\\n'\nprintf 'b\\n'\n"
	got := withAttackLogging(cmd)
	want := "pipe=/tmp/attacker.log.pipe; rm -f \"$pipe\" && mkfifo \"$pipe\" || exit 1; tee -a /logs/attacker.log /proc/1/fd/1 < \"$pipe\" & tee_pid=$!; (printf 'a\\n'\nprintf 'b\\n') > \"$pipe\" 2>&1; status=$?; wait \"$tee_pid\"; rm -f \"$pipe\"; exit \"$status\""
	if got != want {
		t.Fatalf("withAttackLogging() = %q, want %q", got, want)
	}
}

func TestWithAttackLoggingEmptyCommand(t *testing.T) {
	if got := withAttackLogging(" \n\t "); got != "" {
		t.Fatalf("withAttackLogging() = %q, want empty string", got)
	}
}
