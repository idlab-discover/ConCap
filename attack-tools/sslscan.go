package atktools

type Sslscan struct {
	Weight int
	parts  []string
}

func (sslscan Sslscan) BuildAtkCommand() []string {
	sslscan.parts = []string{"sslscan --no-failed --renegotiation --bugs 127.0.0.1:443"}
	sslscan.Weight = 1
	return sslscan.parts
}
