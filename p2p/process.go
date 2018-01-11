/**
 * File        : process.go
 * Description : Artifact processing module.
 * Copyright   : Copyright (c) 2017-2018 DFINITY Stiftung. All rights reserved.
 * Maintainer  : Enzo Haussecker <enzo@dfinity.org>
 * Stability   : Experimental
 */

package p2p

import (
	"encoding/hex"
	"io"
	"io/ioutil"
	"time"

	"gx/ipfs/QmNa31VPzC561NWwRsJLE7nGYZYuuD2QfpK2b1q9BK54J1/go-libp2p-net"
	"gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"

	"github.com/dfinity/go-revolver/artifact"
)

// Process artifacts from a stream.
func (client *client) process(stream net.Stream) {

	var metadata [45]byte
	var witnesses []peer.ID

	pid := stream.Conn().RemotePeer()

Processing:
	for {

		// Read the artifact metadata.
		_, err := io.ReadFull(stream, metadata[:])
		if err != nil {
			if isProbableEOF(err) {
				client.logger.Debug("Disconnecting from", pid)
			} else {
				client.logger.Warning("Cannot get artifact metadata from", pid, err)
			}
			break Processing
		}
		checksum, compression, size, timestamp := artifact.DecodeMetadata(metadata)

		// Log the artifact metadata.
		code := hex.EncodeToString(checksum[:4])
		latency := time.Since(timestamp)
		client.logger.Debugf("Receiving %d byte artifact with checksum %s and latency %s from %v", size, code, latency, pid)

		// Check if the client can buffer the artifact.
		if size > client.config.ArtifactMaxBufferSize {
			client.logger.Warningf("Cannot accept %d byte artifact with checksum %s from %v", size, code, pid)
			break Processing
		}

		// Check if the client has already received the artifact.
		client.artifactCacheLock.Lock()
		if client.artifactCache.Contains(checksum) {
			client.artifactCacheLock.Unlock()
			_, err = io.CopyN(ioutil.Discard, stream, int64(size))
			if err != nil {
				if isProbableEOF(err) {
					client.logger.Debug("Disconnecting from", pid)
				} else {
					client.logger.Warning("Cannot read artifact from", pid, err)
				}
				break Processing
			}
			continue Processing
		}

		// Update the artifact cache.
		client.artifactCache.Add(checksum, size)
		client.artifactCacheLock.Unlock()

		// Update the witnesses of the artifact.
		client.witnessCacheLock.Lock()
		peers, exists := client.witnessCache.Get(checksum)
		if exists {
			witnesses = peers.([]peer.ID)
		}
		client.witnessCache.Add(checksum, append(witnesses, pid))
		client.witnessCacheLock.Unlock()

		// Queue the artifact.
		object := artifact.New(stream, checksum, compression, size, timestamp)
		client.receive <- object

		// Check if the artifact was invalid.
		if object.Wait() != 0 {
			client.logger.Debug("Disconnecting from", pid)
			break Processing
		}

	}

	client.streamstore.Remove(pid)

}

// Check if an error resembles a connection termination scenario that would
// justify assuming that the watch is empty.
func isProbableEOF(err error) bool {
	switch {
	case err == io.EOF:
		return true
	case err.Error() == "connection reset":
		return true
	}
	return false
}
