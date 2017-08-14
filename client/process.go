/**
 * File        : process.go
 * Description : Artifact processing module.
 * Copyright   : Copyright (c) 2017 DFINITY Stiftung. All rights reserved.
 * Maintainer  : Enzo Haussecker <enzo@string.technology>
 * Stability   : Experimental
 */

package p2p

import (
	"encoding/hex"
	"io"
	"io/ioutil"

	"github.com/dfinity/go-dfinity-p2p/artifact"
	"github.com/dfinity/go-dfinity-p2p/util"
	"github.com/libp2p/go-libp2p-net"
	"github.com/libp2p/go-libp2p-peer"
)

// Process artifacts from a stream.
func (client *client) process(stream net.Stream) {

	var buf [4]byte
	var checksum [32]byte
	var witnesses []peer.ID

	pid := stream.Conn().RemotePeer()

Processing:
	for {

		// Get the checksum of the artifact.
		_, err := io.ReadFull(stream, checksum[:])
		if err != nil {
			if err == io.EOF {
				client.logger.Debug("Disconnecting from", pid)
			} else {
				client.logger.Warning("Cannot get checksum of artifact from", pid, err)
			}
			break Processing
		}
		code := hex.EncodeToString(checksum[:4])

		// Update the witnesses of the artifact.
		client.witnessesLock.Lock()
		peers, exists := client.witnesses.Get(checksum)
		if exists {
			witnesses = peers.([]peer.ID)
		}
		client.witnesses.Add(checksum, append(witnesses, pid))
		client.witnessesLock.Unlock()

		// Get the size of the artifact.
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

		// Check if the client can create a buffer that large.
		if size > client.config.ArtifactMaxBufferSize {
			client.logger.Warningf("Cannot accept %d byte artifact with checksum %s from %v", size, code, pid)
			break Processing
		}

		// Log the artifact details.
		client.logger.Debugf("Receiving %d byte artifact with checksum %s from %v", size, code, pid)

		// Check if the client has already received the artifact.
		client.artifactsLock.Lock()
		if client.artifacts.Contains(checksum) {
			client.artifactsLock.Unlock()
			_, err = io.CopyN(ioutil.Discard, stream, int64(size))
			if err != nil {
				if err == io.EOF {
					client.logger.Debug("Disconnecting from", pid)
				} else {
					client.logger.Warning("Cannot read artifact from", pid, err)
				}
				break Processing
			}
			continue Processing
		}

		// Update the artifact cache.
		client.artifacts.Add(checksum, size)
		client.artifactsLock.Unlock()

		// Queue the artifact.
		artifact := artifact.New(stream, checksum, size)
		client.receive <- artifact

		// Check if the artifact was invalid.
		if artifact.Wait() != 0 {
			client.logger.Debug("Disconnecting from", pid)
			break Processing
		}

	}

	client.streamstore.Remove(pid)

}
