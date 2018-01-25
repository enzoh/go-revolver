/**
 * File        : sample_test.go
 * Description : Unit tests.
 * Copyright   : Copyright (c) 2017-2018 DFINITY Stiftung. All rights reserved.
 * Maintainer  : Enzo Haussecker <enzo@dfinity.org>
 * Stability   : Experimental
 */

package p2p

import (
	"testing"

	"gx/ipfs/QmPgDWmTmuzvP7QE5zwo1TmjbJme9pmZHNujB2453jkCTr/go-libp2p-peerstore"
)

// Show that a client can receive a random sample of peers from the routing
// table of a peer.
func TestSample(test *testing.T) {

	// Create a client.
	client1 := newTestClient(test)
	defer client1.Close()

	// Create a second client.
	client2 := newTestClient(test)
	defer client2.Close()

	// Create a third client.
	client3 := newTestClient(test)
	defer client3.Close()

	// Add the second client to the peer store of the first.
	client1.peerstore.AddAddrs(
		client2.id,
		client2.host.Addrs(),
		peerstore.ProviderAddrTTL,
	)

	// Add the third client to the peer store of the second.
	client2.peerstore.AddAddrs(
		client3.id,
		client3.host.Addrs(),
		peerstore.ProviderAddrTTL,
	)

	// Authorize the first client.
	client2.peerstore.Put(client1.id, "AUTHORIZED", true)

	// Add the third client to the routing table of the second.
	client2.table.Add(client3.id)

	// Request peers from the second client.
	sample, err := client1.sample(client2.id)
	if err != nil {
		test.Fatal(err)
	}

	// Verify that the first client learned the contact info of the third.
	exists := false
	for i := range sample {
		exists = exists || sample[i].ID == client3.id
	}
	if !exists {
		test.Fatal("Missing contact info!")
	}

}
