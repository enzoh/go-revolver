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

// Authenticate with a peer.
func (client *client) auth(peerId peer.ID) (bool, error) {

	// Log this action.
	pid := peerId
	client.logger.Debug("Authenticating with", pid)

	// Connect to the target peer.
	stream, err := client.host.NewStream(
		client.context,
		pid,
		client.protocol+"/auth",
	)
	if err != nil {
		addrs := client.peerstore.PeerInfo(pid).Addrs
		client.logger.Debug("Cannot connect to", pid, "at", addrs, err)
		client.peerstore.ClearAddrs(pid)
		client.table.Remove(pid)
		return false, err
	}
	defer stream.Close()

	// Request a commitment and send it.
	commitmentOut := client.requestCommitment()
	err = client.sendCommitment(stream, commitmentOut)
	if err != nil {
		return false, err
	}

	// Receive a challenge.
	challengeIn, err := client.receiveChallenge(stream)
	if err != nil {
		return false, err
	}

	// Request a zero-knowledge proof and send it.
	proofOut := client.requestProof(commitmentOut, challengeIn)
	err = client.sendProof(stream, proofOut)
	if err != nil {
		return false, err
	}

	// Receive a commitment.
	commitmentIn, err := client.receiveCommitment(stream)
	if err != nil {
		return false, err
	}

	// Request a challenge and send it.
	challengeOut := client.requestChallenge()
	err = client.sendChallenge(stream, challengeOut)
	if err != nil {
		return false, err
	}

	// Receive a zero-knowledge proof.
	proofIn, err := client.receiveProof(stream)
	if err != nil {
		return false, err
	}

	// Verify the zero-knowledge proof.
	success := client.requestVerification(
		commitmentIn,
		challengeOut,
		proofIn,
	)
	if success {
		client.peerstore.Put(pid, "AUTHORIZED", true)
		client.peerstore.Put(pid, "AUTHORIZED_AT", time.Now())
		client.table.Add(pid)
	}

	// Done.
	return success, nil

}

// Handle authentication requests.
func (client *client) authHandler(stream net.Stream) {

	defer stream.Close()

	// Log this action.
	pid := stream.Conn().RemotePeer()
	client.logger.Debug("Authenticating with", pid)

	// Receive a commitment.
	commitmentIn, err := client.receiveCommitment(stream)
	if err != nil {
		return
	}

	// Request a challenge and send it.
	challengeOut := client.requestChallenge()
	err = client.sendChallenge(stream, challengeOut)
	if err != nil {
		return
	}

	// Receive a zero-knowledge proof.
	proofIn, err := client.receiveProof(stream)
	if err != nil {
		return
	}

	// Verify the zero-knowledge proof.
	success := client.requestVerification(commitmentIn, challengeOut, proofIn)
	if !success {
		return
	}
	client.peerstore.Put(pid, "AUTHORIZED", true)
	client.peerstore.Put(pid, "AUTHORIZED_AT", time.Now())
	client.table.Add(pid)

	// Request a commitment and send it.
	commitmentOut := client.requestCommitment()
	err = client.sendCommitment(stream, commitmentOut)
	if err != nil {
		return
	}

	// Receive a challenge.
	challengeIn, err := client.receiveChallenge(stream)
	if err != nil {
		return
	}

	// Request a zero-knowledge proof and send it.
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
