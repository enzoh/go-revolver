/**
 * File        : analytics.go
 * Description : Analytics reporting module.
 * Copyright   : Copyright (c) 2017 DFINITY Stiftung. All rights reserved.
 * Maintainer  : Enzo Haussecker <enzo@string.technology>
 * Stability   : Experimental
 */

package p2p

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"
)

// Share analytics with core developers.
func (client *client) activateAnalytics() func() {

	// Create a shutdown function.
	notify := make(chan struct{})
	shutdown := func() {
		close(notify)
	}

	go func() {

		type Report struct {
			Addrs     []string
			ClusterID int
			Network   string
			NodeID    string
			Peers     int
			ProcessID int
			Streams   []string
			Timestamp int64
			Version   string
		}

		for {

			// Check for a shutdown instruction.
			select {
			case <-notify:
				return
			default:
			}

			// Create a report.
			report := Report{
				ClusterID: client.config.ClusterID,
				Network:   string(client.config.Network),
				NodeID:    client.id.Pretty(),
				Peers:     client.table.Size(),
				ProcessID: client.config.ProcessID,
				Timestamp: time.Now().Unix(),
				Version:   string(client.config.Version),
			}
			for _, addr := range client.host.Addrs() {
				report.Addrs = append(report.Addrs, addr.String())
			}
			for _, stream := range client.streamstore.Peers() {
				report.Streams = append(report.Streams, stream.Pretty())
			}

			// Encode it.
			data, err := json.Marshal(&report)
			if err != nil {
				client.logger.Warning("Cannot encode report", err)
				return
			}

			// Create a request.
			req, err := http.NewRequest(
				"POST",
				client.config.AnalyticsURL,
				bytes.NewBuffer(data),
			)
			if err != nil {
				client.logger.Warning("Cannot create request", err)
				return
			}
			req.Header.Set("Content-Type", "application/json")

			// Send it.
			sender := &http.Client{}
			resp, err := sender.Do(req)
			if err != nil {
				client.logger.Warning("Cannot send report", err)
			} else {
				resp.Body.Close()
			}

			// Wait.
			time.Sleep(client.config.AnalyticsIterationInterval)

		}

	}()

	// Return the shutdown function.
	return shutdown

}
