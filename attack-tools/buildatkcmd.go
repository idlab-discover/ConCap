package atktools

type AttackCommandBuilder interface {
	BuildAtkCommand() []string
}
