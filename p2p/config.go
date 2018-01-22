/**
 * File        : config.go
 * Description : Client configuration module.
 * Copyright   : Copyright (c) 2017-2018 DFINITY Stiftung. All rights reserved.
 * Maintainer  : Enzo Haussecker <enzo@dfinity.org>
 * Stability   : Experimental
 */

package p2p

import (
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/url"
	"time"

	"gx/ipfs/QmXY77cVe7rVRQXZZQRioukUM7aRW3BTcAgJe12MCtb3Ji/go-multiaddr"
)

// This type provides all available options to configure a client.
type Config struct {
	AnalyticsInterval       time.Duration
	AnalyticsURL            string
	AnalyticsUserData       string
	ArtifactCacheSize       int
	ArtifactChunkSize       uint32
	ArtifactMaxBufferSize   uint32
	ArtifactQueueSize       int
	ChallengeHandler        ChallengeHandler
	ChallengeMaxBufferSize  uint32
	ClusterID               int
	CommitmentHandler       CommitmentHandler
	CommitmentMaxBufferSize uint32
	DisableAnalytics        bool
	DisableBroadcast        bool
	DisableNATPortMap       bool
	DisablePeerDiscovery    bool
	DisableStreamDiscovery  bool
	IP                      string
	KBucketSize             int
	LatencyTolerance        time.Duration
	NATMonitorInterval      time.Duration
	NATMonitorTimeout       time.Duration
	Network                 string
	Port                    uint16
	ProcessID               int
	ProofHandler            ProofHandler
	ProofMaxBufferSize      uint32
	RandomSeed              string
	SampleMaxBufferSize     uint32
	SampleSize              int
	SeedNodes               []string
	SpammerCacheSize        int
	StreamstoreCapacity     int
	StreamstoreQueueSize    int
	Timeout                 time.Duration
	VerificationHandler     VerificationHandler
	Version                 string
	WitnessCacheSize        int
}

// Get the default configuration parameters.
func DefaultConfig() *Config {
	return &Config{
		AnalyticsInterval:       time.Minute,
		AnalyticsURL:            "https://analytics.dfinity.build/report",
		AnalyticsUserData:       "",
		ArtifactCacheSize:       65536,
		ArtifactChunkSize:       65536,
		ArtifactMaxBufferSize:   8388608,
		ArtifactQueueSize:       8,
		ChallengeHandler:        DefaultChallengeHandler,
		ChallengeMaxBufferSize:  32,
		ClusterID:               0,
		CommitmentHandler:       DefaultCommitmentHandler,
		CommitmentMaxBufferSize: 32,
		DisableAnalytics:        false,
		DisableBroadcast:        false,
		DisableNATPortMap:       false,
		DisablePeerDiscovery:    false,
		DisableStreamDiscovery:  false,
		IP:                   "0.0.0.0",
		KBucketSize:          16,
		LatencyTolerance:     time.Minute,
		NATMonitorInterval:   time.Second,
		NATMonitorTimeout:    time.Minute,
		Network:              "revolver",
		Port:                 0,
		ProcessID:            0,
		ProofHandler:         DefaultProofHandler,
		ProofMaxBufferSize:   32,
		RandomSeed:           "",
		SampleMaxBufferSize:  8192,
		SampleSize:           4,
		SeedNodes:            nil,
		SpammerCacheSize:     16384,
		StreamstoreCapacity:  8,
		StreamstoreQueueSize: 8192,
		Timeout:              10 * time.Second,
		VerificationHandler:  DefaultVerificationHandler,
		Version:              "0.1.0",
		WitnessCacheSize:     65536,
	}
}

func (config *Config) validate() error {

	// The analytics interval must be a positive time duration.
	if config.AnalyticsInterval <= 0 {
		return fmt.Errorf("Invalid analytics interval: %d", config.AnalyticsInterval)
	}

	// The analytics URL must be parsable.
	_, err := url.Parse(config.AnalyticsURL)
	if err != nil {
		return fmt.Errorf("Invalid analytics URL: %s", config.AnalyticsURL)
	}

	// The artifact cache size must be a positive integer.
	if config.ArtifactCacheSize <= 0 {
		return fmt.Errorf("Invalid artifact cache size: %d", config.ArtifactCacheSize)
	}

	// The artifact chunk size must be a non-zero unsigned 32-bit integer.
	if config.ArtifactChunkSize == 0 {
		return errors.New("Invalid artifact chunk size: 0")
	}

	// The artifact max buffer size must be a non-zero unsigned 32-bit integer.
	if config.ArtifactMaxBufferSize == 0 {
		return errors.New("Invalid artifact max buffer size: 0")
	}

	// The artifact queue size must be a positive integer.
	if config.ArtifactQueueSize <= 0 {
		return fmt.Errorf("Invalid artifact queue size: %d", config.ArtifactQueueSize)
	}

	// The IP address must be parsable.
	if net.ParseIP(config.IP) == nil {
		return fmt.Errorf("Invalid IP address: %s", config.IP)
	}

	// The Kademlia bucket size must be a positive integer.
	if config.KBucketSize <= 0 {
		return fmt.Errorf("Invalid Kademlia bucket size: %d", config.KBucketSize)
	}

	// The latency tolerance must be a positive time duration.
	if config.LatencyTolerance <= 0 {
		return fmt.Errorf("Invalid latency tolerance: %d", config.LatencyTolerance)
	}

	// The NAT monitor interval must be a positive time duration.
	if config.NATMonitorInterval <= 0 {
		return fmt.Errorf("Invalid NAT monitor interval: %d", config.NATMonitorInterval)
	}

	// The NAT monitor timeout must be a positive time duration.
	if config.NATMonitorTimeout <= 0 {
		return fmt.Errorf("Invalid NAT monitor timeout: %d", config.NATMonitorTimeout)
	}

	// The random seed must be a zero or 32-byte hex-encoded string.
	_, err = hex.DecodeString(config.RandomSeed)
	if len(config.RandomSeed) != 0 && len(config.RandomSeed) != 64 || err != nil {
		return fmt.Errorf("Invalid random seed: %s", config.RandomSeed)
	}

	// The peer sample max buffer size must be a non-zero unsigned 32-bit integer.
	if config.SampleMaxBufferSize == 0 {
		return errors.New("Invalid peer sample max buffer size: 0")
	}

	// The peer sample size must be a positive integer.
	if config.SampleSize <= 0 {
		return fmt.Errorf("Invalid peer sample size: %d", config.SampleSize)
	}

	// The seed nodes must be parsable.
	for i := range config.SeedNodes {
		_, err = multiaddr.NewMultiaddr(config.SeedNodes[i])
		if err != nil {
			return fmt.Errorf("Invalid seed node: %s", config.SeedNodes[i])
		}
	}

	// The stream store capacity must be a positive integer.
	if config.StreamstoreCapacity <= 0 {
		return fmt.Errorf("Invalid stream store capacity: %d", config.StreamstoreCapacity)
	}

	// The stream store transaction queue size must be a positive integer.
	if config.StreamstoreQueueSize <= 0 {
		return fmt.Errorf("Invalid stream store transaction queue size: %d", config.StreamstoreQueueSize)
	}

	// The stream timeout must be a positive time duration.
	if config.Timeout <= 0 {
		return fmt.Errorf("Invalid stream timeout: %d", config.Timeout)
	}

	// The witness cache size must be a positive integer.
	if config.WitnessCacheSize <= 0 {
		return fmt.Errorf("Invalid witness cache size: %d", config.WitnessCacheSize)
	}

	return nil

}
