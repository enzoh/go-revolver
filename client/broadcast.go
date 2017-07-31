/**
 * File        : broadcast.go
 * Description : Artifact broadcasting module.
 * Copyright   : Copyright (c) 2017 DFINITY Stiftung. All rights reserved.
 * Maintainer  : Enzo Haussecker <enzo@string.technology>
 * Stability   : Experimental
 */

package p2p

import (
	"bytes"
	"errors"
	"io"

	"github.com/dfinity/go-dfinity-p2p/artifact"
	"github.com/dfinity/go-dfinity-p2p/util"
	"github.com/libp2p/go-libp2p-peer"
	"golang.org/x/crypto/sha3"
)

// Activate the artifact broadcast.
func (client *client) activateBroadcast() func() {

	// Create a shutdown function.
	notify := make(chan struct {}, 1)
	shutdown := func() {
		notify <-struct {}{}
	}

	// Broadcast artifacts from the send queue.
	go func() {
		Broadcast:
		for {
			select {
			case <-notify:
				break Broadcast
			case data := <-client.send:
				checksum := sha3.Sum256(data)
				size := uint32(len(data))
				artifact := artifact.New(
					bytes.NewReader(data),
					checksum,
					size,
				)
				client.artifacts.Add(checksum, size)
				client.broadcast(artifact)
			}
		}
	}()

	// Return the shutdown function.
	return shutdown

}

// Broadcast an artifact.
func (client *client) broadcast(artifact artifact.Artifact) {

	// Get the artifact metadata.
	checksum := artifact.Checksum()
	size := util.EncodeBigEndianUInt32(artifact.Size())
	metadata := append(checksum[:], size[:]...)

	// Calculate the number of chunks to transfer.
	chunks := int((artifact.Size() + client.config.ArtifactChunkSize - 1) /
		client.config.ArtifactChunkSize + 1)

	// Send the artifact metadata to those who have not seen it.
	errors := make([]map[peer.ID]chan error, chunks)
	errors[0] = client.streamstore.Apply(
	func(peerId peer.ID, writer io.Writer) error {
		if client.witness(peerId, checksum) {
			return escape
		}
		return util.WriteWithTimeout(
			writer,
			metadata,
			client.config.Timeout,
		)
	})

	// Send the artifact in chunks.
	leftover := artifact.Size()
	for i := 1; i < chunks; i++ {

		// Create a chunk.
		var data []byte
		if leftover < client.config.ArtifactChunkSize {
			data = make([]byte, leftover)
			leftover = 0
		} else {
			data = make([]byte, client.config.ArtifactChunkSize)
			leftover -= client.config.ArtifactChunkSize
		}
		_, err := io.ReadFull(artifact, data)
		if err != nil {
			client.logger.Warning("Cannot read artifact")
			artifact.Closer() <-1
			return
		}

		// Send the chunk to those who received the previous chunk.
		previous := errors[i-1]
		errors[i] = client.streamstore.Apply(
		func(peerId peer.ID, writer io.Writer) error {
			result, exists := previous[peerId]
			if exists {
				err := <-result
				if err != nil {
					return err
				}
				return util.WriteWithTimeout(
					writer,
					data,
					client.config.Timeout,
				)
			}
			return nil
		})

	}

	// Remove anyone who failed to receive the artifact.
	for peerId, result := range errors[chunks-1] {
		go func(peerId peer.ID, result chan error) {
			pid := peerId
			err := <-result
			if err != nil && err != escape {
				client.logger.Debug(pid, "failed to receive the artifact", err)
				client.streamstore.Remove(pid)
			}
		}(peerId, result)
	}

	// Close the reader.
	artifact.Closer() <-0

}

// Check if a peer has received an artifact.
func (client * client) witness(peerId peer.ID, checksum [32]byte) bool {
	witnesses, exists := client.witnesses.Get(checksum)
	if exists {
		for _, id := range witnesses.([]peer.ID) {
			if id == peerId {
				return true
			}
		}
	}
	return false
}

// An error to indicate that control was transferred to another operation.
var escape = errors.New("ESC")
