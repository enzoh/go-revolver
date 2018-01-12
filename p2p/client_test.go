/**
 * File        : client_test.go
 * Description : Unit tests.
 * Copyright   : Copyright (c) 2017-2018 DFINITY Stiftung. All rights reserved.
 * Maintainer  : Enzo Haussecker <enzo@dfinity.org>
 * Stability   : Experimental
 */

package p2p

import (
	"testing"
)

// Create a test client.
func newTestClient(test *testing.T) (*client, func()) {

	// Create a configuration.
	config := DefaultConfig()
	config.DisableAnalytics = true
	config.DisableNATPortMap = true
	config.DisablePeerDiscovery = true
	config.DisableStreamDiscovery = true
	config.IP = "127.0.0.1"
	config.LogLevel = "debug"

	// Create a client.
	client, shutdown, err := config.create()
	if err != nil {
		test.Fatal(err)
	}

	// Ready for tests.
	return client, shutdown

}
