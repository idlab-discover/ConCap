package scenarios

import "strings"

const attackerLogPipe = "/tmp/attacker.log.pipe"

func withAttackLogging(cmd string) string {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return ""
	}

	return "pipe=" + attackerLogPipe +
		"; rm -f \"$pipe\" && mkfifo \"$pipe\" || exit 1" +
		"; tee -a /logs/attacker.log /proc/1/fd/1 < \"$pipe\" & tee_pid=$!" +
		"; (" + cmd + ") > \"$pipe\" 2>&1" +
		"; status=$?" +
		"; wait \"$tee_pid\"" +
		"; rm -f \"$pipe\"" +
		"; exit \"$status\""
}
