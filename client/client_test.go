/**
 * File        : client_test.go
 * Description : Unit tests.
 * Copyright   : Copyright (c) 2017 DFINITY Stiftung. All rights reserved.
 * Maintainer  : Enzo Haussecker <enzo@string.technology>
 * Stability   : Experimental
 */

package p2p

import (
	"crypto/rand"
	"encoding/hex"
	"testing"
)

// Create a test client.
func newTestClient(test *testing.T, port uint16) (*client, func()) {

	// Create a random seed.
	seed := make([]byte, 32)
	_ ,err := rand.Read(seed)
	if err != nil {
		test.Fatal(err)
	}

	// Create a configuration.
	config, err := DefaultConfig()
	if err != nil {
		test.Fatal(err)
	}
	config.DisableNATPortMap = true
	config.DisablePeerDiscovery = true
	config.DisableStreamDiscovery = true
	config.ListenIP = "127.0.0.1"
	config.ListenPort = port
	config.RandomSeed = hex.EncodeToString(seed)

	// Create a client.
	client, shutdown, err := config.new()
	if err != nil {
		test.Fatal(err)
	}

	// Ready for tests.
	return client, shutdown

}
