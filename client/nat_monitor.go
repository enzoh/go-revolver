/**
 * File        : nat_monitor.go
 * Description : NAT monitoring module.
 * Copyright   : Copyright (c) 2017 DFINITY Stiftung. All rights reserved.
 * Maintainer  : Enzo Haussecker <enzo@string.technology>
 * Stability   : Experimental
 */

package p2p

import (
	"time"

	"github.com/libp2p/go-libp2p-peerstore"
	"github.com/libp2p/go-libp2p/p2p/host/basic"
	"github.com/multiformats/go-multiaddr"
)

// Monitor port mapping in NAT devices.
func (client *client) newNATMonitor(listener multiaddr.Multiaddr, manager basichost.NATManager) func() {

	// Create a shutdown function.
	notify := make(chan struct{}, 1)
	shutdown := func() {
		notify <- struct{}{}
	}

	go func(listener multiaddr.Multiaddr, manager basichost.NATManager) {
		select {
		case <-time.After(client.config.NATMonitorTimeout):
			client.logger.Warning("Failed to locate NAT device")
		case <-manager.Ready():
			nat := manager.NAT()
			addr := listener
		NATMonitor:
			for {
				select {
				case <-notify:
					break NATMonitor
				default:
				}
				addrs := nat.MappedAddrs()
				for key, value := range addrs {
					if key.Equal(listener) && value != nil && !addr.Equal(value) {
						client.logger.Infof("I am %s/ipfs/%s", value, client.id.Pretty())
						client.peerstore.AddAddr(client.id, value, peerstore.PermanentAddrTTL)
						addr = value
					}
				}
				time.Sleep(client.config.NATMonitorIterationInterval)
			}
		}
	}(listener, manager)

	// Return the shutdown function.
	return shutdown

}
