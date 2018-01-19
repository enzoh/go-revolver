/**
 * File        : authenticate.go
 * Description : Authentication module.
 * Copyright   : Copyright (c) 2017-2018 DFINITY Stiftung. All rights reserved.
 * Maintainer  : Enzo Haussecker <enzo@dfinity.org>
 * Stability   : Experimental
 */

package p2p

import (
	"time"

	"gx/ipfs/QmNa31VPzC561NWwRsJLE7nGYZYuuD2QfpK2b1q9BK54J1/go-libp2p-net"
	"gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"
)

// Authenticate a peer.
func (client *client) auth(peerId peer.ID) (bool, error) {

	pid := peerId
	client.logger.Debug("proving eligibility to", pid)

	stream, err := client.host.NewStream(
		client.context,
		pid,
		client.protocol+"/auth",
	)
	if err != nil {
		addrs := client.peerstore.PeerInfo(pid).Addrs
		client.logger.Debug("cannot connect to", pid, "at", addrs, err)
		client.peerstore.ClearAddrs(pid)
		client.table.Remove(pid)
		return false, err
	}
	defer stream.Close()

	commitmentOut := client.requestCommitment()
	err = client.sendCommitment(stream, commitmentOut)
	if err != nil {
		return false, err
	}

	challengeIn, err := client.receiveChallenge(stream)
	if err != nil {
		return false, err
	}

	proofOut := client.requestProof(commitmentOut, challengeIn)
	err = client.sendProof(stream, proofOut)
	if err != nil {
		return false, err
	}

	client.logger.Debug("verifying eligibility of", pid)

	commitmentIn, err := client.receiveCommitment(stream)
	if err != nil {
		return false, err
	}

	challengeOut := client.requestChallenge()
	err = client.sendChallenge(stream, challengeOut)
	if err != nil {
		return false, err
	}

	proofIn, err := client.receiveProof(stream)
	if err != nil {
		return false, err
	}

	success := client.requestVerification(commitmentIn, challengeOut, proofIn)

	return success, nil

}

// Handle authentication requests.
func (client *client) authHandler(stream net.Stream) {

	defer stream.Close()

	pid := stream.Conn().RemotePeer()
	client.logger.Debug("verifying eligibility of", pid)

	client.spammerCacheLock.Lock()
	timestamp, exists := client.spammerCache.Get(pid)
	if exists && time.Since(timestamp.(time.Time)) < 10*time.Minute {
		client.spammerCacheLock.Unlock()
		time.Sleep(client.config.Timeout)
		return
	}
	client.spammerCache.Add(pid, time.Now)
	client.spammerCacheLock.Unlock()

	commitmentIn, err := client.receiveCommitment(stream)
	if err != nil {
		return
	}

	challengeOut := client.requestChallenge()
	err = client.sendChallenge(stream, challengeOut)
	if err != nil {
		return
	}

	proofIn, err := client.receiveProof(stream)
	if err != nil {
		return
	}

	success := client.requestVerification(commitmentIn, challengeOut, proofIn)
	if !success {
		return
	}

	client.logger.Debug("Proving eligibility to", pid)

	commitmentOut := client.requestCommitment()
	err = client.sendCommitment(stream, commitmentOut)
	if err != nil {
		return
	}

	challengeIn, err := client.receiveChallenge(stream)
	if err != nil {
		return
	}

	proofOut := client.requestProof(commitmentOut, challengeIn)
	err = client.sendProof(stream, proofOut)
	if err != nil {
		return
	}

}

// Register the authentication handler.
func (client *client) registerAuthService() {
	uri := client.protocol + "/auth"
	client.host.SetStreamHandler(uri, client.authHandler)
}
