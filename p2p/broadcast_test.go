/**
 * File        : broadcast_test.go
 * Description : Unit tests.
 * Copyright   : Copyright (c) 2017-2018 DFINITY Stiftung. All rights reserved.
 * Maintainer  : Enzo Haussecker <enzo@dfinity.org>
 * Stability   : Experimental
 */

package p2p

import (
	"bytes"
	"math/rand"
	"testing"
	"time"

	"gx/ipfs/QmPgDWmTmuzvP7QE5zwo1TmjbJme9pmZHNujB2453jkCTr/go-libp2p-peerstore"

	"github.com/dfinity/go-revolver/artifact"
)

// Show that a client can broadcast artifacts to its peers.
func TestBroadcast(test *testing.T) {

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

	// Add the second client to the routing table of the first.
	client2.table.Update(client1.id)

	// Pair the first and second client.
	success, err := client1.pair(client2.id)
	if err != nil || !success {
		test.Fatal(err)
	}

	// Begin the broadcast.
	for i := 0; i < 10; i++ {

		// Generate a random byte slice.
		dataOut := make([]byte, rand.Intn(int(client1.config.ArtifactMaxBufferSize)))
		_, err = rand.Read(dataOut)
		if err != nil {
			test.Fatal(err)
		}

		// Create an artifact from the byte slice.
		compress := rand.Intn(2) > 0
		artifactOut, err := artifact.FromBytes(dataOut, compress)
		if err != nil {
			test.Fatal(err)
		}

		// Send the artifact to the second client.
		client1.send <- artifactOut

		select {

		// Wait for the second client to receive the artifact.
		case artifactIn := <-client2.receive:

			// Create a byte slice from the artifact.
			dataIn, err := artifact.ToBytes(artifactIn)
			if err != nil {
				test.Fatal(err)
			}

			// Verify that the data sent and received is the same.
			if !bytes.Equal(dataOut, dataIn) {
				test.Fatal("Corrupt artifact!")
			}

		case <-time.After(time.Second):
			test.Fatal("Missing artifact!")

		}

	}

}
