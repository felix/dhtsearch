package dht

import (
	"fmt"
	"net"
	"testing"

	"src.userspace.com.au/dhtsearch/models"
)

func TestPriorityQueue(t *testing.T) {
	id := "d1c5676ae7ac98e8b19f63565905105e3c4c37a2"

	tests := []string{
		"d1c5676ae7ac98e8b19f63565905105e3c4c37b9",
		"d1c5676ae7ac98e8b19f63565905105e3c4c37a9",
		"d1c5676ae7ac98e8b19f63565905105e3c4c37a4",
		"d1c5676ae7ac98e8b19f63565905105e3c4c37a3", // distance of 159
	}

	ih, err := models.InfohashFromString(id)
	if err != nil {
		t.Errorf("failed to create infohash: %s\n", err)
	}

	pq, err := newRoutingTable(*ih, 3)
	if err != nil {
		t.Errorf("failed to create kTable: %s\n", err)
	}

	for i, idt := range tests {
		iht, err := models.InfohashFromString(idt)
		if err != nil {
			t.Errorf("failed to create infohash: %s\n", err)
		}
		addr, _ := net.ResolveUDPAddr("udp", fmt.Sprintf("0.0.0.0:%d", i))
		pq.add(&remoteNode{id: *iht, addr: addr})
	}

	if len(pq.items) != len(pq.addresses) {
		t.Errorf("items and addresses out of sync")
	}

	first := pq.items[0].value.id
	if first.String() != "d1c5676ae7ac98e8b19f63565905105e3c4c37a3" {
		t.Errorf("first is %s with distance %d\n", first, ih.Distance(first))
	}
}
