package atktools

type Shareenum struct {
	Weight int
	parts  []string
}

func (shareenum Shareenum) BuildAtkCommand() []string {
	// null sessions for now, but password forcing should be a thing later and when that happens, these are the options
	// -u USER	Username, otherwise go anonymous. If using a domain, it should be in the format of DOMAIN\user.
	// -p PASS	Password, otherwise go anonymous. This can be a NTLM has in the format LMHASH:NTLMHASH. If so, we'll pass the hash.
	shareenum.parts = []string{"shareenum", "-o -", "localhost"}
	shareenum.Weight = 1
	return shareenum.parts
}
