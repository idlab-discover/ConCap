package atk

import "strings"

type AttackCommandBuilder interface {
	BuildAtkCommand() []string
}

func JoinParts(strs []string) string {
	var sb strings.Builder
	for _, str := range strs {
		sb.WriteString(str)
	}
	return sb.String()
}
