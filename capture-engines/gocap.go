// Package gocap provides the ability to capture traffic between
// attacker, target, and support pods within a Kubernetes framework
// using Go programming capabilities.
package capengi

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"text/template"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/google/gopacket/pcapgo"

	kubeapi "gitlab.ilabt.imec.be/lpdhooge/containercap/kube-api-interaction"
	"gitlab.ilabt.imec.be/lpdhooge/containercap/ledger"
	"gitlab.ilabt.imec.be/lpdhooge/containercap/scenario"
)

var (
	device       string = "weave"
	snapshot_len int32  = 65536 // or 2048
	promiscuous  bool   = true
	err          error
	timout       time.Duration = 1 * time.Second
	handle       *pcap.Handle
	packetCount  int = 0
)

type IPAddress struct {
	AttackAddress  net.IP
	TargetAddress  net.IP
	SupportAddress []net.IP
}

// CreatePcap creates an instance of pcapgo to capture traffic on the weave interface.
// It writes to the given output file and applies a filter based on the included pods.
// The filter takes in AttackAddress, TargetAddress, and SupportAddress.
// PcapCreator is a helper function that creates a pcap file from packets captured on a device
// based on a given scenario and pod specifications. It opens an output file, writes the pcap header,
// sets up a packet source and filter, and writes packets to the file until the scenario state changes.
//
// Parameters:
// - scn: A pointer to the scenario object that defines the filter criteria and state for capturing packets.
// - outputPath: A string that specifies the path and name of the output pcap file.
// - pods: A variadic parameter of PodSpec objects that contains information about each pod involved in the scenario.
func PcapCreator(scn *scenario.Scenario, outputPath string, pods ...kubeapi.PodSpec) {
	// Open output pcap file and write header
	f, err := os.Create(outputPath)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	// Make a new writer to write packets to a pcap file
	w := pcapgo.NewWriter(f)
	w.WriteFileHeader(uint32(snapshot_len), layers.LinkTypeEthernet)

	// Open the device for capturing
	handle, err = pcap.OpenLive(device, snapshot_len, promiscuous, timout)
	if err != nil {
		fmt.Println("Error opening device " + device + ": " + err.Error())
		os.Exit(1)
	}
	defer handle.Close()

	// Get the filter string from our helper function
	filterStr, err := generateFilterString(scn, pods)
	if err != nil {
		fmt.Println("Error generating filter string: ", err)
		return
	}

	// Set the device filter based on the scenario's info
	SetFilter(handle, filterStr)

	// Start processing packets
	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	fmt.Println("Starting capturing...")

	// Create channel to signal completion
	done := make(chan struct{})

	// Use a buffered channel for better performance
	packetBuffer := make(chan gopacket.Packet, 1000)

	go func() {
		// Loop while waiting for packets, adding them to the buffer
		for packet := range packetSource.Packets() {
			packetBuffer <- packet
		}
	}()

	// Check scenario state every 200 milliseconds
	ticker := time.Tick(200 * time.Millisecond)

	for {
		select {
		case packet := <-packetBuffer:
			// updating packet metadata and writing it down
			packet.Metadata().CaptureLength = len(packet.Data())
			packet.Metadata().Length = len(packet.Data())

			// Write the packet to the pcap file
			err = w.WritePacket(packet.Metadata().CaptureInfo, packet.Data())

			if err != nil {
				fmt.Println("could not write packet to pcap file: " + err.Error())
			}

			packetCount++

		case <-ticker:
			if state := ledger.GetScenarioState(scn.UUID.String()); state == string(ledger.COMPLETED) {
				fmt.Println("GoCapture Completed...")
				close(done)
				close(packetBuffer)

				return
			}
		}

		select {
		// if done channel returns signal that processing was completed earlier
		case <-done:
			// Completed earlier
			return

		default:
			// No action
		}
	}
}

// CreatePcap creates an instance of pcapgo to capture traffic on the weave interface.
// It writes to the given output file and applies a filter based on the included pods.
// The filter takes in AttackAddress, TargetAddress, and SupportAddress.
func PcapCreator2(scn *scenario.Scenario, outputPath string, attackpod kubeapi.PodSpec, targetpod kubeapi.PodSpec, supportpods ...kubeapi.PodSpec) {

	// Open output pcap file and write header
	f, err := os.Create(outputPath)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	// Make a new writer to write packets to a pcap file
	w := pcapgo.NewWriter(f)
	w.WriteFileHeader(uint32(snapshot_len), layers.LinkTypeEthernet)

	// Open the device for capturing
	handle, err = pcap.OpenLive(device, snapshot_len, promiscuous, timout)
	if err != nil {
		fmt.Println("Error opening device " + device + ": " + err.Error())
		os.Exit(1)
	}
	defer handle.Close()

	var pods []kubeapi.PodSpec
	pods = append(pods, attackpod, targetpod)
	if len(supportpods) == 0 {
		fmt.Println("No support pods")
	} else if len(supportpods) > 0 {
		fmt.Println("There are support pods")
		pods = append(pods, supportpods...)
	}
	// Get the filter string from our helper function => Needs to be changed
	filterStr, err := generateFilterString(scn, pods)
	if err != nil {
		fmt.Println("Error generating filter string: ", err)
		return
	}

	// Set the device filter based on the scenario's info
	SetFilter(handle, filterStr)

	// Start processing packets
	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	fmt.Println("Starting capturing...")

	// Create channel to signal completion
	done := make(chan struct{})

	// Use a buffered channel for better performance
	packetBuffer := make(chan gopacket.Packet, 1000)

	go func() {
		// Loop while waiting for packets, adding them to the buffer
		for packet := range packetSource.Packets() {
			packetBuffer <- packet
		}
	}()

	// Check scenario state every 200 milliseconds
	ticker := time.Tick(200 * time.Millisecond)

	for {
		select {
		case packet := <-packetBuffer:
			// updating packet metadata and writing it down
			packet.Metadata().CaptureLength = len(packet.Data())
			packet.Metadata().Length = len(packet.Data())

			// Write the packet to the pcap file
			err = w.WritePacket(packet.Metadata().CaptureInfo, packet.Data())

			if err != nil {
				fmt.Println("could not write packet to pcap file: " + err.Error())
			}

			packetCount++

		case <-ticker:
			if state := ledger.GetScenarioState(scn.UUID.String()); state == string(ledger.COMPLETED) {
				fmt.Println("GoCapture Completed...")
				close(done)
				close(packetBuffer)

				return
			}
		}

		select {
		// if done channel returns signal that processing was completed earlier
		case <-done:
			// Completed earlier
			return

		default:
			// No action
		}
	}
}

// SetFilter is a local function that takes a handle to capture packets in PcapCreator and a filter string
// as input. It sets the specified filter on the handle using the SetBPFFilter method from the pcap package.
// More info on setting filters here: https://biot.com/capstats/bpf.html
//
// Parameters:
//   - handle: A pointer to the pcap handle used to capture packets.
//   - filter: A string containing the filter to be set on the handle.
func SetFilter(handle *pcap.Handle, filter string) {
	// Set filter
	err = handle.SetBPFFilter(filter)
	if err != nil {
		fmt.Println("Error setting filter (gocap.go): " + err.Error())
		os.Exit(1)
	}

}

// Helper function to display all available devices.
func DisplayAllDevices() {
	// Find all devices
	devices, err := pcap.FindAllDevs()
	if err != nil {
		fmt.Println("Error finding devices (gocap.go): " + err.Error())
	}

	// Print device information
	for _, device := range devices {
		fmt.Println("Name:" + device.Name + "\nDescription: " + device.Description + "\n Device Addresses: ")
		for _, address := range device.Addresses {
			fmt.Println("\tIP Address: " + string(address.IP) + "\n\tSubnet mask: " + string(address.Netmask))
		}
	}
}

// generateFilterString is a helper function that takes a scenario and a slice of Kubernetes Pod specifications as input.
// It extracts IP addresses of relevant pods and generates a filter string to be used in packet capture for the given scenario.
// The function initializes an IPAddress struct with the IP addresses of the first and second pods as AttackAddress and TargetAddress,
// and appends the IP addresses of any additional pods to the SupportAddress slice of the struct.
// It then uses a template to generate the filter string based on the extracted IP addresses.
//
// Parameters:
//   - scn: A pointer to the scenario.Scenario struct that specifies the capture engine and the filter to be used for packet capture.
//   - pods: A slice of kubeapi.PodSpec structs that specify the IP addresses of relevant pods.
//
// Returns:
//   - A string containing the filter to be set on the pcap handle for packet capture, generated using the provided scenario and pod specifications.
//   - An error if there are less than 2 pods or if there are issues with parsing the template or executing the filter.
func generateFilterString(scn *scenario.Scenario, pods []kubeapi.PodSpec) (string, error) {

	var podIPs IPAddress

	if len(pods) < 2 {
		return "", fmt.Errorf("invalid number of pods: got %d, expected 2 or more", len(pods))
	}
	// initialize the struct with AttackAddress and TargetAddress fields using IP addresses from the first and second pods respectively
	podIPs = IPAddress{AttackAddress: net.ParseIP(pods[0].PodIP), TargetAddress: net.ParseIP(pods[1].PodIP)}

	// if there are more than two pods start a loop to add the support addresses to the podIPs.SupportAddress slice
	if len(pods) > 2 {
		// Initialize the SupportAddress field with the capacity equal to len(pods) - 2
		podIPs.SupportAddress = make([]net.IP, 0, len(pods)-2)
		for _, p := range pods[2:] {
			podIP := net.ParseIP(p.PodIP)
			if podIP != nil {
				podIPs.SupportAddress = append(podIPs.SupportAddress, podIP)
			}
		}
	}

	buf := new(bytes.Buffer)

	// get the filter string and parse it
	tmpl, err := template.New("test").Parse(scn.CaptureEngine.Filter)
	if err != nil {
		return "", fmt.Errorf("could not parse filter: %v", err)
	}

	// execute the template and store it in the buffer
	err = tmpl.Execute(buf, podIPs)
	if err != nil {
		return "", fmt.Errorf("could not execute filter template: %v", err)
	}

	// convert buffer to string and return result along with nil error.
	return buf.String(), nil
}
