package atktools

import "math/rand"

type Amass struct {
	parts, wordlists []string
}

func NewAmass() Amass {
	amass := Amass{}
	amass.wordlists = []string{
		"/usr/share/amass/wordlists/all.txt",
		"/usr/share/amass/wordlists/asnlist.txt",
		"/usr/share/amass/wordlists/bitquark_subdomains_top100K.txt",
		"/usr/share/amass/wordlists/deepmagic.com_top500prefixes.txt",
		"/usr/share/amass/wordlists/deepmagic.com_top50kprefixes.txt",
		"/usr/share/amass/wordlists/fierce_hostlist.txt",
		"/usr/share/amass/wordlists/jhaddix_all.txt",
		"/usr/share/amass/wordlists/namelist.txt",
		"/usr/share/amass/wordlists/nameservers.txt",
		"/usr/share/amass/wordlists/sorted_knock_dnsrecon_fierce_recon-ng.txt",
		"/usr/share/amass/wordlists/subdomains-top1mil-110000.txt",
		"/usr/share/amass/wordlists/subdomains-top1mil-20000.txt",
		"/usr/share/amass/wordlists/subdomains-top1mil-5000.txt",
		"/usr/share/amass/wordlists/subdomains.lst",
		"/usr/share/amass/wordlists/user_agents.txt"}
	return amass
}

func (amass Amass) BuildAtkCommand() []string {
	amass.parts = []string{"amass -active -brute -include-unresolvable", "-ip", "-min-for-recursive 1", "-src", "-p 80,443,8080,8081,8443"}

	if rand.Float32() < 0.5 {
		amass.parts = append(amass.parts, "-w", amass.wordlists[rand.Intn(len(amass.wordlists))])
	}
	amass.parts = append(amass.parts, "-d", "ugent.be,example.com,kuleuven.be")
	return amass.parts
}
