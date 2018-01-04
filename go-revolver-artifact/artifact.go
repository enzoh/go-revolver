/**
 * File        : artifact.go
 * Description : High-level artifact interface.
 * Copyright   : Copyright (c) 2017-2018 DFINITY Stiftung. All rights reserved.
 * Maintainer  : Enzo Haussecker <enzo@dfinity.org>
 * Stability   : Stable
 */

package artifact

import (
	"bytes"
	"compress/gzip"
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

	// Check if an artifact uses gzip compression.
	Compression() bool

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
	checksum    [32]byte
	closer      chan int
	compression bool
	reader      io.Reader
	size        uint32
	timestamp   time.Time
}

// Get the purported checksum of an artifact.
func (artifact *artifact) Checksum() [32]byte {
	return artifact.checksum
}

// Close an artifact.
func (artifact *artifact) Close() {
	artifact.closer <- 0
}

// Check if an artifact uses gzip compression.
func (artifact *artifact) Compression() bool {
	return artifact.compression
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
func New(reader io.Reader, checksum [32]byte, compression bool, size uint32, timestamp time.Time) Artifact {
	return &artifact{
		checksum,
		make(chan int, 1),
		compression,
		reader,
		size,
		timestamp,
	}
}

// Create an artifact from a byte slice.
func FromBytes(data []byte, compression bool) (Artifact, error) {

	var (
		buffer bytes.Buffer
		reader io.Reader
	)

	if compression {

		writer, err := gzip.NewWriterLevel(&buffer, gzip.BestSpeed)
		if err != nil {
			return nil, err
		}
		defer writer.Close()

		writer.Write(data)
		writer.Flush()

		reader = bytes.NewReader(buffer.Bytes())

	} else {

		reader = bytes.NewReader(data)

	}

	return New(
		reader,
		sha256.Sum256(data),
		compression,
		uint32(len(data)),
		time.Now(),
	), nil

}

// Create a byte slice from an artifact. This will consume the artifact and
// apply a finalizer. Do not use the artifact after calling this function.
func ToBytes(artifact Artifact) ([]byte, error) {

	data := make([]byte, artifact.Size())

	if artifact.Compression() {

		reader, err := gzip.NewReader(artifact)
		if err != nil {
			artifact.Disconnect()
			return nil, err
		}
		defer reader.Close()

		_, err = io.ReadFull(reader, data)
		if err != nil {
			artifact.Disconnect()
			return nil, err
		}

	} else {

		_, err := io.ReadFull(artifact, data)
		if err != nil {
			artifact.Disconnect()
			return nil, err
		}

	}

	if sha256.Sum256(data) != artifact.Checksum() {
		artifact.Disconnect()
		return nil, errors.New("Cannot verify checksum of artifact")
	}

	artifact.Close()

	return data, nil

}
