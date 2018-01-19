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
	"errors"
	"fmt"
	"sync"

	"gx/ipfs/QmPgDWmTmuzvP7QE5zwo1TmjbJme9pmZHNujB2453jkCTr/go-libp2p-peerstore"
	"gx/ipfs/QmSAFA8v42u4gpJNy1tb7vW3JiiXiaYDC2b845c2RnNSJL/go-libp2p-kbucket"
	"gx/ipfs/QmSAFA8v42u4gpJNy1tb7vW3JiiXiaYDC2b845c2RnNSJL/go-libp2p-kbucket/keyspace"
	"gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"
	"gx/ipfs/QmZNkThpqfVXs9GNbexPrfBbXSLNYeKrE7jwFM2oqHbyqN/go-libp2p-protocol"
	"gx/ipfs/QmaPbCnUMBohSGo3KnxEa2bHqyJVVeEEcwtqJAYxerieBo/go-libp2p-crypto"
	"gx/ipfs/QmefgzMbKZYsmHFkLqxgaTBG9ypeEjrdWRD5WXH4j1cWDL/go-libp2p/p2p/host/basic"

	"github.com/dfinity/go-revolver/artifact"
	"github.com/dfinity/go-revolver/streamstore"
	"github.com/enzoh/go-logging"
	"github.com/hashicorp/golang-lru"
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

	// Request an artifact.
	Request(checksum [32]byte) (artifact.Artifact, error)

	// Register an artifact request handler.
	SetArtifactHandler(handler ArtifactHandler)

	// Register a commitment request handler.
	SetCommitmentHandler(handler CommitmentHandler)

	// Register a challenge request handler.
	SetChallengeHandler(handler ChallengeHandler)

	// Register a zero-knowledge proof request handler.
	SetProofHandler(handler ProofHandler)

	// Register a verification request handler.
	SetVerificationHandler(handler VerificationHandler)
}

type client struct {
	artifactCache            *lru.Cache
	artifactCacheLock        *sync.Mutex
	artifactRequests         chan artifactRequest
	challengeRequests        chan challengeRequest
	commitmentRequests       chan commitmentRequest
	config                   *Config
	context                  context.Context
	host                     *basichost.BasicHost
	id                       peer.ID
	key                      keyspace.Key
	logger                   *logging.Logger
	peerstore                peerstore.Peerstore
	proofRequests            chan proofRequest
	protocol                 protocol.ID
	receive                  chan artifact.Artifact
	send                     chan artifact.Artifact
	spammerCache             *lru.Cache
	spammerCacheLock         *sync.Mutex
	streamstore              streamstore.Streamstore
	table                    *kbucket.RoutingTable
	unsetArtifactHandler     func()
	unsetChallengeHandler    func()
	unsetCommitmentHandler   func()
	unsetHandlerLock         *sync.Mutex
	unsetProofHandler        func()
	unsetVerificationHandler func()
	verificationRequests     chan verificationRequest
	witnessCache             *lru.Cache
	witnessCacheLock         *sync.Mutex
}

// ArtifactHandler -- This type represents a function that executes when
// receiving an artifact request. The function can be registered as a callback
// using SetArtifactHandler.
type ArtifactHandler func(checksum [32]byte, response chan artifact.Artifact)

type artifactRequest struct {
	checksum [32]byte
	response chan artifact.Artifact
}

// Addresses -- List the addresses.
func (client *client) Addresses() []string {
	addrs := client.host.Addrs()
	result := make([]string, len(addrs))
	for i := range result {
		result[i] = addrs[i].String()
	}
	return result
}

// ID -- Get the ID.
func (client *client) ID() string {
	return client.id.Pretty()
}

// PeerCount -- Get the peer count.
func (client *client) PeerCount() int {
	return client.table.Size()
}

// StreamCount -- Get the stream count.
func (client *client) StreamCount() int {
	return client.streamstore.InboundSize() + client.streamstore.OutboundSize()
}

// Send -- Send an artifact.
func (client *client) Send(artifact artifact.Artifact) {
	client.send <- artifact
}

// Receive -- Receive an artifact.
func (client *client) Receive() artifact.Artifact {
	return <-client.receive
}

// Request -- Request an artifact.
func (client *client) Request(checksum [32]byte) (artifact.Artifact, error) {
	return nil, errors.New("TODO: Implement request method.")
}

// SetArtifactHandler -- Register an artifact request handler.
func (client *client) SetArtifactHandler(handler ArtifactHandler) {

	notify := make(chan struct{})

	client.unsetHandlerLock.Lock()
	client.unsetArtifactHandler()
	client.unsetArtifactHandler = func() {
		close(notify)
	}
	client.unsetHandlerLock.Unlock()

	go func() {
		for {
			select {
			case <-notify:
				return
			case request := <-client.artifactRequests:
				handler(request.checksum, request.response)
			}
		}
	}()

}

// New -- Create a client.
func (config *Config) New() (Client, func(), error) {
	return config.create()
}

func (config *Config) create() (*client, func(), error) {

	// Validate the configuration.
	err := config.validate()
	if err != nil {
		return nil, nil, err
	}

	// Create a client from the configuration.
	client := &client{}
	client.config = &Config{}
	*client.config = *config

	// Create an artifact cache.
	client.artifactCache, err = lru.New(client.config.ArtifactCacheSize)
	if err != nil {
		return nil, nil, err
	}
	client.artifactCacheLock = &sync.Mutex{}

	// Create an artifact request queue.
	client.artifactRequests = make(chan artifactRequest, client.config.ArtifactQueueSize)

	// Create a challenge request queue.
	client.challengeRequests = make(chan challengeRequest, 1)

	// Create a commitment request queue.
	client.commitmentRequests = make(chan commitmentRequest, 1)

	// Create a context.
	client.context = context.Background()

	// Create or decode the random seed.
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

	// Create a key in the Kademlia key space.
	client.key = keyspace.XORKeySpace.Key(kbucket.ConvertPeerID(client.id))

	// Create a logger.
	client.logger = logging.MustGetLogger("p2p")

	// Create a peer store.
	client.peerstore = peerstore.NewPeerstore()
	client.peerstore.AddPrivKey(client.id, secretKey)
	client.peerstore.AddPubKey(client.id, publicKey)

	// Create a zero-knowledge proof request queue.
	client.proofRequests = make(chan proofRequest, 1)

	// Create a protocol.
	client.protocol = protocol.ID(
		fmt.Sprintf(
			"/%s/%s",
			client.config.Network,
			client.config.Version,
		),
	)

	// Create the artifact queues.
	client.send = make(chan artifact.Artifact, client.config.ArtifactQueueSize)
	client.receive = make(chan artifact.Artifact, client.config.ArtifactQueueSize)

	// Create a spammer cache.
	client.spammerCache, err = lru.New(client.config.SpammerCacheSize)
	if err != nil {
		return nil, nil, err
	}
	client.spammerCacheLock = &sync.Mutex{}

	// Create a stream store.
	client.streamstore = streamstore.New(
		client.config.StreamstoreInboundCapacity,
		client.config.StreamstoreOutboundCapacity,
		client.config.StreamstoreQueueSize,
		client.probeLatency,
	)

	// Create a routing table.
	client.table = kbucket.NewRoutingTable(
		client.config.KBucketSize,
		client.key.Bytes,
		client.config.LatencyTolerance,
		client.peerstore,
	)

	// Add the client to its routing table.
	client.table.Update(client.id)

	// Initialize the handler deregistration functions.
	client.unsetArtifactHandler = func() {}
	client.unsetCommitmentHandler = func() {}
	client.unsetChallengeHandler = func() {}
	client.unsetHandlerLock = &sync.Mutex{}
	client.unsetProofHandler = func() {}
	client.unsetVerificationHandler = func() {}

	// Create a verification request queue.
	client.verificationRequests = make(chan verificationRequest, 1)

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
		client.unsetArtifactHandler()
		client.unsetCommitmentHandler()
		client.unsetChallengeHandler()
		client.unsetProofHandler()
		client.unsetVerificationHandler()
		shutdown()
	}, nil

}
