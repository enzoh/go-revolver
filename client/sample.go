/**
 * File        : sample.go
 * Description : Service for sampling routing tables.
 * Copyright   : Copyright (c) 2017 DFINITY Stiftung. All rights reserved.
 * Maintainer  : Enzo Haussecker <enzo@string.technology>
 * Stability   : Experimental
 */

package p2p

import (
	"encoding/json"
	"math"
	"math/rand"

	"github.com/dfinity/go-dfinity-p2p/util"
	"github.com/libp2p/go-libp2p-kbucket"
	"github.com/libp2p/go-libp2p-net"
	"github.com/libp2p/go-libp2p-peer"
	"github.com/libp2p/go-libp2p-peerstore"
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
		client.peerstore.SetAddrs(pid, addrs, 0)
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

	// Select peers from the routing table.
	peers := kbucket.SortClosestPeers(
		client.table.ListPeers(),
		kbucket.ConvertPeerID(pid),
	)
	var sample []peerstore.PeerInfo
	for len(peers) > 0 {
		if client.config.SampleSize <= len(sample) {
			break
		}
		n := float64(len(peers))
		j := int(math.Floor(math.Exp(math.Log(n+1)*rand.Float64()) - 1))
		info := client.peerstore.PeerInfo(peers[j])
		if info.ID != client.id && info.ID != pid && len(info.Addrs) != 0 {
			sample = append(sample, info)
		}
		peers = append(peers[:j], peers[j+1:]...)
	}

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

	// Update the routing table.
	client.table.Update(pid)

}

// Register the sampling handler.
func (client *client) registerSampleService() {
	uri := client.protocol + "/sample"
	client.host.SetStreamHandler(uri, client.sampleHandler)
}
