/**
 * File        : analytics_test.go
 * Description : Unit test.
 * Copyright   : Copyright (c) 2017-2018 DFINITY Stiftung. All rights reserved.
 * Maintainer  : Enzo Haussecker <enzo@dfinity.org>
 * Stability   : Experimental
 */

package p2p

import (
	"fmt"
	"net"
	"net/http"
	"sync"
	"testing"

	"gx/ipfs/QmPgDWmTmuzvP7QE5zwo1TmjbJme9pmZHNujB2453jkCTr/go-libp2p-peerstore"

	"github.com/dfinity/go-revolver/analytics"
)

// Show that clients can report their streams to an analytics server.
func TestAnalytics(test *testing.T) {

	// Create a simple key-value store for analytics reports.
	reports := make(map[string]*analytics.Report)
	lock := &sync.Mutex{}

	// Create a TCP listener to be used by the analytics server.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		test.Fatal(err)
	}
	defer listener.Close()

	// Start the analytics server on a separate thread.
	go func(reports map[string]*analytics.Report, lock *sync.Mutex, listener net.Listener) {
		http.HandleFunc("/report", analytics.ReportHandler(reports, lock))
		http.Serve(listener, nil)
	}(reports, lock, listener)

	// Define the URL used for report submission.
	url := fmt.Sprintf(
		"http://127.0.0.1:%d/report",
		listener.Addr().(*net.TCPAddr).Port,
	)

	// Create a client.
	client1 := newAnalyticsClient(test, url)
	defer client1.Close()

	// Create a second client.
	client2 := newAnalyticsClient(test, url)
	defer client2.Close()

	// Add the second client to the peer store of the first.
	client1.peerstore.AddAddrs(
		client2.id,
		client2.host.Addrs(),
		peerstore.ProviderAddrTTL,
	)

	// Add the first client to the routing table of the second.
	client2.table.Add(client1.id)

	// Pair the clients. This will allow them to exchange artifacts.
	success, err := client1.pair(client2.id)
	if err != nil {
		test.Fatal(err)
	}
	if !success {
		test.Fatal("Cannot negotiate artifact exchange")
	}

	// Send an analytics report.
	err = client1.genReport().Send(client1.config.AnalyticsURL)
	if err != nil {
		test.Fatal(err)
	}

	// Locate the analytics report.
	lock.Lock()
	report, exists := reports[client1.id.Pretty()]
	lock.Unlock()
	if !exists {
		test.Fatal("Cannot locate analytics report")
	}

	// Verify that the report references the artifact stream.
	if len(report.Streams) == 0 {
		test.Fatal("Missing reference to artifact stream")
	}
	if report.Streams[0] != client2.id.Pretty() {
		test.Fatal("Unexpected reference to artifact stream")
	}

}

func newAnalyticsClient(test *testing.T, url string) *client {

	// Configure an analytics client.
	config := DefaultConfig()
	config.AnalyticsURL = url
	config.DisableAnalytics = true
	config.DisableNATPortMap = true
	config.DisablePeerDiscovery = true
	config.DisableStreamDiscovery = true
	config.IP = "127.0.0.1"

	// Instantiate the client.
	client, err := config.create()
	if err != nil {
		test.Fatal(err)
	}

	// Ready for action!
	return client

}
