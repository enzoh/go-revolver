/**
 * File        : process_test.go
 * Description : Unit tests.
 * Copyright   : Copyright (c) 2017 DFINITY Stiftung. All rights reserved.
 * Maintainer  : Enzo Haussecker <enzo@string.technology>
 * Stability   : Experimental
 */

package p2p

import (
	"encoding/hex"
	"math/rand"
	"testing"
	"time"

	"github.com/dfinity/go-dfinity-p2p/artifact"
	"github.com/libp2p/go-libp2p-peerstore"
)

// Show that a client cannot receive duplicate artifacts.
func TestDuplication(test *testing.T) {

	const N = 1000

	// Create a client.
	client1, shutdown1 := newTestClient(test, 25432)
	defer shutdown1()

	// Create a second client.
	client2, shutdown2 := newTestClient(test, 31498)
	defer shutdown2()

	// Create a third client.
	client3, shutdown3 := newTestClient(test, 30880)
	defer shutdown3()

	// Create a fourth client.
	client4, shutdown4 := newTestClient(test, 46012)
	defer shutdown4()

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

	// Add the fourth client to the peer store of the third.
	client3.peerstore.AddAddrs(
		client4.id,
		client4.host.Addrs(),
		peerstore.ProviderAddrTTL,
	)

	// Add the first client to the peer store of the fourth.
	client4.peerstore.AddAddrs(
		client1.id,
		client1.host.Addrs(),
		peerstore.ProviderAddrTTL,
	)

	// Update the routing table of each client.
	client1.table.Update(client4.id)
	client2.table.Update(client1.id)
	client3.table.Update(client2.id)
	client4.table.Update(client3.id)

	// Pair the first and second client.
	success1, err := client1.pair(client2.id)
	if err != nil || !success1 {
		test.Fatal(err)
	}

	// Pair the second and third client.
	success2, err := client2.pair(client3.id)
	if err != nil || !success2 {
		test.Fatal(err)
	}

	// Pair the third and fourth client.
	success3, err := client3.pair(client4.id)
	if err != nil || !success3 {
		test.Fatal(err)
	}

	// Pair the fourth and first client.
	success4, err := client4.pair(client1.id)
	if err != nil || !success4 {
		test.Fatal(err)
	}

	// Create notifications to shutdown the artifact forwarding loops.
	notify2 := make(chan struct{})
	notify4 := make(chan struct{})
	defer func() {
		close(notify2)
		close(notify4)
	}()

	// Forward artifacts from the first client to the third client.
	go func() {
	Forwarding:
		for {
			select {
			case <-notify2:
				break Forwarding
			case artifact := <-client2.Receive():
				client2.Send() <- artifact
			}
		}
	}()

	// Forward artifacts from the first client to the third client.
	go func() {
	Forwarding:
		for {
			select {
			case <-notify4:
				break Forwarding
			case artifact := <-client4.Receive():
				client4.Send() <- artifact
			}
		}
	}()

	// Send artifacts to the second and fourth client.
	go func() {
		for i := 0; i < N; i++ {
			time.Sleep(25 * time.Microsecond)
			dataOut := make([]byte, 32)
			rand.Read(dataOut)
			artifactOut := artifact.FromBytes(dataOut)
			client1.Send() <- artifactOut
		}
	}()

	// Receive artifacts from the second and fourth client.
	cache := make(map[string]struct{})
	for i := 0; i < N; i++ {

		select {

		// Wait for the third client to receive an artifact.
		case artifactIn := <-client3.Receive():

			// Consume the artifact.
			dataIn, err := artifact.ToBytes(artifactIn)
			if err != nil {
				test.Fatal(err)
			}

			// Verify that the artifact is not a duplicate.
			key := hex.EncodeToString(dataIn)
			_, exists := cache[key]
			if exists {
				test.Fatal("Duplicate artifact!")
			}
			cache[key] = struct{}{}

		case <-time.After(time.Second):
			test.Fatal("Missing artifact!")

		}

	}

}
