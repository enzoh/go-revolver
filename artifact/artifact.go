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

	// Get the checksum of an artifact.
	Checksum() [32]byte

	// Get the closer of an artifact.
	Closer() chan int

	// Get the size of an artifact.
	Size() uint32

	io.Reader
}

type artifact struct {
	checksum [32]byte
	closer   chan int
	size     uint32
	reader   io.Reader
}

// Get the checksum of an artifact.
func (artifact *artifact) Checksum() [32]byte {
	return artifact.checksum
}

// Get the closer of an artifact.
func (artifact *artifact) Closer() chan int {
	return artifact.closer
}

// Get the size of an artifact.
func (artifact *artifact) Size() uint32 {
	return artifact.size
}

// Read bytes from an artifact.
func (artifact *artifact) Read(data []byte) (n int, err error) {
	return artifact.reader.Read(data)
}

// Create an artifact.
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

// Create a byte slice from an artifact.
func ToBytes(artifact Artifact) ([]byte, error) {
	data := make([]byte, artifact.Size())
	_, err := io.ReadFull(artifact, data)
	if err != nil {
		artifact.Closer() <- 1
		return nil, err
	}
	if sha3.Sum256(data) != artifact.Checksum() {
		artifact.Closer() <- 1
		return nil, errors.New("Cannot verify checksum of artifact")
	}
	artifact.Closer() <- 0
	return data, nil
}
