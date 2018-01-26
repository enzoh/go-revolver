/**
 * File        : analytics.go
 * Description : Analytics reporting module.
 * Copyright   : Copyright (c) 2017-2018 DFINITY Stiftung. All rights reserved.
 * Maintainer  : Enzo Haussecker <enzo@dfinity.org>
 * Stability   : Experimental
 */

package p2p

import (
	"time"

	"github.com/dfinity/go-revolver/report"
)

// Share analytics with core developers.
func (client *client) activateAnalytics() func() {

	// Create a shutdown function.
	notify := make(chan struct{})
	shutdown := func() {
		close(notify)
	}

	go func() {

		for {

			// Check for a shutdown notification.
			select {
			case <-notify:
				return
			default:
			}

			// Send an analytics report.
			err := client.genReport().Send(client.config.AnalyticsURL)
			if err != nil {
				client.logger.Warning("Cannot send analytics report", err)
			}

			// Wait.
			time.Sleep(client.config.AnalyticsInterval)

		}

	}()

	// Return the shutdown function.
	return shutdown

}

// Generate an analytics report.
func (client *client) genReport() *report.Report {

	// Create a report.
	report := report.Report{
		ClusterID: client.config.ClusterID,
		Network:   string(client.config.Network),
		NodeID:    client.id.Pretty(),
		Peers:     client.table.Size(),
		ProcessID: client.config.ProcessID,
		Timestamp: time.Now().Unix(),
		UserData:  client.config.AnalyticsUserData,
		Version:   string(client.config.Version),
	}

	// Add the addresses.
	for _, addr := range client.host.Addrs() {
		report.Addrs = append(report.Addrs, addr.String())
	}

	// Add the streams.
	for _, stream := range client.streamstore.Peers() {
		report.Streams = append(report.Streams, stream.Pretty())
	}

	// Done.
	return &report

}
