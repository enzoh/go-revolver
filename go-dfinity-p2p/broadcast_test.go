/**
 * File        : broadcast_test.go
 * Description : Unit tests.
 * Copyright   : Copyright (c) 2017 DFINITY Stiftung. All rights reserved.
 * Maintainer  : Enzo Haussecker <enzo@string.technology>
 * Stability   : Experimental
 */

package p2p

import (
	"bytes"
	"math/rand"
	"testing"
	"time"

	"gx/ipfs/QmPgDWmTmuzvP7QE5zwo1TmjbJme9pmZHNujB2453jkCTr/go-libp2p-peerstore"
	"gx/ipfs/QmXfZLaUjKGKUCs4ikNQsAtFWXXx5xZ7tp4s1GoVysNNBj/go-dfinity-artifact"
)

// Show that a client can broadcast artifacts to its peers.
func TestBroadcast(test *testing.T) {

	// Create a client.
	client1, shutdown1 := newTestClient(test, 64583)
	defer shutdown1()

	// Create a second client.
	client2, shutdown2 := newTestClient(test, 59505)
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

		// Generate a random artifact.
		dataOut := make([]byte, rand.Intn(int(client1.config.ArtifactMaxBufferSize)))
		_, err = rand.Read(dataOut)
		if err != nil {
			test.Fatal(err)
		}
		artifactOut := artifact.FromBytes(dataOut)

		// Send the artifact to the second client.
		client1.Send() <- artifactOut

		select {

		// Wait for the second client to receive the artifact.
		case artifactIn := <-client2.Receive():

			// Consume the artifact.
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
