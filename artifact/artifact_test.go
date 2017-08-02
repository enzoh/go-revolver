/**
 * File        : artifact_test.go
 * Description : Unit tests.
 * Copyright   : Copyright (c) 2017 DFINITY Stiftung. All rights reserved.
 * Maintainer  : Enzo Haussecker <enzo@string.technology>
 * Stability   : Stable
 */

package artifact

import (
	"bytes"
	"math/rand"
	"testing"
)

// Show that we can create an artifact from a byte slice.
func TestFromToBytes(test *testing.T) {
	data1 := make([]byte, 32)
	_, err := rand.Read(data1)
	if err != nil {
		test.Fatal(err)
	}
	data2, err := ToBytes(FromBytes(data1))
	if err != nil {
		test.Fatal(err)
	}
	if !bytes.Equal(data1, data2) {
		test.Fatal("Corrupt artifact!")
	}
}
