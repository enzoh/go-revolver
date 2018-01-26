/**
 * File        : main_test.go
 * Description : Unit tests.
 * Copyright   : Copyright (c) 2017-2018 DFINITY Stiftung. All rights reserved.
 * Maintainer  : Enzo Haussecker <enzo@dfinity.org>
 * Stability   : Experimental
 */

package streamstore

import (
	"os"
	"testing"

	"github.com/dfinity/go-logging"
)

// Run the unit tests.
func TestMain(m *testing.M) {

	// Format the log output.
	backend := logging.NewLogBackend(os.Stdout, "", 0)
	format := "\033[0;37m%{time:15:04:05.000} %{color}%{level} \033[0;34m[%{module}] \033[0m%{message} \033[0;37m%{shortfile}\033[0m"
	logging.SetBackend(logging.AddModuleLevel(backend))
	logging.SetFormatter(logging.MustStringFormatter(format))

	// Set the log level.
	logging.SetLevel(logging.DEBUG, "p2p")
	logging.SetLevel(logging.DEBUG, "streamstore")

	// Run the unit tests.
	os.Exit(m.Run())
}
