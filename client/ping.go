/**
 * File        : ping.go
 * Description : Service for testing the reachability of peers.
 * Copyright   : Copyright (c) 2017 DFINITY Stiftung. All rights reserved.
 * Maintainer  : Enzo Haussecker <enzo@string.technology>
 * Stability   : Experimental
 */

package p2p

import (
	"bytes"
	"crypto/rand"
	"errors"
	"time"

	"github.com/dfinity/go-dfinity-p2p/util"
	"github.com/libp2p/go-libp2p-net"
	"github.com/libp2p/go-libp2p-peer"
)

// Ping a peer.
func (client *client) ping(peerId peer.ID) error {

	// Log this action.
	pid := peerId
	client.logger.Debug("Ping", pid)

	// Connect to the target peer.
	stream, err := client.host.NewStream(
		client.context,
		pid,
		client.protocol+"/ping",
	)
	if err != nil {
		addrs := client.peerstore.PeerInfo(pid).Addrs
		client.logger.Debug("Cannot connect to", pid, "at", addrs, err)
		client.peerstore.SetAddrs(pid, addrs, 0)
		client.table.Remove(pid)
		return err
	}
	defer stream.Close()

	// Generate random data.
	wbuf := make([]byte, client.config.PingBufferSize)
	_, err = rand.Reader.Read(wbuf)
	if err != nil {
		client.logger.Warning("Cannot generate random data", err)
		return err
	}

	// Observe the current time.
	before := time.Now()

	// Send data to the target peer.
	err = util.WriteWithTimeout(stream, wbuf, client.config.Timeout)
	if err != nil {
		client.logger.Warning("Cannot send data to", pid, err)
		return err
	}

	// Receive data from the target peer.
	rbuf, err := util.ReadWithTimeout(
		stream,
		client.config.PingBufferSize,
		client.config.Timeout,
	)
	if err != nil {
		client.logger.Warning("Cannot receive data from", pid, err)
		return err
	}

	// Verify that the data sent and received is the same.
	if !bytes.Equal(wbuf, rbuf) {
		err = errors.New("Corrupt data!")
		client.logger.Warning("Cannot verify data received from", pid, err)
		return err
	}

	// Record the observed latency.
	client.peerstore.RecordLatency(pid, time.Since(before))

	// Success.
	return nil

}

// Handle incomming pings.
func (client *client) pingHandler(stream net.Stream) {

	defer stream.Close()

	// Log this action.
	pid := stream.Conn().RemotePeer()
	client.logger.Debug("Pong", pid)

	// Receive data from the target peer.
	rbuf, err := util.ReadWithTimeout(
		stream,
		client.config.PingBufferSize,
		client.config.Timeout,
	)
	if err != nil {
		client.logger.Warning("Cannot receive data from", pid, err)
		return
	}

	// Send data to the target peer.
	err = util.WriteWithTimeout(stream, rbuf, client.config.Timeout)
	if err != nil {
		client.logger.Warning("Cannot send data to", pid, err)
	}

	// Update the routing table.
	client.table.Update(pid)

}

// Register the ping handler.
func (client *client) registerPingService() {
	uri := client.protocol + "/ping"
	client.host.SetStreamHandler(uri, client.pingHandler)
}
