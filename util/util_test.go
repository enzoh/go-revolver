/**
 * File        : util_test.go
 * Description : Unit tests.
 * Copyright   : Copyright (c) 2017-2018 DFINITY Stiftung. All rights reserved.
 * Maintainer  : Enzo Haussecker <enzo@dfinity.org>
 * Stability   : Stable
 */

package util

import (
	"bytes"
	"crypto/rand"
	"io"
	"math"
	"math/big"
	"testing"
	"time"
)

// Show that data can be written to and read from a pipe using a timeout.
func TestReadWriteWithTimeout(test *testing.T) {
	const n = 2048
	input := make([]byte, n)
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
	for i := 0; i < 100; i++ {
		u := big.NewInt(math.MaxUint32)
		u.Add(u, big.NewInt(1))
		v, err := rand.Int(rand.Reader, u)
		if err != nil {
			test.Fatal(err)
		}
		n := uint32(v.Int64())
		if DecodeBigEndianUInt32(EncodeBigEndianUInt32(n)) != n {
			test.Fatal(n)
		}
	}
}

// Show that a signed 64-bit integer can be encoded and decoded using big-endian
// byte order.
func TestEncodeDecodeBigEndianInt64(test *testing.T) {
	for i := 0; i < 100; i++ {
		u := big.NewInt(math.MaxInt64)
		u.Add(u, big.NewInt(1))
		u.Sub(u, big.NewInt(math.MinInt64))
		v, err := rand.Int(rand.Reader, u)
		if err != nil {
			test.Fatal(err)
		}
		v.Add(v, big.NewInt(math.MinInt64))
		n := v.Int64()
		if DecodeBigEndianInt64(EncodeBigEndianInt64(n)) != n {
			test.Fatal(n)
		}
	}
}
