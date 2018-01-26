/**
 * File        : discover_test.go
 * Description : Benchmark.
 * Copyright   : Copyright (c) 2017-2018 DFINITY Stiftung. All rights reserved.
 * Maintainer  : Enzo Haussecker <enzo@dfinity.org>
 * Stability   : Experimental
 */

package p2p

import (
	"testing"
	"time"
)

const discover_test_N = 25

// Benchmark how long it takes for clients to discover each other.
func BenchmarkDiscovery(benchmark *testing.B) {

	// Create a custom configuration.
	config := DefaultConfig()
	config.DisableAnalytics = true
	config.DisableBroadcast = true
	config.DisableNATPortMap = true
	config.DisableStreamDiscovery = true
	config.IP = "127.0.0.1"

	// Create a set of notification channels.
	setup := make(chan []string, discover_test_N)
	ready := make(chan struct{}, discover_test_N)

	// Start the seed node.
	go newDiscoveryClient(config, setup, ready)

	// Get the addresses of the seed node.
	var addresses []string
	select {
	case addresses = <-setup:
	case <-time.After(time.Second):
		benchmark.Fatal("Seed node failed to initialize within one second")
	}

	// Update the configuration.
	config.SeedNodes = addresses

	// Start the timer.
	benchmark.StartTimer()

	// Start the test network.
	for j := 1; j < discover_test_N; j++ {
		go newDiscoveryClient(config, setup, ready)
	}

	// Wait for clients to discover each other.
	done := make(chan struct{}, 1)
	go func() {
		for j := 0; j < discover_test_N; j++ {
			<-ready
		}
		done <- struct{}{}
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		benchmark.Fatal("Nodes failed to discover each other within ten seconds")
	}

	// Stop the timer.
	benchmark.StopTimer()

}

func newDiscoveryClient(config *Config, setup chan []string, ready chan struct{}) {

	// Instantiate the client.
	client, err := config.create()
	if err != nil {
		client.logger.Error(err)
		return
	}
	defer client.Close()

	// Relay the addresses of the client.
	var addresses []string
	for _, addr := range client.Addresses() {
		addresses = append(addresses, addr+"/ipfs/"+client.ID())
	}
	setup <- addresses

	// Wait for the client to discover others in its network.
	for {
		if float64(client.PeerCount()) >= float64(discover_test_N-1)*0.75 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	// Relay the status of the client.
	ready <- struct{}{}

	// Hang.
	time.Sleep(10 * time.Second)

}
