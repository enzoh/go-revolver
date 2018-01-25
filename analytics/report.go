/**
 * File        : report.go
 * Description : Analytics reporting module.
 * Copyright   : Copyright (c) 2017-2018 DFINITY Stiftung. All rights reserved.
 * Maintainer  : Enzo Haussecker <enzo@dfinity.org>
 * Stability   : Stable
 *
 * This module provides a common type used by both the client and the network
 * topology inspector.
 */

package analytics

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"
)

type Report struct {
	Addrs     []string
	ClusterID int
	Network   string
	NodeID    string
	Peers     int
	ProcessID int
	Streams   []string
	Timestamp int64
	UserData  string
	Version   string
}

// Handle an analytics report.
func ReportHandler(reports map[string]Report, lock *sync.Mutex) http.HandlerFunc {

	return func(resp http.ResponseWriter, req *http.Request) {

		log.Println("Receiving analytics report ...")

		lock.Lock()
		defer lock.Unlock()

		decoder := json.NewDecoder(req.Body)
		defer req.Body.Close()

		var report Report
		err := decoder.Decode(&report)
		if err != nil {
			log.Printf("Cannot decode report: \033[1;31m%s\033[0m\n", err.Error())
			http.Error(resp, "400 Bad Request", http.StatusBadRequest)
			return
		}

		report.Timestamp = time.Now().Unix()
		reports[report.NodeID] = report

	}

}
