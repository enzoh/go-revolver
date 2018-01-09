/**
 * File        : artifact_test.go
 * Description : Unit tests.
 * Copyright   : Copyright (c) 2017-2018 DFINITY Stiftung. All rights reserved.
 * Maintainer  : Enzo Haussecker <enzo@dfinity.org>
 * Stability   : Stable
 */

package artifact

import (
	"bytes"
	"testing"
)

// Show that we can create an artifact from a byte slice and consume it.
func TestFromToBytes(test *testing.T) {

	dataOut := []byte("This is a test.")

	artifact, err := FromBytes(dataOut, true)
	if err != nil {
		test.Fatal(err)
	}

	dataIn, err := ToBytes(artifact)
	if err != nil {
		test.Fatal(err)
	}

	if !bytes.Equal(dataOut, dataIn) {
		test.Fatal("Unexpected artifact!", dataIn)
	}

}

// Show that we can encode and decode the metadata of an artifact.
func TestEncodeDecodeMetadata(test *testing.T) {

	data := []byte("This is a test.")

	artifact, err := FromBytes(data, true)
	if err != nil {
		test.Fatal(err)
	}

	metadata := EncodeMetadata(artifact)

	checksum, compression, size, timestamp := DecodeMetadata(metadata)

	if artifact.Checksum() != checksum {
		test.Fatal("Unexpected checksum!", checksum)
	}

	if artifact.Compression() != compression {
		test.Fatal("Unexpected compression!", compression)
	}

	if artifact.Size() != size {
		test.Fatal("Unexpected size!", size)
	}

	if artifact.Timestamp() != timestamp {
		test.Fatal("Unexpected timestamp!", timestamp)
	}

}
