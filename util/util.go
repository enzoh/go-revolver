/**
 * File        : util.go
 * Description : Miscellaneous functions.
 * Copyright   : Copyright (c) 2017 DFINITY Stiftung. All rights reserved.
 * Maintainer  : Enzo Haussecker <enzo@string.technology>
 * Stability   : Stable
 */

package util

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"time"
)

// Write data to a stream using a timeout.
func WriteWithTimeout(writer io.Writer, data []byte, timeout time.Duration) error {
	result := make(chan error, 1)
	go func(writer io.Writer, data []byte) {
		_, err := writer.Write(data)
		result <-err
	}(writer, data)
	select {
	case err := <-result:
		return err
	case <-time.After(timeout):
		select {
		case result <-errors.New("Timeout!"):
		default:
		}
		err := <-result
		return err
	}
}

// Read data from a stream using a timeout.
func ReadWithTimeout(reader io.Reader, n uint32, timeout time.Duration) ([]byte, error) {
	data := make([]byte, n)
	result := make(chan error, 1)
	go func(reader io.Reader) {
		_, err := io.ReadFull(reader, data)
		result <-err
	}(reader)
	select {
	case err := <-result:
		return data, err
	case <-time.After(timeout):
		select {
		case result <-errors.New("Timeout!"):
		default:
		}
		err := <-result
		return data, err
	}
}

// Read an unsigned 32-bit integer from a stream using a timeout.
func ReadUInt32WithTimeout(reader io.Reader, timeout time.Duration) (uint32, error) {
	var arr [4]byte
	data, err := ReadWithTimeout(reader, 4, timeout)
	if err != nil {
		return 0, nil
	}
	copy(arr[:], data)
	n := DecodeBigEndianUInt32(arr)
	return n, nil
}

// Encode an unsigned 32-bit integer using big-endian byte order.
func EncodeBigEndianUInt32(n uint32) (data [4]byte) {
	var buf bytes.Buffer
	writer := bufio.NewWriter(&buf)
	binary.Write(writer, binary.BigEndian, &n)
	writer.Flush()
	copy(data[:], buf.Bytes())
	return
}

// Decode an unsigned 32-bit integer using big-endian byte order.
func DecodeBigEndianUInt32(data [4]byte) (n uint32) {
	reader := bytes.NewReader(data[:])
	binary.Read(reader, binary.BigEndian, &n)
	return
}
