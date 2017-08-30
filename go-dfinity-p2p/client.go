/**
 * File        : client.go
 * Description : High-level client interface.
 * Copyright   : Copyright (c) 2017 DFINITY Stiftung. All rights reserved.
 * Maintainer  : Enzo Haussecker <enzo@string.technology>
 * Stability   : Experimental
 */

package p2p

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/hashicorp/golang-lru"

	"gx/ipfs/QmNXRzmFYCiQtjJKiA2zxThcRVrvQGYWbj1C9cyc6tp4tP/go-dfinity-streamstore"
	"gx/ipfs/QmPgDWmTmuzvP7QE5zwo1TmjbJme9pmZHNujB2453jkCTr/go-libp2p-peerstore"
	"gx/ipfs/QmQvJiADDe7JR4m968MwXobTCCzUqQkP87aRHe29MEBGHV/go-logging"
	"gx/ipfs/QmVU26BGUSt3LkbWmoH7dP16mNz5VVRg4hDmWZBHAkq97w/go-libp2p-kbucket"
	"gx/ipfs/QmVU26BGUSt3LkbWmoH7dP16mNz5VVRg4hDmWZBHAkq97w/go-libp2p-kbucket/keyspace"
	"gx/ipfs/QmXY77cVe7rVRQXZZQRioukUM7aRW3BTcAgJe12MCtb3Ji/go-multiaddr"
	"gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"
	"gx/ipfs/QmZNkThpqfVXs9GNbexPrfBbXSLNYeKrE7jwFM2oqHbyqN/go-libp2p-protocol"
	"gx/ipfs/QmZyngpQxUGyx1T2bzEcst6YzERkvVwDzBMbsSQF4f1smE/go-libp2p/p2p/host/basic"
	"gx/ipfs/QmaPbCnUMBohSGo3KnxEa2bHqyJVVeEEcwtqJAYxerieBo/go-libp2p-crypto"
)

type Config struct {
	AnalyticsIterationInterval  time.Duration
	AnalyticsURL                string
	ArtifactCacheSize           int
	ArtifactChunkSize           uint32
	ArtifactMaxBufferSize       uint32
	ArtifactQueueSize           int
	ClusterID                   int
	DisableAnalytics            bool
	DisableBroadcast            bool
	DisableNATPortMap           bool
	DisablePeerDiscovery        bool
	DisableStreamDiscovery      bool
	KBucketSize                 int
	LatencyTolerance            time.Duration
	ListenIP                    string
	ListenPort                  uint16
	LogLevel                    logging.Level
	NATMonitorIterationInterval time.Duration
	NATMonitorTimeout           time.Duration
	Network                     protocol.ID
	PingBufferSize              uint32
	ProcessID                   int
	RandomSeed                  string
	SampleMaxBufferSize         uint32
	SampleSize                  int
	SeedNodes                   []multiaddr.Multiaddr
	StreamstoreCapacity         int
	StreamstoreQueueSize        int
	Timeout                     time.Duration
	Version                     protocol.ID
	WitnessCacheSize            int
}

// Get the default configuration parameters of the client.
func DefaultConfig() (*Config, error) {
	return &Config{
		AnalyticsIterationInterval:  time.Minute,
		AnalyticsURL:                "https://analytics.dfinity.build/report",
		ArtifactCacheSize:           8192,
		ArtifactChunkSize:           1048576,
		ArtifactMaxBufferSize:       16777216,
		ArtifactQueueSize:           1024,
		ClusterID:                   0,
		DisableAnalytics:            false,
		DisableBroadcast:            false,
		DisableNATPortMap:           false,
		DisablePeerDiscovery:        false,
		DisableStreamDiscovery:      false,
		KBucketSize:                 16,
		LatencyTolerance:            time.Minute,
		ListenIP:                    "0.0.0.0",
		ListenPort:                  4000,
		LogLevel:                    logging.DEBUG,
		NATMonitorIterationInterval: time.Second,
		NATMonitorTimeout:           time.Minute,
		Network:                     "DFINITY",
		PingBufferSize:              32,
		ProcessID:                   0,
		RandomSeed:                  "0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF",
		SampleMaxBufferSize:         8192,
		SampleSize:                  4,
		SeedNodes:                   nil,
		StreamstoreCapacity:         8,
		StreamstoreQueueSize:        4096,
		Timeout:                     peerstore.TempAddrTTL,
		Version:                     "0.0.2",
		WitnessCacheSize:            1024,
	}, nil
}

type Client interface {

	// Get the addresses of a client.
	Addresses() []multiaddr.Multiaddr

	// Get the ID of a client.
	ID() string

	// Get the peer count of a client.
	PeerCount() int

	// Get the artifact receive queue of a client.
	Receive() chan []byte

	// Get the artifact send queue of a client.
	Send() chan []byte

	// Get the stream count of a client.
	StreamCount() int
}

type client struct {
	artifacts     *lru.Cache
	artifactsLock *sync.Mutex
	config        *Config
	context       context.Context
	host          *basichost.BasicHost
	id            peer.ID
	key           keyspace.Key
	logger        *logging.Logger
	peerstore     peerstore.Peerstore
	protocol      protocol.ID
	receive       chan []byte
	send          chan []byte
	streamstore   streamstore.Streamstore
	table         *kbucket.RoutingTable
	witnesses     *lru.Cache
	witnessesLock *sync.Mutex
}

// Get the addresses of a client.
func (client *client) Addresses() []multiaddr.Multiaddr {
	return client.host.Addrs()
}

// Get the ID of a client.
func (client *client) ID() string {
	return client.id.Pretty()
}

// Get the peer count of a client.
func (client *client) PeerCount() int {
	return client.table.Size()
}

// Get the artifact receive queue of a client.
func (client *client) Receive() chan []byte {
	return client.receive
}

// Get the artifact send queue of a client.
func (client *client) Send() chan []byte {
	return client.send
}

// Get the stream count of a client.
func (client *client) StreamCount() int {
	return client.streamstore.Size()
}

// Create a client.
func (config *Config) New() (Client, func(), error) {
	return config.new()
}

func (config *Config) new() (*client, func(), error) {

	var err error
	client := &client{}

	// Copy the configuration parameters.
	client.config = &Config{}
	*client.config = *config

	// Check for any misconfigurations.
	if client.config.ArtifactCacheSize <= 0 {
		return nil, nil, errors.New("Artifact cache size must be a positive integer.")
	}
	if client.config.ArtifactChunkSize <= 44 {
		return nil, nil, errors.New("Artifact chunk size must be an integer greater than forty-four.")
	}
	if client.config.ArtifactMaxBufferSize <= 0 {
		return nil, nil, errors.New("Artifact max buffer size must be a positive integer.")
	}
	if client.config.ArtifactQueueSize <= 0 {
		return nil, nil, errors.New("Artifact queue size must be a positive integer.")
	}
	if client.config.KBucketSize <= 0 {
		return nil, nil, errors.New("K-bucket size must be a positive integer.")
	}
	if client.config.LatencyTolerance <= 0 {
		return nil, nil, errors.New("Latency tolerance must be a positive time duration.")
	}
	if client.config.NATMonitorIterationInterval <= 0 {
		return nil, nil, errors.New("NAT monitor iteration interval must be a positive time duration.")
	}
	if client.config.NATMonitorTimeout <= 0 {
		return nil, nil, errors.New("NAT monitor timeout must be a positive time duration.")
	}
	if client.config.PingBufferSize <= 0 {
		return nil, nil, errors.New("Ping buffer size must be a positive integer.")
	}
	if len(client.config.RandomSeed) != 64 {
		return nil, nil, errors.New("Random seed must be 32 bytes.")
	}
	if client.config.SampleMaxBufferSize <= 0 {
		return nil, nil, errors.New("Peer sample max buffer size must be a positive integer.")
	}
	if client.config.SampleSize <= 0 {
		return nil, nil, errors.New("Peer sample size must be a positive integer.")
	}
	if client.config.StreamstoreCapacity <= 0 {
		return nil, nil, errors.New("Stream store capacity must be a positive integer.")
	}
	if client.config.StreamstoreQueueSize <= 0 {
		return nil, nil, errors.New("Stream store transaction queue size must be a positive integer.")
	}
	if client.config.Timeout <= 0 {
		return nil, nil, errors.New("Request timeout must be a positive time duration.")
	}
	if client.config.WitnessCacheSize <= 0 {
		return nil, nil, errors.New("Witness cache size must be a positive integer.")
	}

	// Create an artifact cache.
	client.artifacts, err = lru.New(client.config.ArtifactCacheSize)
	if err != nil {
		return nil, nil, err
	}
	client.artifactsLock = &sync.Mutex{}

	// Create a context.
	client.context = context.Background()

	// Decode the random seed.
	seed, err := hex.DecodeString(client.config.RandomSeed)
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

	// Create a key in the XOR key space.
	client.key = keyspace.XORKeySpace.Key(kbucket.ConvertPeerID(client.id))

	// Create a logger.
	client.logger = logging.MustGetLogger("p2p")
	logging.SetLevel(client.config.LogLevel, "p2p")

	// Create a peer store.
	client.peerstore = peerstore.NewPeerstore()
	client.peerstore.AddPrivKey(client.id, secretKey)
	client.peerstore.AddPubKey(client.id, publicKey)

	// Create a protocol.
	client.protocol = protocol.ID(
		fmt.Sprintf(
			"/%s/%s",
			client.config.Network,
			client.config.Version,
		),
	)

	// Create the artifact queues.
	client.send = make(chan []byte, client.config.ArtifactQueueSize)
	client.receive = make(chan []byte, client.config.ArtifactQueueSize)

	// Create a stream store.
	client.streamstore = streamstore.New(
		client.config.StreamstoreCapacity,
		client.config.StreamstoreQueueSize,
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

	// Create a witness cache.
	client.witnesses, err = lru.New(client.config.WitnessCacheSize)
	if err != nil {
		return nil, nil, err
	}
	client.witnessesLock = &sync.Mutex{}

	// Start the client.
	shutdown, err := client.bootstrap()
	if err != nil {
		return nil, nil, err
	}

	// Ready for action.
	return client, shutdown, nil

}
