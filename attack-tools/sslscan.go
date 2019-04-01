package atktools

type Sslscan struct {
	parts []string
}

func (sslscan Sslscan) BuildAtkCommand() []string {
	sslscan.parts = []string{"sslscan --no-failed --renegotiation --bugs 127.0.0.1:443"}
	return sslscan.parts
}
