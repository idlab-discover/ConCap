package Command

import (
	"strings"

	atktools "gitlab.ilabt.imec.be/lpdhooge/containercap/attack-tools"
	"gitlab.ilabt.imec.be/lpdhooge/containercap/scenario"
)

// GenerateAttackCommand is a helper function that returns a string that contains the command to launch an attack
// on a given target using a selected attacker tool. It calls the BuildAtkCommand method of the attacker tool to get
// the command parts, and then replaces any occurrences of localhost or 127.0.0.1 with the target IP address in the
// command parts. It then joins the command parts together into a single string and returns it.
//
// Parameters:
//   - scn: A pointer to the Scenario struct that contains the attacker tool and target information.
//
// Returns:
//   - A string containing the attack command for the given scenario.
func GenerateAttackCommand(scn *scenario.Scenario) string {

	// Call BuildAtkCommand method of the selected attacker category to get command string
	atk := (*(atktools.SelectAttacker(scn.Attacker.Category, scn.Attacker.Name))).BuildAtkCommand()

	// Extract target IP address from scenario struct and store into target variable
	target := "{{.TargetAddress}}"

	// Loop through the atk command parts array slice
	for i, part := range atk {

		// If localhost is present inside part then replace it with target using strings.Replace()
		if strings.Contains(part, "localhost") {
			atk[i] = strings.Replace(part, "localhost", target, -1)
			break

			// If https://127.0.0.1, http://127.0.0.1, or 127.0.0.1 are present inside part then replace it with target using strings.Replace()
		} else if strings.Contains(part, "https://127.0.0.1") {
			atk[i] = strings.Replace(part, "https://127.0.0.1", target, -1)
			break
		} else if strings.Contains(part, "http://127.0.0.1") {
			atk[i] = strings.Replace(part, "http://127.0.0.1", target, -1)
			break
		} else if strings.Contains(part, "127.0.0.1") {
			atk[i] = strings.Replace(part, "127.0.0.1", target, -1)
			break
		}
	}

	// Return a string by concatenating the modified parts of Nmap command together using Join function
	atkC := strings.Join(atk, " ")
	return atkC
}
