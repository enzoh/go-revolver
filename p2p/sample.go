/**
 * File        : sample.go
 * Description : Service for sampling routing tables.
 * Copyright   : Copyright (c) 2017-2018 DFINITY Stiftung. All rights reserved.
 * Maintainer  : Enzo Haussecker <enzo@dfinity.org>
 * Stability   : Experimental
 */

package p2p

import (
	"encoding/json"

	"gx/ipfs/QmNa31VPzC561NWwRsJLE7nGYZYuuD2QfpK2b1q9BK54J1/go-libp2p-net"
	"gx/ipfs/QmPgDWmTmuzvP7QE5zwo1TmjbJme9pmZHNujB2453jkCTr/go-libp2p-peerstore"
	"gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"

	"github.com/dfinity/go-revolver/util"
)

// Get a random sample of peers from the routing table of a peer.
func (client *client) sample(peerId peer.ID) ([]peerstore.PeerInfo, error) {

	// Log this action.
	pid := peerId
	client.logger.Debug("Requesting peers from", pid)

	// Connect to the target peer.
	stream, err := client.host.NewStream(
		client.context,
		pid,
		client.protocol+"/sample",
	)
	if err != nil {
		addrs := client.peerstore.PeerInfo(pid).Addrs
		client.logger.Debug("Cannot connect to", pid, "at", addrs, err)
		client.peerstore.ClearAddrs(pid)
		client.table.Remove(pid)
		return nil, err
	}
	defer stream.Close()

	// Receive a buffer size from the target peer.
	size, err := util.ReadUInt32WithTimeout(
		stream,
		client.config.Timeout,
	)
	if err != nil {
		client.logger.Warning("Cannot receive buffer size from", pid, err)
		return nil, err
	}

	// Check if the client can create a buffer that large.
	if size > client.config.SampleMaxBufferSize {
		client.logger.Warningf("Cannot accept %d byte peer list from %v", size, pid)
		return nil, err
	}

	// Receive data from the target peer.
	data, err := util.ReadWithTimeout(
		stream,
		size,
		client.config.Timeout,
	)
	if err != nil {
		client.logger.Warning("Cannot receive data from", pid, err)
		return nil, err
	}

	// Decode the data received from the target peer.
	var sample []peerstore.PeerInfo
	err = json.Unmarshal(data, &sample)
	if err != nil {
		client.logger.Warning("Cannot decode data received from", pid, err)
		return nil, err
	}

	// Success.
	return sample, nil

}

// Handle incomming requests for peers.
func (client *client) sampleHandler(stream net.Stream) {

	defer stream.Close()

	// Log this action.
	pid := stream.Conn().RemotePeer()
	client.logger.Debug("Receiving request for peers from", pid)

	// Check if the target peer is authorized to perform this action.
	authorized, err := client.peerstore.Get(pid, "AUTHORIZED")
	if err != nil || !authorized.(bool) {
		client.logger.Warning("Unauthorized request from", pid)
		return
	}

	// Select peers from the routing table.
	sample := client.table.Sample(client.config.SampleSize, []peer.ID{pid})

	// Encode the peer list.
	data, err := json.Marshal(sample)
	if err != nil {
		client.logger.Warning("Cannot encode peers")
		return
	}

	// Send the peer list to the target peer.
	client.logger.Debug("Providing", pid, "with peers", sample)
	size := util.EncodeBigEndianUInt32(uint32(len(data)))
	err = util.WriteWithTimeout(
		stream,
		append(size[:], data...),
		client.config.Timeout,
	)
	if err != nil {
		client.logger.Warning("Cannot send peers to", pid, err)
	}

}

// Register the sampling handler.
func (client *client) registerSampleService() {
	uri := client.protocol + "/sample"
	client.host.SetStreamHandler(uri, client.sampleHandler)
}
