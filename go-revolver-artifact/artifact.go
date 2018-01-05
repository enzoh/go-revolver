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

	"gx/ipfs/QmPbEVvboS8vFGwnesWYzKXNRH82p2gh3SMExNsAycwwe3/go-revolver-util"
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
		timestamp.UTC(),
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

// Encode the metadata of an artifact.
func EncodeMetadata(artifact Artifact) (metadata [45]byte) {

	checksum := artifact.Checksum()
	copy(metadata[00:], checksum[:])

	if artifact.Compression() {
		metadata[32] = 0x01
	}

	size := util.EncodeBigEndianUInt32(artifact.Size())
	copy(metadata[33:], size[:])

	timestamp := util.EncodeBigEndianInt64(artifact.Timestamp().UnixNano())
	copy(metadata[37:], timestamp[:])

	return metadata

}

// Decode the metadata of an artifact.
func DecodeMetadata(metadata [45]byte) (checksum [32]byte, compression bool, size uint32, timestamp time.Time) {

	var (
		buf4 [4]byte
		buf8 [8]byte
	)

	copy(checksum[:], metadata[00:])

	compression = metadata[32] > 0x00

	copy(buf4[:], metadata[33:])
	size = util.DecodeBigEndianUInt32(buf4)

	copy(buf8[:], metadata[37:])
	nanos := util.DecodeBigEndianInt64(buf8)
	timestamp = time.Unix(nanos/1000000000, nanos%1000000000).UTC()

	return

}
