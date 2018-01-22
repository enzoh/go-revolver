/**
 * File        : discover_test.go
 * Description : Unit test.
 * Copyright   : Copyright (c) 2017-2018 DFINITY Stiftung. All rights reserved.
 * Maintainer  : Enzo Haussecker <enzo@dfinity.org>
 * Stability   : Experimental
 */

package p2p

import (
	"testing"
	"time"
)

const test_N = 32

// Show that clients can discover each other.
func TestDiscovery(test *testing.T) {

	setup := make(chan []string, test_N)
	ready := make(chan struct{}, test_N)

	config := DefaultConfig()
	config.DisableAnalytics = true
	config.DisableBroadcast = true
	config.DisableNATPortMap = true
	config.DisableStreamDiscovery = true
	config.IP = "127.0.0.1"

	go newDiscoveryClient(config, setup, ready)

	var addresses []string
	select {
	case addresses = <-setup:
	case <-time.After(time.Second):
		test.Fatal("Seed node failed to initialize within one second")
	}
	config.SeedNodes = addresses

	for i := 1; i < test_N; i++ {
		go newDiscoveryClient(config, setup, ready)
	}

	done := make(chan struct{}, 1)

	go func() {
		for i := 0; i < test_N; i++ {
			<-ready
		}
		done <- struct{}{}
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		test.Fatal("Nodes failed to discover each other within five seconds")
	}

}

func newDiscoveryClient(config *Config, setup chan []string, ready chan struct{}) {

	client, shutdown, err := config.create()
	if err != nil {
		client.logger.Error(err)
		return
	}
	defer shutdown()

	var addresses []string
	for _, addr := range client.Addresses() {
		addresses = append(addresses, addr+"/ipfs/"+client.ID())
	}

	setup <- addresses

	for {
		if float64(client.PeerCount()) >= float64(test_N) * 0.75 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	ready <- struct{}{}

	time.Sleep(5 * time.Second)

}
