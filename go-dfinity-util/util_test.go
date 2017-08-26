/**
 * File        : util_test.go
 * Description : Unit tests.
 * Copyright   : Copyright (c) 2017 DFINITY Stiftung. All rights reserved.
 * Maintainer  : Enzo Haussecker <enzo@string.technology>
 * Stability   : Stable
 */

package util

import (
	"bytes"
	"io"
	"math/rand"
	"testing"
	"time"
)

// Show that data can be written to and read from a pipe using a timeout.
func TestReadWriteWithTimeout(test *testing.T) {
	const n = 2048
	input := make([]byte, n)
	rand.Seed(time.Now().UnixNano())
	_, err := rand.Read(input)
	if err != nil {
		test.Fatal(err)
	}
	reader, writer := io.Pipe()
	defer reader.Close()
	defer writer.Close()
	go WriteWithTimeout(writer, input, time.Second)
	output, err := ReadWithTimeout(reader, n, time.Second)
	if err != nil {
		test.Fatal(err)
	}
	if !bytes.Equal(input, output) {
		test.Fatal("Corrupt data!")
	}
}

// Show that an unsigned 32-bit integer can be encoded and decoded using big-
// endian byte order.
func TestEncodeDecodeBigEndianUInt32(test *testing.T) {
	rand.Seed(time.Now().UnixNano())
	n := rand.Uint32()
	if DecodeBigEndianUInt32(EncodeBigEndianUInt32(n)) != n {
		test.Fatal(n)
	}
}

// Show that a signed 64-bit integer can be encoded and decoded using big-endian
// byte order.
func TestEncodeDecodeBigEndianInt64(test *testing.T) {
	rand.Seed(time.Now().UnixNano())
	n := rand.Int63()
	if DecodeBigEndianInt64(EncodeBigEndianInt64(n)) != n {
		test.Fatal(n)
	}
}
