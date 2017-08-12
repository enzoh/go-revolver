/**
 * File        : process.go
 * Description : Artifact processing module.
 * Copyright   : Copyright (c) 2017 DFINITY Stiftung. All rights reserved.
 * Maintainer  : Enzo Haussecker <enzo@string.technology>
 * Stability   : Experimental
 */

package p2p

import (
	"io"

	"github.com/dfinity/go-dfinity-p2p/artifact"
	"github.com/dfinity/go-dfinity-p2p/util"
	"github.com/libp2p/go-libp2p-net"
	"github.com/libp2p/go-libp2p-peer"
	"golang.org/x/crypto/sha3"
)

// Process artifacts from a stream.
func (client *client) process(stream net.Stream) {

	pid := stream.Conn().RemotePeer()

Processing:
	for {

		// Get the checksum of the artifact.
		var checksum [32]byte
		_, err := io.ReadFull(stream, checksum[:])
		if err != nil {
			if err == io.EOF {
				client.logger.Debug("Disconnecting from", pid)
			} else {
				client.logger.Warning("Cannot get checksum of artifact from", pid, err)
			}
			break Processing
		}

		// Get the size of the artifact.
		var buf [4]byte
		_, err = io.ReadFull(stream, buf[:])
		if err != nil {
			if err == io.EOF {
				client.logger.Debug("Disconnecting from", pid)
			} else {
				client.logger.Warning("Cannot get size of artifact from", pid, err)
			}
			break Processing
		}
		size := util.DecodeBigEndianUInt32(buf)

		// Check if the client can create an artifact that large.
		if size > client.config.ArtifactMaxBufferSize {
			client.logger.Warningf("Cannot accept %d byte artifact from %v", size, pid)
			break Processing
		}

		// Create the artifact.
		artifact := artifact.New(stream, checksum, size)

		// Read the artifact.
		data := make([]byte, size)
		_, err = io.ReadFull(artifact, data)
		if err != nil {
			if err == io.EOF {
				client.logger.Debug("Disconnecting from", pid)
			} else {
				client.logger.Warning("Cannot read artifact from", pid, err)
			}
			break Processing
		}

		// Verify the checksum of the artifact.
		if sha3.Sum256(data) != checksum {
			client.logger.Warning("Cannot verify checksum of artifact from", pid, err)
			break Processing
		}

		// Update the witnesses of the artifact.
		var witnesses []peer.ID
		peers, exists := client.witnesses.Get(checksum)
		if exists {
			witnesses = peers.([]peer.ID)
		}
		client.witnesses.Add(checksum, append(witnesses, pid))

		// Check if the client has already received the artifact.
		client.artifactsLock.Lock()
		if client.artifacts.Contains(checksum) {
			client.artifactsLock.Unlock()
			continue Processing
		}

		// Update the artifact cache.
		client.artifacts.Add(checksum, size)
		client.artifactsLock.Unlock()

		// Queue the artifact.
		client.receive <- data

	}

	client.streamstore.Remove(pid)

}
