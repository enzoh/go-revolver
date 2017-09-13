/**
 * File        : ping_test.go
 * Description : Unit tests.
 * Copyright   : Copyright (c) 2017 DFINITY Stiftung. All rights reserved.
 * Maintainer  : Enzo Haussecker <enzo@string.technology>
 * Stability   : Experimental
 */

package p2p

import (
	"testing"

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

}
