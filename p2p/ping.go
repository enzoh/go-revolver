/**
 * File        : ping.go
 * Description : Service for testing the reachability of peers.
 * Copyright   : Copyright (c) 2017-2018 DFINITY Stiftung. All rights reserved.
 * Maintainer  : Enzo Haussecker <enzo@dfinity.org>
 * Stability   : Experimental
 */

package p2p

import (
	"bytes"
	"crypto/rand"
	"errors"
	"time"

	"gx/ipfs/QmNa31VPzC561NWwRsJLE7nGYZYuuD2QfpK2b1q9BK54J1/go-libp2p-net"
	"gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"

	"github.com/dfinity/go-revolver/util"
)

func (c *client) probeLatency(pid peer.ID) (zero time.Duration, err error) {
	// Log this action.
	c.logger.Debug("Ping", pid)

	// Connect to the target peer.
	stream, err := c.host.NewStream(
		c.context,
		pid,
		c.protocol+"/ping",
	)
	if err != nil {
		addrs := c.peerstore.PeerInfo(pid).Addrs
		c.logger.Debug("Cannot connect to", pid, "at", addrs, err)
		c.peerstore.ClearAddrs(pid)
		c.table.Remove(pid)
		return zero, err
	}
	defer stream.Close()

	// Generate random data.
	wbuf := make([]byte, c.config.PingBufferSize)
	_, err = rand.Reader.Read(wbuf)
	if err != nil {
		c.logger.Warning("Cannot generate random data", err)
		return zero, err
	}

	// Observe the current time.
	before := time.Now()

	// Send data to the target peer.
	err = util.WriteWithTimeout(stream, wbuf, c.config.Timeout)
	if err != nil {
		c.logger.Warning("Cannot send data to", pid, err)
		return zero, err
	}

	// Receive data from the target peer.
	rbuf, err := util.ReadWithTimeout(
		stream,
		c.config.PingBufferSize,
		c.config.Timeout,
	)
	if err != nil {
		c.logger.Warning("Cannot receive data from", pid, err)
		return zero, err
	}

	// Verify that the data sent and received is the same.
	if !bytes.Equal(wbuf, rbuf) {
		err = errors.New("Corrupt data!")
		c.logger.Warning("Cannot verify data received from", pid, err)
		return zero, err
	}

	return time.Since(before), nil
}

// Ping a peer.
func (c *client) ping(pid peer.ID) error {
	// Measure the latency
	latency, err := c.probeLatency(pid)
	if err != nil {
		return err
	}

	// Record the observed latency.
	c.peerstore.RecordLatency(pid, latency)

	// Success.
	return nil

}

// Handle incomming pings.
func (c *client) pingHandler(stream net.Stream) {

	defer stream.Close()

	// Log this action.
	pid := stream.Conn().RemotePeer()
	c.logger.Debug("Pong", pid)

	// Receive data from the target peer.
	rbuf, err := util.ReadWithTimeout(
		stream,
		c.config.PingBufferSize,
		c.config.Timeout,
	)
	if err != nil {
		c.logger.Warning("Cannot receive data from", pid, err)
		return
	}

	// Send data to the target peer.
	err = util.WriteWithTimeout(stream, rbuf, c.config.Timeout)
	if err != nil {
		c.logger.Warning("Cannot send data to", pid, err)
	}

	// Update the routing table.
	c.table.Update(pid)

}

// Register the ping handler.
func (c *client) registerPingService() {
	uri := c.protocol + "/ping"
	c.host.SetStreamHandler(uri, c.pingHandler)
}
