/**
 * File        : client.go
 * Description : High-level client interface.
 * Copyright   : Copyright (c) 2017-2018 DFINITY Stiftung. All rights reserved.
 * Maintainer  : Enzo Haussecker <enzo@dfinity.org>
 * Stability   : Experimental
 */

package p2p

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"sync"

	"gx/ipfs/QmNa31VPzC561NWwRsJLE7nGYZYuuD2QfpK2b1q9BK54J1/go-libp2p-net"
	"gx/ipfs/QmPgDWmTmuzvP7QE5zwo1TmjbJme9pmZHNujB2453jkCTr/go-libp2p-peerstore"
	"gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"
	"gx/ipfs/QmZNkThpqfVXs9GNbexPrfBbXSLNYeKrE7jwFM2oqHbyqN/go-libp2p-protocol"
	"gx/ipfs/QmaPbCnUMBohSGo3KnxEa2bHqyJVVeEEcwtqJAYxerieBo/go-libp2p-crypto"
	"gx/ipfs/QmefgzMbKZYsmHFkLqxgaTBG9ypeEjrdWRD5WXH4j1cWDL/go-libp2p/p2p/host/basic"

	"github.com/dfinity/go-logging"
	"github.com/dfinity/go-lru"
	"github.com/dfinity/go-revolver/artifact"
	"github.com/dfinity/go-revolver/routing"
	"github.com/dfinity/go-revolver/streamstore"
)

type Client interface {

	// List the addresses.
	Addresses() []string

	// Get the ID.
	ID() string

	// Get the peer count.
	PeerCount() int

	// Get the stream count.
	StreamCount() int

	// Send an artifact.
	Send(artifact artifact.Artifact)

	// Receive an artifact.
	Receive() artifact.Artifact

	// Request an ad hoc artifact exchange.
	Sync(command []byte) (net.Stream, error)
}

type client struct {
	artifactCache            *lru.Cache
	artifactCacheLock        *sync.Mutex
	challengeRequests        chan challengeRequest
	commitmentRequests       chan commitmentRequest
	config                   *Config
	context                  context.Context
	host                     *basichost.BasicHost
	id                       peer.ID
	logger                   *logging.Logger
	peerstore                peerstore.Peerstore
	proofRequests            chan proofRequest
	protocol                 protocol.ID
	receive                  chan artifact.Artifact
	send                     chan artifact.Artifact
	streamstore              streamstore.Streamstore
	syncRequests             chan syncRequest
	table                    routing.RoutingTable
	unsetChallengeHandler    func()
	unsetCommitmentHandler   func()
	unsetProofHandler        func()
	unsetSyncHandler         func()
	unsetVerificationHandler func()
	verificationRequests     chan verificationRequest
	witnessCache             *lru.Cache
	witnessCacheLock         *sync.Mutex
}

// List the addresses.
func (client *client) Addresses() []string {
	addrs := client.host.Addrs()
	result := make([]string, len(addrs))
	for i := range result {
		result[i] = addrs[i].String()
	}
	return result
}

// Get the ID.
func (client *client) ID() string {
	return client.id.Pretty()
}

// Get the peer count.
func (client *client) PeerCount() int {
	return client.table.Size()
}

// Get the stream count.
func (client *client) StreamCount() int {
	return client.streamstore.Size()
}

// Send an artifact.
func (client *client) Send(artifact artifact.Artifact) {
	client.send <- artifact
}

// Receive an artifact.
func (client *client) Receive() artifact.Artifact {
	return <-client.receive
}

// Create a client.
func (config *Config) New() (Client, func(), error) {
	return config.create()
}

func (config *Config) create() (*client, func(), error) {

	// Check if the configuration contains any invalid parameters.
	err := config.validate()
	if err != nil {
		return nil, nil, err
	}

	// Copy the configuration to a new client.
	client := &client{}
	client.config = &Config{}
	*client.config = *config

	// Create a context.
	client.context = context.Background()

	// Create a logger.
	client.logger = logging.MustGetLogger("p2p")

	// Define a protocol prefix.
	client.protocol = protocol.ID(
		"/" + client.config.Network + "/" + client.config.Version,
	)

	// Create a random seed.
	seed := make([]byte, 32)
	if len(client.config.RandomSeed) == 0 {
		_, err = rand.Read(seed)
	} else {
		seed, err = hex.DecodeString(client.config.RandomSeed)
	}
	if err != nil {
		return nil, nil, err
	}

	// Create an Ed25519 key pair from the random seed.
	secretKey, publicKey, err := crypto.GenerateEd25519Key(
		bytes.NewReader(seed),
	)
	if err != nil {
		return nil, nil, err
	}

	// Create an identity from the public key.
	client.id, err = peer.IDFromPublicKey(publicKey)
	if err != nil {
		return nil, nil, err
	}

	// Create a peer store.
	client.peerstore = peerstore.NewPeerstore()
	client.peerstore.AddPrivKey(client.id, secretKey)
	client.peerstore.AddPubKey(client.id, publicKey)

	// Create a stream store.
	client.streamstore = streamstore.New(
		client.config.StreamstoreCapacity,
		client.config.StreamstoreQueueSize,
	)

	// Create a routing table.
	client.table = routing.NewRoutingTable(
		client.config.KBucketSize,
		client.config.LatencyTolerance,
		client.id,
		client.peerstore,
		client.streamstore,
	)

	// Create the authentication request queues.
	client.commitmentRequests = make(chan commitmentRequest, 1)
	client.challengeRequests = make(chan challengeRequest, 1)
	client.proofRequests = make(chan proofRequest, 1)
	client.verificationRequests = make(chan verificationRequest, 1)

	// Register the authentication request handlers.
	client.setCommitmentHandler(client.config.CommitmentHandler)
	client.setChallengeHandler(client.config.ChallengeHandler)
	client.setProofHandler(client.config.ProofHandler)
	client.setVerificationHandler(client.config.VerificationHandler)

	// Create the artifact queues.
	client.send = make(chan artifact.Artifact, client.config.ArtifactQueueSize)
	client.receive = make(chan artifact.Artifact, client.config.ArtifactQueueSize)

	// Create the artifact sync request queue.
	client.syncRequests = make(chan syncRequest, 1)

	// Register the artifact sync request handler.
	client.setSyncHandler(client.config.SyncHandler)

	// Create an artifact cache.
	client.artifactCache, err = lru.New(client.config.ArtifactCacheSize)
	if err != nil {
		return nil, nil, err
	}
	client.artifactCacheLock = &sync.Mutex{}

	// Create a witness cache.
	client.witnessCache, err = lru.New(client.config.WitnessCacheSize)
	if err != nil {
		return nil, nil, err
	}
	client.witnessCacheLock = &sync.Mutex{}

	// Start the client.
	shutdown, err := client.bootstrap()
	if err != nil {
		return nil, nil, err
	}

	// Ready for action!
	return client, func() {
		shutdown()
		client.unsetCommitmentHandler()
		client.unsetChallengeHandler()
		client.unsetProofHandler()
		client.unsetSyncHandler()
		client.unsetVerificationHandler()
	}, nil

}
