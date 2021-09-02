// Package capengi (capture-engine) is a start to integrate packet capture from within the code, rather than by relying on invoking tcpdump inside the pods / containers
// This functionality has not yet been integrated
package capengi

import (
	"fmt"
	"log"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/afpacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

// AfpacketCap creates a tap with AFpacket
// AFpacket is a module built on top of the Linux memory-mapped packet socket interface (AF_PACKET).
// This interface provides direct access to copies of raw packets received on Linux network devices in an adjuct ring buffer.
// It supports both passive and inline modes.
func AfpacketCap(iface string) {
	tpack, err := afpacket.NewTPacket(afpacket.OptInterface(iface))
	if err != nil {
		panic(err)
	} else {
		packetSource := gopacket.NewPacketSource(tpack, layers.LinkTypeEthernet)
		for packet := range packetSource.Packets() {
			handlePacket(packet)
		}
	}
	tpack.Close()
}

// LibpcapCap creates a tap built on libpcap
// Package pcap allows users of gopacket to read packets off the wire or from pcap files.
func LibpcapCap(iface string) {
	inactive, err := pcap.NewInactiveHandle(iface)
	if err != nil {
		log.Fatal(err)
	}
	defer inactive.CleanUp()

	// Call various functions on inactive to set it up the way you'd like:
	if err = inactive.SetTimeout(time.Minute); err != nil {
		log.Fatal(err)

		// Finally, create the actual handle by calling Activate:
		handle, err := inactive.Activate() // after this, inactive is no longer valid
		if err != nil {
			log.Fatal(err)
		} else {
			packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
			for packet := range packetSource.Packets() {
				handlePacket(packet) // Do something with a packet here.
			}
		}
		defer handle.Close()
	}
}

func handlePacket(packet gopacket.Packet) {
	fmt.Println(packet)
}
