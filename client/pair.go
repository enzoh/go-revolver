/**
 * File        : pair.go
 * Description : Service for pairing artifact streams.
 * Copyright   : Copyright (c) 2017 DFINITY Stiftung. All rights reserved.
 * Maintainer  : Enzo Haussecker <enzo@string.technology>
 * Stability   : Experimental
 */

package p2p

import (
	"fmt"

	"github.com/dfinity/go-dfinity-p2p/util"
	"github.com/libp2p/go-libp2p-net"
	"github.com/libp2p/go-libp2p-peer"
)

const (
	ack = 0x06
	nak = 0x15
)

// Request to exchange artifacts with a peer.
func (client *client) pair(peerId peer.ID) (bool, error) {

	// Log this action.
	pid := peerId
	client.logger.Debug("Requesting to pair with", pid)

	// Connect to the target peer.
	stream, err := client.host.NewStream(
		client.context,
		pid,
		client.protocol + "/pair",
	)
	if err != nil {
		addrs := client.peerstore.PeerInfo(pid).Addrs
		client.logger.Debug("Cannot connect to", pid, "at", addrs, err)
		client.peerstore.SetAddrs(pid, addrs, 0)
		client.table.Remove(pid)
		return false, err
	}

	// Receive data from the target peer.
	data, err := util.ReadWithTimeout(
		stream,
		1,
		client.config.Timeout,
	)
	if err != nil {
		client.logger.Warning("Cannot receive data from", pid, err)
		stream.Close()
		return false, err
	}

	// Add the stream to the stream store.
	var success bool
	if data[0] == ack && client.streamstore.Add(pid, stream) {

		// Ready to exchange artifacts.
		client.logger.Debug("Ready to exchange artifacts with", pid)
		go client.process(stream)
		success = true

	} else {

		// Cannot pair with the target peer.
		client.logger.Debug("Cannot pair with", pid)
		stream.Close()
		success = false

	}

	// Return without error.
	return success, nil

}

// Handle incomming pairing requests.
func (client *client) pairHandler() net.StreamHandler {

	return func(stream net.Stream) {

		// Log this action.
		pid := stream.Conn().RemotePeer()
		client.logger.Debug("Receiving request to pair with", pid)

		// Prepare to reject the request.
		reject := func(reason ...interface{}) {
			client.logger.Debug("Cannot pair with", pid, "because", fmt.Sprint(reason...))
			err := util.WriteWithTimeout(
				stream,
				[]byte{nak},
				client.config.Timeout,
			)
			if err != nil {
				client.logger.Warning("Cannot send data to", pid, err)
			}
			stream.Close()
		}

		// Calculate the targets.
		targetL := client.streamstore.Capacity() / 2
		targetR := client.streamstore.Capacity() - targetL

		// Identify candidates for pairing.
		queueL, queueR := client.neighbors(client.table.ListPeers())
		if len(queueL) > targetL {
			queueL = queueL[:targetL]
		}
		if len(queueR) > targetR {
			queueR = queueR[:targetR]
		}
		candidates := difference(
			append(queueL, queueR...),
			client.streamstore.Peers(),
		)

		// Check if the peer is a neighbor.
		neighbor := false
		for i := range candidates {
			if candidates[i] == pid {
				neighbor = true
				break
			}
		}
		if !neighbor {
			reject(pid, " is not a neighbor")
			return
		}

		// Create space for the stream.
		trashL, trashR := client.neighbors(
			append(client.streamstore.Peers(), pid),
		)
		if len(trashL) > targetL {
			trashL = trashL[targetL:]
			for i := range trashL {
				client.streamstore.Remove(trashL[i])
			}
		}
		if len(trashR) > targetR {
			trashR = trashR[targetR:]
			for i := range trashR {
				client.streamstore.Remove(trashR[i])
			}
		}

		// Check if the client can add the stream to the stream store.
		if !client.streamstore.Add(pid, stream) {
			reject(pid, " cannot be added to the stream store")
			return
		}

		// Send an acknowledgement.
		err := util.WriteWithTimeout(
			stream,
			[]byte{ack},
			client.config.Timeout,
		)
		if err != nil {
			client.logger.Warning("Cannot send data to", pid, err)
			client.streamstore.Remove(pid)
			return
		}

		// Ready to exchange artifacts.
		client.logger.Debug("Ready to exchange artifacts with", pid)
		go client.process(stream)

	}

}

// Register the pairing handler.
func (client *client) registerPairService() {
	uri := client.protocol + "/pair"
	client.host.SetStreamHandler(uri, client.pairHandler())
}
