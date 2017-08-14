/**
 * File        : artifact.go
 * Description : High-level artifact interface.
 * Copyright   : Copyright (c) 2017 DFINITY Stiftung. All rights reserved.
 * Maintainer  : Enzo Haussecker <enzo@string.technology>
 * Stability   : Stable
 */

package artifact

import (
	"bytes"
	"errors"
	"io"

	"golang.org/x/crypto/sha3"
)

// A simple interface for reading artifacts.
type Artifact interface {

	// Get the purported checksum of an artifact.
	Checksum() [32]byte

	// Close an artifact.
	Close()

	// Close an artifact and disconnect from its sender.
	Disconnect()

	// Get the purported size of an artifact.
	Size() uint32

	// Wait for a finalizer to close an artifact.
	Wait() int

	io.Reader
}

type artifact struct {
	checksum [32]byte
	closer   chan int
	size     uint32
	reader   io.Reader
}

// Get the purported checksum of an artifact.
func (artifact *artifact) Checksum() [32]byte {
	return artifact.checksum
}

// Close an artifact.
func (artifact *artifact) Close() {
	artifact.closer <- 0
}

// Close an artifact and disconnect from its sender.
func (artifact *artifact) Disconnect() {
	artifact.closer <- 1
}

// Get the purported size of an artifact.
func (artifact *artifact) Size() uint32 {
	return artifact.size
}

// Wait for a finalizer to close an artifact.
func (artifact *artifact) Wait() int {
	return <-artifact.closer
}

// Read bytes from an artifact.
func (artifact *artifact) Read(data []byte) (n int, err error) {
	return artifact.reader.Read(data)
}

// Create an artifact from a reader.
func New(reader io.Reader, checksum [32]byte, size uint32) Artifact {
	return &artifact{
		checksum,
		make(chan int, 1),
		size,
		reader,
	}
}

// Create an artifact from a byte slice.
func FromBytes(data []byte) Artifact {
	return New(bytes.NewReader(data), sha3.Sum256(data), uint32(len(data)))
}

// Create a byte slice from an artifact. This will consume the artifact and
// apply a finalizer. Do not use the artifact after calling this function.
func ToBytes(artifact Artifact) ([]byte, error) {
	data := make([]byte, artifact.Size())
	_, err := io.ReadFull(artifact, data)
	if err != nil {
		artifact.Disconnect()
		return nil, err
	}
	if sha3.Sum256(data) != artifact.Checksum() {
		artifact.Disconnect()
		return nil, errors.New("Cannot verify checksum of artifact")
	}
	artifact.Close()
	return data, nil
}
