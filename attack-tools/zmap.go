package atktools

import (
	"math/rand"
	"strconv"

	"gitlab.ilabt.imec.be/lpdhooge/containercap/scenario"
)

type Zmap struct {
	scenarioType                 scenario.ScenarioType
	weight                       int
	parts, probeType, percentage []string
	probes, timeout              uint32
}

func NewZmap() Zmap {
	zmap := Zmap{weight: 10, scenarioType: scenario.Scanning}
	zmap.probeType = []string{"tcp_synscan", "icmp_echoscan", "icmp_echo_time", "udp", "ntp", "upnp"}
	zmap.percentage = []string{"0.1%", "0.01%", "0.001%", "0.0001%"}
	return zmap
}

func (zmap Zmap) BuildAtkCommand() []string {
	zmap.parts = []string{"zmap", "-T 50", "-G 02:42:ac:11:00:08"}
	zmap.parts = append(zmap.parts, "-M", zmap.probeType[rand.Intn(len(zmap.probeType))])
	zmap.parts = append(zmap.parts, "-n", zmap.percentage[rand.Intn(len(zmap.percentage))])
	zmap.probes = topPorts.TCP[rand.Intn(len(topPorts.TCP))]
	zmap.parts = append(zmap.parts, "-p", strconv.FormatUint(uint64(zmap.probes), 10))
	zmap.timeout = uint32(30 + rand.Intn(270))
	zmap.parts = append(zmap.parts, "-t", strconv.FormatUint(uint64(zmap.timeout), 10))
	return zmap.parts
}

func (zmap Zmap) Weight() int {
	return zmap.weight
}

func (zmap Zmap) ScenarioType() scenario.ScenarioType {
	return zmap.scenarioType
}
