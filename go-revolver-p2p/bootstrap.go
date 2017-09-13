/**
 * File        : bootstrap.go
 * Description : Client bootstrapping module.
 * Copyright   : Copyright (c) 2017 DFINITY Stiftung. All rights reserved.
 * Maintainer  : Enzo Haussecker <enzo@string.technology>
 * Stability   : Experimental
 */

package p2p

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"gx/ipfs/QmPgDWmTmuzvP7QE5zwo1TmjbJme9pmZHNujB2453jkCTr/go-libp2p-peerstore"
	"gx/ipfs/QmVJefKHXEx28RFpmj5GeRg43AqeBH3npPwvgJ875fBPm7/go-libp2p-swarm"
	"gx/ipfs/QmXY77cVe7rVRQXZZQRioukUM7aRW3BTcAgJe12MCtb3Ji/go-multiaddr"
	"gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"
	"gx/ipfs/QmZyngpQxUGyx1T2bzEcst6YzERkvVwDzBMbsSQF4f1smE/go-libp2p/p2p/host/basic"
)

// Bootstrap a client.
func (client *client) bootstrap() (func(), error) {

	// Create an address to listen on.
	listener, err := multiaddr.NewMultiaddr(
		fmt.Sprintf(
			"/ip%d/%s/tcp/%d",
			ipVersion(client.config.IP),
			client.config.IP,
			client.config.Port,
		),
	)
	if err != nil {
		return nil, err
	}

	// Check if the client can listen on its address.
	conn, err := net.Dial(
		"tcp",
		fmt.Sprintf(
			"%s:%d",
			client.config.IP,
			client.config.Port,
		),
	)
	if err == nil {
		conn.Close()
		return nil, errors.New("Address already in use")
	}

	// Create a network to be used by the service host.
	network, err := swarm.NewNetwork(
		client.context,
		[]multiaddr.Multiaddr{listener},
		client.id,
		client.peerstore,
		nil,
	)
	if err != nil {
		return nil, err
	}

	// Create a service host.
	options := &basichost.HostOpts{}
	shutdownNATMonitor := func() {}
	if !client.config.DisableNATPortMap {
		options.NATManager = basichost.NewNATManager(network)
		shutdownNATMonitor = client.newNATMonitor(
			listener,
			options.NATManager,
		)
	}
	client.host, err = basichost.NewHost(client.context, network, options)
	if err != nil {
		return nil, err
	}

	// Add the addresses used by the service host to the peer store.
	addresses := client.host.Addrs()
	for i := range addresses {
		address := addresses[i]
		client.logger.Infof(
			"I am %s/ipfs/%s",
			address,
			client.id.Pretty(),
		)
		client.peerstore.AddAddr(
			client.id,
			address,
			peerstore.PermanentAddrTTL,
		)
	}

	// Register services.
	client.registerPairService()
	client.registerPingService()
	client.registerSampleService()

	// Greet the seed nodes.
	var group sync.WaitGroup
	for _, address := range client.config.SeedNodes {
		group.Add(1)
		go func(address multiaddr.Multiaddr) {
			defer group.Done()
			err := client.hello(address)
			if err != nil {
				client.logger.Error("Cannot connect to seed node", address, err)
			}
		}(address)
	}
	group.Wait()

	// Discover peers.
	shutdownPeerDiscovery := func() {}
	if !client.config.DisablePeerDiscovery {
		shutdownPeerDiscovery = client.discoverPeers()
	}

	// Discover streams.
	shutdownStreamDiscovery := func() {}
	if !client.config.DisableStreamDiscovery {
		shutdownStreamDiscovery = client.discoverStreams()
	}

	// Broadcast artifacts.
	shutdownBroadcast := func() {}
	if !client.config.DisableBroadcast {
		shutdownBroadcast = client.activateBroadcast()
	}

	// Share analytics with core developers.
	shutdownAnalytics := func() {}
	if !client.config.DisableAnalytics {
		shutdownAnalytics = client.activateAnalytics()
	}

	// Create a shutdown function.
	shutdown := func() {
		shutdownAnalytics()
		shutdownBroadcast()
		shutdownNATMonitor()
		shutdownPeerDiscovery()
		shutdownStreamDiscovery()
		client.host.Close()
	}

	// Return the shutdown function.
	return shutdown, nil

}

// Detect the version of an IP address.
func ipVersion(ip string) int {

	// Match the first IPv4 or IPv6-specific character.
	for i := 0; i < len(ip); i++ {
		switch ip[i] {
		case '.':
			return 4
		case ':':
			return 6
		}
	}

	// Unknown.
	return 0

}

// Greet a seed node.
func (client *client) hello(seedAddress multiaddr.Multiaddr) error {

	// Parse the seed address.
	seedAddr, seedId, err := parseIPFSAddress(seedAddress)
	if err != nil {
		return err
	}

	// Prevent self dialing.
	if client.id == seedId {
		return nil
	}

	// Ping the seed node.
	for i := 0; i < 5; i++ {
		client.peerstore.AddAddr(
			seedId,
			seedAddr,
			peerstore.TempAddrTTL,
		)
		err = client.ping(seedId)
		if err != nil {
			time.Sleep(time.Second)
			continue
		}
		break
	}
	if err != nil {
		return err
	}

	// Permanently add the seed node to the peer store.
	client.peerstore.SetAddr(
		seedId,
		seedAddr,
		peerstore.PermanentAddrTTL,
	)

	// Update the routing table.
	client.table.Update(seedId)

	// Success.
	return nil

}

// Parse an IPFS address.
func parseIPFSAddress(address multiaddr.Multiaddr) (multiaddr.Multiaddr, peer.ID, error) {

	// Get the prefix.
	addr, err := multiaddr.NewMultiaddr(
		strings.Split(address.String(), "/ipfs/")[0],
	)
	if err != nil {
		return nil, "", err
	}

	// Get the suffix.
	b58, err := address.ValueForProtocol(multiaddr.P_IPFS)
	if err != nil {
		return nil, "", err
	}

	// Decode the identifier.
	id, err := peer.IDB58Decode(b58)
	if err != nil {
		return nil, "", err
	}

	// Return the prefix and identifier.
	return addr, id, nil

}
