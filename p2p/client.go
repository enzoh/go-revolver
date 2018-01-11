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
	"os"
	"sync"
	"time"

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

type Config struct {
	AnalyticsInterval      time.Duration
	AnalyticsURL           string
	AnalyticsUserData      string
	ArtifactCacheSize      int
	ArtifactChunkSize      uint32
	ArtifactMaxBufferSize  uint32
	ArtifactQueueSize      int
	ClusterID              int
	DisableAnalytics       bool
	DisableBroadcast       bool
	DisableNATPortMap      bool
	DisablePeerDiscovery   bool
	DisableStreamDiscovery bool
	IP                     string
	KBucketSize            int
	LatencyTolerance       time.Duration
	LogFile                *os.File
	LogLevel               string
	NATMonitorInterval     time.Duration
	NATMonitorTimeout      time.Duration
	Network                string
	PingBufferSize         uint32
	Port                   uint16
	ProcessID              int
	ProofBufferSize        uint32
	RandomSeed             string
	SampleMaxBufferSize    uint32
	SampleSize             int
	SeedNodes              []string
	StreamstoreCapacity    int
	StreamstoreQueueSize   int
	Timeout                time.Duration
	VerificationBufferSize uint32
	Version                string
	WitnessCacheSize       int
}

// DefaultConfig -- Get the default configuration parameters.
func DefaultConfig() (*Config, error) {

	entropy := make([]byte, 32)
	_, err := rand.Read(entropy)
	if err != nil {
		return nil, err
	}

	return &Config{
		AnalyticsInterval:      time.Minute,
		AnalyticsURL:           "https://analytics.dfinity.build/report",
		AnalyticsUserData:      "",
		ArtifactCacheSize:      65536,
		ArtifactChunkSize:      65536,
		ArtifactMaxBufferSize:  16777216,
		ArtifactQueueSize:      8192,
		ClusterID:              0,
		DisableAnalytics:       false,
		DisableBroadcast:       false,
		DisableNATPortMap:      false,
		DisablePeerDiscovery:   false,
		DisableStreamDiscovery: false,
		IP:                     "0.0.0.0",
		KBucketSize:            16,
		LatencyTolerance:       time.Minute,
		LogFile:                os.Stdout,
		LogLevel:               "INFO",
		NATMonitorInterval:     time.Second,
		NATMonitorTimeout:      time.Minute,
		Network:                "revolver",
		PingBufferSize:         32,
		Port:                   0,
		ProcessID:              0,
		ProofBufferSize:        0,
		RandomSeed:             hex.EncodeToString(entropy),
		SampleMaxBufferSize:    8192,
		SampleSize:             4,
		SeedNodes:              nil,
		StreamstoreCapacity:    8,
		StreamstoreQueueSize:   8192,
		Timeout:                peerstore.TempAddrTTL,
		VerificationBufferSize: 0,
		Version:                "0.1.0",
		WitnessCacheSize:       65536,
	}, nil

}

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

	// Register a zero-knowledge proof request handler.
	SetProofHandler(handler ProofHandler)

	// Register a verification request handler.
	SetVerificationHandler(handler VerificationHandler)
}

// ArtifactHandler -- This type represents a function that executes when
// receiving an artifact request. The function can be registered as a callback
// using SetArtifactHandler.
type ArtifactHandler func(checksum [32]byte, response chan artifact.Artifact)

// ProofHandler -- This type represents a function that executes when receiving
// a zero-knowledge proof request. The function can be registered as a callback
// using SetProofHandler.
type ProofHandler func(data []byte, response chan []byte)

// VerificationHandler -- This type represents a function that executes when
// receiving a verification request. The function can be registered as a
// callback using SetVerificationHandler.
type VerificationHandler func(data []byte, response chan bool)

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
	return client.streamstore.Size()
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

	client.unsetArtifactHandlerLock.Lock()
	client.unsetArtifactHandler()
	client.unsetArtifactHandler = func() {
		close(notify)
	}
	client.unsetArtifactHandlerLock.Unlock()

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

// SetProofHandler -- Register a zero-knowledge proof request handler.
func (client *client) SetProofHandler(handler ProofHandler) {

	notify := make(chan struct{})

	client.unsetProofHandlerLock.Lock()
	client.unsetProofHandler()
	client.unsetProofHandler = func() {
		close(notify)
	}
	client.unsetProofHandlerLock.Unlock()

	go func() {
		for {
			select {
			case <-notify:
				return
			case request := <-client.proofRequests:
				handler(request.data, request.response)
			}
		}
	}()

}

// SetVerificationHandler -- Register a verification request handler.
func (client *client) SetVerificationHandler(handler VerificationHandler) {

	notify := make(chan struct{})

	client.unsetVerificationHandlerLock.Lock()
	client.unsetVerificationHandler()
	client.unsetVerificationHandler = func() {
		close(notify)
	}
	client.unsetVerificationHandlerLock.Unlock()

	go func() {
		for {
			select {
			case <-notify:
				return
			case request := <-client.verificationRequests:
				handler(request.data, request.response)
			}
		}
	}()

}

// New -- Create a client.
func (config *Config) New() (Client, func(), error) {
	return config.create()
}

type client struct {
	artifactCache                *lru.Cache
	artifactCacheLock            *sync.Mutex
	artifactRequests             chan struct {checksum [32]byte; response chan artifact.Artifact}
	config                       *Config
	context                      context.Context
	host                         *basichost.BasicHost
	id                           peer.ID
	key                          keyspace.Key
	logger                       *logging.Logger
	peerstore                    peerstore.Peerstore
	proofRequests                chan struct {data []byte; response chan []byte}
	protocol                     protocol.ID
	receive                      chan artifact.Artifact
	send                         chan artifact.Artifact
	streamstore                  streamstore.Streamstore
	table                        *kbucket.RoutingTable
	unsetArtifactHandler         func()
	unsetArtifactHandlerLock     *sync.Mutex
	unsetProofHandler            func()
	unsetProofHandlerLock        *sync.Mutex
	unsetVerificationHandler     func()
	unsetVerificationHandlerLock *sync.Mutex
	verificationRequests         chan struct {data []byte; response chan bool}
	witnessCache                 *lru.Cache
	witnessCacheLock             *sync.Mutex
}

func (config *Config) create() (*client, func(), error) {

	var err error
	client := &client{}

	// Copy the configuration parameters.
	client.config = &Config{}
	*client.config = *config

	// Check for any misconfigurations.
	if client.config.AnalyticsInterval <= 0 {
		return nil, nil, errors.New("Analytics iteration interval must be a positive time duration.")
	}
	if client.config.ArtifactCacheSize <= 0 {
		return nil, nil, errors.New("Artifact cache size must be a positive integer.")
	}
	if client.config.ArtifactChunkSize <= 45 {
		return nil, nil, errors.New("Artifact chunk size must be an integer greater than forty-five.")
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
	if client.config.NATMonitorInterval <= 0 {
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
	client.artifactCache, err = lru.New(client.config.ArtifactCacheSize)
	if err != nil {
		return nil, nil, err
	}
	client.artifactCacheLock = &sync.Mutex{}

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
	backend := logging.NewLogBackend(client.config.LogFile, "", 0)
	formatter := "\033[0;37m%{time:15:04:05.000} %{color}%{level} \033[0;34m[%{module}] \033[0m%{message} \033[0;37m%{shortfile}\033[0m"
	logging.SetBackend(logging.AddModuleLevel(backend))
	logging.SetFormatter(logging.MustStringFormatter(formatter))

	// Set the log level.
	switch client.config.LogLevel {
	case "CRITICAL":
		logging.SetLevel(logging.CRITICAL, "p2p")
	case "ERROR":
		logging.SetLevel(logging.ERROR, "p2p")
	case "WARNING":
		logging.SetLevel(logging.WARNING, "p2p")
	case "NOTICE":
		logging.SetLevel(logging.NOTICE, "p2p")
	case "INFO":
		logging.SetLevel(logging.INFO, "p2p")
	default:
		logging.SetLevel(logging.DEBUG, "p2p")
	}
	logging.SetLevel(logging.INFO, "streamstore")

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

	// Create the broadcast queues.
	client.send = make(chan artifact.Artifact, client.config.ArtifactQueueSize)
	client.receive = make(chan artifact.Artifact, client.config.ArtifactQueueSize)

	// Create the request queues.
	client.artifactRequests = make(chan struct {checksum [32]byte; response chan artifact.Artifact}, client.config.ArtifactQueueSize)
	client.proofRequests = make(chan struct {data []byte; response chan []byte}, 1)
	client.verificationRequests = make(chan struct {data []byte; response chan bool}, 1)

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
	client.witnessCache, err = lru.New(client.config.WitnessCacheSize)
	if err != nil {
		return nil, nil, err
	}
	client.witnessCacheLock = &sync.Mutex{}

	// Initialize the handle deregistration functions.
	client.unsetArtifactHandler = func() {}
	client.unsetArtifactHandlerLock = &sync.Mutex{}
	client.unsetProofHandler = func() {}
	client.unsetProofHandlerLock = &sync.Mutex{}
	client.unsetVerificationHandler = func() {}
	client.unsetVerificationHandlerLock = &sync.Mutex{}

	// Start the client.
	shutdown, err := client.bootstrap()
	if err != nil {
		return nil, nil, err
	}

	// Ready for action.
	return client, shutdown, nil

}
