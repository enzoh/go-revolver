/**
 * File        : report.go
 * Description : Common analytics report type and related functions.
 * Copyright   : Copyright (c) 2017-2018 DFINITY Stiftung. All rights reserved.
 * Maintainer  : Enzo Haussecker <enzo@dfinity.org>
 * Stability   : Stable
 *
 * This module provides a common analytics report type and related functions
 * used by both the client and the network topology inspector.
 */

package report

import (
	"bytes"
	"encoding/json"
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

// Send a report to an analytics server.
func (report *Report) Send(url string) error {

	// Encode the report.
	data, err := json.Marshal(report)
	if err != nil {
		return err
	}

	// Create a request.
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	// Send it.
	sender := &http.Client{}
	resp, err := sender.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()

	// Done.
	return nil

}

// Process an analytics report.
func ReportHandler(reports map[string]*Report, lock *sync.Mutex) http.HandlerFunc {

	return func(resp http.ResponseWriter, req *http.Request) {

		lock.Lock()
		defer lock.Unlock()

		decoder := json.NewDecoder(req.Body)
		defer req.Body.Close()

		var report Report
		err := decoder.Decode(&report)
		if err != nil {
			http.Error(resp, "400 Bad Request", http.StatusBadRequest)
			return
		}

		report.Timestamp = time.Now().Unix()
		reports[report.NodeID] = &report

	}

}
