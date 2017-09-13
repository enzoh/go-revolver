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

	"gx/ipfs/QmPgDWmTmuzvP7QE5zwo1TmjbJme9pmZHNujB2453jkCTr/go-libp2p-peerstore"
	"gx/ipfs/QmXY77cVe7rVRQXZZQRioukUM7aRW3BTcAgJe12MCtb3Ji/go-multiaddr"
	"gx/ipfs/QmZyngpQxUGyx1T2bzEcst6YzERkvVwDzBMbsSQF4f1smE/go-libp2p/p2p/host/basic"
)

// Monitor port mapping in NAT devices.
func (client *client) newNATMonitor(listener multiaddr.Multiaddr, manager basichost.NATManager) func() {

	// Create a shutdown function.
	notify := make(chan struct{})
	shutdown := func() {
		close(notify)
	}

	go func(listener multiaddr.Multiaddr, manager basichost.NATManager) {
		select {
		case <-time.After(client.config.NATMonitorTimeout):
			client.logger.Warning("Failed to locate NAT device")
		case <-manager.Ready():
			nat := manager.NAT()
			addr := listener
			for {
				select {
				case <-notify:
					return
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
				time.Sleep(client.config.NATMonitorInterval)
			}
		}
	}(listener, manager)

	// Return the shutdown function.
	return shutdown

}
