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
	"time"

	"gx/ipfs/QmUEoLmhwH2CkiwHkfHVNeHm9WtMAxTh7jjUQAMRs1rNDe/go-revolver-util"
	"gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"
	"gx/ipfs/QmZMeWroC7S4X8183A3TyGrzgsGjMWaBMedXyox1GUAvej/go-revolver-artifact"
	"gx/ipfs/QmahYsGWry85Y7WUe2SX5G4JkH2zifEQAUtJVLZ24aC9DF/go-libp2p-net"
)

// Process artifacts from a stream.
func (client *client) process(stream net.Stream) {

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

		// Get the size of the artifact.
		size, err := util.ReadUInt32WithTimeout(
			stream,
			client.config.Timeout,
		)
		if err != nil {
			if err == io.EOF {
				client.logger.Debug("Disconnecting from", pid)
			} else {
				client.logger.Warning("Cannot get size of artifact from", pid, err)
			}
			break Processing
		}

		// Check if the client can create a buffer that large.
		if size > client.config.ArtifactMaxBufferSize {
			client.logger.Warningf("Cannot accept %d byte artifact with checksum %s from %v", size, code, pid)
			break Processing
		}

		// Get the timestamp of the artifact.
		n, err := util.ReadInt64WithTimeout(
			stream,
			client.config.Timeout,
		)
		if err != nil {
			if err == io.EOF {
				client.logger.Debug("Disconnecting from", pid)
			} else {
				client.logger.Warning("Cannot get timestamp of artifact from", pid, err)
			}
			break Processing
		}
		timestamp := time.Unix(n/1000000000, n%1000000000).UTC()
		latency := time.Since(timestamp)

		// Log the artifact details.
		client.logger.Debugf("Receiving %d byte artifact with checksum %s and latency %s from %v", size, code, latency, pid)

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

		// Consume the artifact.
		object := artifact.New(stream, checksum, size, timestamp)
		data, err := artifact.ToBytes(object)
		if err != nil {
			if err == io.EOF {
				client.logger.Debug("Disconnecting from", pid)
			} else {
				client.logger.Warning("Cannot read artifact from", pid, err)
			}
			break Processing
		}

		// Update the witnesses of the artifact.
		client.witnessesLock.Lock()
		peers, exists := client.witnesses.Get(checksum)
		if exists {
			witnesses = peers.([]peer.ID)
		}
		client.witnesses.Add(checksum, append(witnesses, pid))
		client.witnessesLock.Unlock()

		// Queue the artifact.
		client.receive <- data

		// Check if the artifact was invalid.
		if object.Wait() != 0 {
			client.logger.Debug("Disconnecting from", pid)
			break Processing
		}

	}

	client.streamstore.Remove(pid)

}
