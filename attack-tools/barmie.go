package atktools

type Barmie struct {
	weight int
	parts  []string
}

func (barmie Barmie) BuildAtkCommand() []string {
	barmie.parts = []string{"barmie", "-attack", "127.0.0.1"}
	barmie.weight = 1
	return barmie.parts
}
