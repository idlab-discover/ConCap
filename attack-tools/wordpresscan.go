package atktools

import "strconv"

type WordpressScan struct {
	weight int
	parts  []string
}

func (wpsscan WordpressScan) BuildAtkCommand() []string {
	wpsscan.parts = []string{"wordpresscan", "-u", "http://127.0.0.1/wordpress", "--fuzz", "--random-agent", "--threads", strconv.Itoa(50)}
	wpsscan.weight = 1
	return wpsscan.parts
}
