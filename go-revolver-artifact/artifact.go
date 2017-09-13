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
	"crypto/sha256"
	"errors"
	"io"
	"time"
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

	// Get the purported timestamp of an artifact.
	Timestamp() time.Time

	// Wait for a finalizer to close an artifact.
	Wait() int

	io.Reader
}

type artifact struct {
	checksum  [32]byte
	closer    chan int
	size      uint32
	timestamp time.Time
	reader    io.Reader
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

// Get the purported timestamp of an artifact.
func (artifact *artifact) Timestamp() time.Time {
	return artifact.timestamp
}

// Wait for a finalizer to close an artifact.
func (artifact *artifact) Wait() int {
	return <-artifact.closer
}

// Read bytes from an artifact.
func (artifact *artifact) Read(data []byte) (n int, err error) {
	return artifact.reader.Read(data)
}

// Create an artifact.
func New(reader io.Reader, checksum [32]byte, size uint32, timestamp time.Time) Artifact {
	return &artifact{
		checksum,
		make(chan int, 1),
		size,
		timestamp,
		reader,
	}
}

// Create an artifact from a byte slice.
func FromBytes(data []byte) Artifact {
	return New(
		bytes.NewReader(data),
		sha256.Sum256(data),
		uint32(len(data)),
		time.Now(),
	)
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
	if sha256.Sum256(data) != artifact.Checksum() {
		artifact.Disconnect()
		return nil, errors.New("Cannot verify checksum of artifact")
	}
	artifact.Close()
	return data, nil
}
