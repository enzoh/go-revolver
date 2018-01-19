/**
 * File        : mesh_test.go
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

// Show that twenty-four clients are capable of meshing within five seconds.
func TestMesh(test *testing.T) {

	const N = 24

	setup := make(chan struct{}, N)
	ready := make(chan struct{}, N)

	config := DefaultConfig()
	config.DisableAnalytics = true
	config.DisableNATPortMap = true
	config.IP = "127.0.0.1"
	config.Port = 5555
	config.RandomSeed = "0000000000000000000000000000000000000000000000000000000000000000"

	go newTestMeshClient(config, setup, ready)

	select {
	case <-setup:
	case <-time.After(time.Second):
		test.Fatal("seed node failed to initialize within one second")
	}

	config.Port = 0
	config.RandomSeed = ""
	config.SeedNodes = []string{"/ip4/127.0.0.1/tcp/5555/ipfs/QmbxanwNroEVkz3RuFTpHrCvBzJip631dhLtK7qqDkMzd4"}

	for i := 1; i < N; i++ {
		go newTestMeshClient(config, setup, ready)
	}

	done := make(chan struct{}, 1)

	go func() {
		for i := 0; i < N; i++ {
			<-ready
		}
		done <- struct{}{}
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		test.Fatal("nodes failed to mesh within five seconds")
	}

}

func newTestMeshClient(config *Config, setup chan struct{}, ready chan struct{}) {

	client, shutdown, err := config.New()
	if err != nil {
		panic(err)
	}
	defer shutdown()

	setup <- struct{}{}

	for {
		if client.StreamCount() >= 6 {
			break
		}
		time.Sleep(time.Second)
	}

	ready <- struct{}{}

	time.Sleep(5 * time.Second)

}
