/**
 * File        : client_test.go
 * Description : Unit tests.
 * Copyright   : Copyright (c) 2017-2018 DFINITY Stiftung. All rights reserved.
 * Maintainer  : Enzo Haussecker <enzo@dfinity.org>
 * Stability   : Experimental
 */

package p2p

import (
	"crypto/rand"
	"encoding/hex"
	"testing"

	"github.com/enzoh/go-logging"
)

// Create a test client.
func newTestClient(test *testing.T) (*client, func()) {

	// Create a random seed.
	seed := make([]byte, 32)
	_, err := rand.Read(seed)
	if err != nil {
		test.Fatal(err)
	}

	// Create a configuration.
	config, err := DefaultConfig()
	if err != nil {
		test.Fatal(err)
	}
	config.DisableAnalytics = true
	config.DisableNATPortMap = true
	config.DisablePeerDiscovery = true
	config.DisableStreamDiscovery = true
	config.IP = "127.0.0.1"
	config.LogLevel = logging.DEBUG
	config.Port = 0
	config.RandomSeed = hex.EncodeToString(seed)

	// Create a client.
	client, shutdown, err := config.new()
	if err != nil {
		test.Fatal(err)
	}

	// Ready for tests.
	return client, shutdown

}
