/**
 * File        : streamstore_test.go
 * Description : Unit tests.
 * Copyright   : Copyright (c) 2017-2018 DFINITY Stiftung. All rights reserved.
 * Maintainer  : Enzo Haussecker <enzo@dfinity.org>
 * Stability   : Stable
 */

package streamstore

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	mathrand "math/rand"
	"testing"
	"time"

	"gx/ipfs/QmNa31VPzC561NWwRsJLE7nGYZYuuD2QfpK2b1q9BK54J1/go-libp2p-net"
	"gx/ipfs/QmPgDWmTmuzvP7QE5zwo1TmjbJme9pmZHNujB2453jkCTr/go-libp2p-peerstore"
	"gx/ipfs/QmXY77cVe7rVRQXZZQRioukUM7aRW3BTcAgJe12MCtb3Ji/go-multiaddr"
	"gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"
	"gx/ipfs/QmaPbCnUMBohSGo3KnxEa2bHqyJVVeEEcwtqJAYxerieBo/go-libp2p-crypto"
	"gx/ipfs/QmdQFrFnPrKRQtpeHKjZ3cVNwxmGKKS2TvhJTuN9C9yduh/go-libp2p-swarm"
	"gx/ipfs/QmefgzMbKZYsmHFkLqxgaTBG9ypeEjrdWRD5WXH4j1cWDL/go-libp2p/p2p/host/basic"

	"github.com/enzoh/go-logging"
)

const QUEUE_SIZE = 10

type client struct {
	address     multiaddr.Multiaddr
	ctx         context.Context
	host        *basichost.BasicHost
	id          peer.ID
	peerstore   peerstore.Peerstore
	queue       chan [32]byte
	streamstore Streamstore
}

// A latency prob function that returns random latency
func randomProbe(_ peer.ID) (time.Duration, error) {
	return time.Duration(mathrand.Intn(1000) * int(time.Millisecond)), nil
}

// Create a client.
func new(test *testing.T, port uint16) *client {

	var err error
	client := &client{}

	// Create an address.
	client.address, err = multiaddr.NewMultiaddr(
		fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", port),
	)
	if err != nil {
		test.Fatal(err)
	}

	// Create a context.
	client.ctx = context.Background()

	// Create a key pair.
	secret, pubkey, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		test.Fatal(err)
	}

	// Create an identity.
	client.id, err = peer.IDFromPublicKey(pubkey)
	if err != nil {
		test.Fatal(err)
	}

	// Create a peer store.
	client.peerstore = peerstore.NewPeerstore()
	client.peerstore.AddAddr(client.id, client.address, peerstore.PermanentAddrTTL)
	client.peerstore.AddPrivKey(client.id, secret)
	client.peerstore.AddPubKey(client.id, pubkey)

	// Create an artifact queue.
	client.queue = make(chan [32]byte, QUEUE_SIZE)

	// Create a stream store.
	client.streamstore = New(1, 1, QUEUE_SIZE, randomProbe)

	// Create a network.
	network, err := swarm.NewNetwork(
		client.ctx,
		[]multiaddr.Multiaddr{client.address},
		client.id,
		client.peerstore,
		nil,
	)
	if err != nil {
		test.Fatal(err)
	}

	// Create a service host.
	client.host, err = basichost.NewHost(client.ctx, network, &basichost.HostOpts{})
	if err != nil {
		test.Fatal(err)
	}

	// Register a service with the service host.
	client.host.SetStreamHandler("/test", func(stream net.Stream) {
		pid := stream.Conn().RemotePeer()
		if client.streamstore.Add(pid, stream, false) {
			go client.read(stream)
		} else {
			test.Fatal("Cannot add", pid, "to stream store")
		}
	})

	// Return the client.
	return client

}

// Read artifacts from a stream.
func (client *client) read(stream net.Stream) {
	pid := stream.Conn().RemotePeer()
	var artifact [32]byte
	for {
		_, err := io.ReadFull(stream, artifact[:])
		if err != nil {
			break
		}
		select {
		case client.queue <- artifact:
		default:
		}
	}
	client.streamstore.Remove(pid)
}

// Demonstrate the transaction lifecycle of the stream store.
func TestStreamstore(test *testing.T) {

	// Set the log level to debug.
	logging.SetLevel(logging.DEBUG, "streamstore")

	// Create a client.
	client1 := new(test, 12345)
	defer client1.host.Close()

	// Create a target peer.
	client2 := new(test, 23456)
	defer client2.host.Close()

	// Add the target peer to the peer store of the client.
	client1.peerstore.AddAddr(
		client2.id,
		client2.address,
		peerstore.TempAddrTTL,
	)

	// Connect to the target peer.
	stream, err := client1.host.NewStream(client1.ctx, client2.id, "/test")
	if err != nil {
		test.Fatal(err)
	}
	if client1.streamstore.Add(client2.id, stream, true) {
		go client1.read(stream)
	} else {
		test.Fatal("Cannot add", client2.id, "to stream store")
	}

	// Generate random artifacts.
	artifacts := make([][32]byte, QUEUE_SIZE)
	for i := range artifacts {
		_, err := rand.Reader.Read(artifacts[i][:])
		if err != nil {
			test.Fatal(err)
		}
	}

	// Send the artifacts to the target peer.
	errors := make([]map[peer.ID]chan error, QUEUE_SIZE)
	for i := range artifacts {
		artifact := artifacts[i]
		errors[i] = client1.streamstore.Apply(
			func(pid peer.ID, writer io.Writer) error {
				_, err := writer.Write(artifact[:])
				return err
			},
			nil,
		)
	}

	// Verify that the artifacts were sent to the target peer.
	for i := range errors {
		select {
		case err := <-errors[i][client2.id]:
			if err != nil {
				test.Fatal(err)
			}
		case <-time.After(time.Second):
			test.Fatal("Timeout!")
		}
	}

	// Verify that the artifacts were received by the target peer.
	for i := range artifacts {
		select {
		case artifact := <-client2.queue:
			if artifacts[i] != artifact {
				test.Fatal("Corrupt artifact!")
			}
		case <-time.After(time.Second):
			test.Fatal("Missing artifact!")
		}
	}

}
