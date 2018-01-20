/**
 * File        : ping_test.go
 * Description : Unit tests.
 * Copyright   : Copyright (c) 2017-2018 DFINITY Stiftung. All rights reserved.
 * Maintainer  : Enzo Haussecker <enzo@dfinity.org>
 * Stability   : Experimental
 */

package p2p

import (
	"math"
	"testing"
	"time"

	"gx/ipfs/QmPgDWmTmuzvP7QE5zwo1TmjbJme9pmZHNujB2453jkCTr/go-libp2p-peerstore"
)

// Show that a client can ping a peer.
func TestPing(test *testing.T) {

	// Create a client.
	client1, shutdown1 := newTestClient(test)
	defer shutdown1()

	// Create a second client.
	client2, shutdown2 := newTestClient(test)
	defer shutdown2()

	// Add the second client to the peer store of the first.
	client1.peerstore.AddAddrs(
		client2.id,
		client2.host.Addrs(),
		peerstore.ProviderAddrTTL,
	)

	// Ping the second client.
	err := client1.ping(client2.id)
	if err != nil {
		test.Fatal(err)
	}

	// Verify the ping was recorded.
	latency := client1.peerstore.LatencyEWMA(client2.id)
	if latency < time.Nanosecond {
		test.Fatalf("Invalid latency: %s", latency)
	}
	timestamp, err := client1.peerstore.Get(client2.id, "LAST_PING")
	if err != nil {
		test.Fatal(err)
	}

	// Verify the recording is fresh.
	diration := math.Abs(float64(time.Since(timestamp.(time.Time)).Nanoseconds())) / 1000
	if diration > 1000 {
		test.Fatalf("Stale recording: %.0fμs", diration)
	}

}
