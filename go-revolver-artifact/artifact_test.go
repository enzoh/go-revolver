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

// Show that we can create an artifact from a byte slice.
func TestFromToBytes(test *testing.T) {
	from := []byte("This is a test.")
	artifact, err := FromBytes(from, true)
	if err != nil {
		test.Fatal(err)
	}
	to, err := ToBytes(artifact)
	if err != nil {
		test.Fatal(err)
	}
	if !bytes.Equal(from, to) {
		test.Fatal("Corrupt artifact!")
	}
}
