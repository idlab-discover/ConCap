package scenarios

import "strings"

func withAttackLogging(cmd string) string {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return ""
	}

	return "(" + cmd + ") 2>&1 | tee -a /logs/attacker.log | tee -a /proc/1/fd/1"
}
