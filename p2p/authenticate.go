/**
 * File        : authenticate.go
 * Description : Service for authenticating peers.
 * Copyright   : Copyright (c) 2017-2018 DFINITY Stiftung. All rights reserved.
 * Maintainer  : Enzo Haussecker <enzo@dfinity.org>
 * Stability   : Experimental
 */

package p2p

import (
	"time"

	"gx/ipfs/QmNa31VPzC561NWwRsJLE7nGYZYuuD2QfpK2b1q9BK54J1/go-libp2p-net"
	"gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"

	"github.com/dfinity/go-revolver/util"
)

// Authenticate a peer.
func (client *client) authenticate(peerId peer.ID) (bool, error) {

	// Prove eligibility to the target peer.
	pid := peerId
	client.logger.Debug("Proving eligibility to", pid)

	// Connect to the target peer.
	stream, err := client.host.NewStream(
		client.context,
		pid,
		client.protocol+"/authenticate",
	)
	if err != nil {
		addrs := client.peerstore.PeerInfo(pid).Addrs
		client.logger.Debug("Cannot connect to", pid, "at", addrs, err)
		client.peerstore.ClearAddrs(pid)
		client.table.Remove(pid)
		return false, err
	}
	defer stream.Close()

	// Receive the size of the challenge from the target peer.
	challengeInSize, err := util.ReadUInt32WithTimeout(
		stream,
		client.config.Timeout,
	)
	if err != nil {
		client.logger.Warning("Cannot receive challenge size from", pid, err)
		return false, err
	}

	// Check if the client can buffer the challenge.
	if challengeInSize > client.config.ChallengeMaxBufferSize {
		client.logger.Warningf("Cannot accept %d byte challenge from %v", challengeInSize, pid)
		return false, nil
	}

	// Receive the challenge from the target peer.
	challengeIn, err := util.ReadWithTimeout(
		stream,
		challengeInSize,
		client.config.Timeout,
	)
	if err != nil {
		client.logger.Warning("Cannot receive challenge from", pid, err)
		return false, err
	}

	// Request a zero-knowledge proof from the client.
	proofResponse := make(chan []byte, 1)
	client.proofRequests <- proofRequest{challengeIn, proofResponse}
	proofOut := <-proofResponse
	proofOutSize := util.EncodeBigEndianUInt32(uint32(len(proofOut)))
	close(proofResponse)

	// Send the zero-knowledge proof to the target peer.
	err = util.WriteWithTimeout(
		stream,
		append(proofOutSize[:], proofOut...),
		client.config.Timeout,
	)
	if err != nil {
		client.logger.Warning("Cannot send zero-knowledge proof to", pid, err)
		return false, err
	}

	// Verifying eligibility of the target peer.
	client.logger.Debug("Verifying eligibility of", pid)

	// Request a challenge from the client.
	challengeResponse := make(chan []byte, 1)
	client.challengeRequests <- challengeRequest{challengeResponse}
	challengeOut := <-challengeResponse
	challengeOutSize := util.EncodeBigEndianUInt32(uint32(len(challengeOut)))
	close(challengeResponse)

	// Send the challenge to the target peer.
	err = util.WriteWithTimeout(
		stream,
		append(challengeOutSize[:], challengeOut...),
		client.config.Timeout,
	)
	if err != nil {
		client.logger.Warning("Cannot send challenge to", pid, err)
		return false, err
	}

	// Receive the size of the zero-knowledge proof from the target peer.
	proofInSize, err := util.ReadUInt32WithTimeout(
		stream,
		client.config.Timeout,
	)
	if err != nil {
		client.logger.Warning("Cannot receive zero-knowledge proof size from", pid, err)
		return false, err
	}

	// Check if the client can buffer the zero-knowledge proof.
	if proofInSize > client.config.ProofMaxBufferSize {
		client.logger.Warningf("Cannot accept %d byte zero-knowledge proof from %v", proofInSize, pid)
		return false, nil
	}

	// Receive the zero-knowledge proof from the target peer.
	proofIn, err := util.ReadWithTimeout(
		stream,
		proofInSize,
		client.config.Timeout,
	)
	if err != nil {
		client.logger.Warning("Cannot receive zero-knowledge proof from", pid, err)
		return false, err
	}

	// Request a verification from the client.
	verificationResponse := make(chan bool, 1)
	client.verificationRequests <- verificationRequest{proofIn, verificationResponse}
	success := <-verificationResponse
	close(verificationResponse)

	// Done.
	return success, nil

}

// Handle incomming authentication requests.
func (client *client) authenticateHandler(stream net.Stream) {

	defer stream.Close()

	// Verifying eligibility of the target peer.
	pid := stream.Conn().RemotePeer()
	client.logger.Debug("Verifying eligibility of", pid)

	// Check if the request is spam.
	client.spammerCacheLock.Lock()
	timestamp, exists := client.spammerCache.Get(pid)
	if exists && time.Since(timestamp.(time.Time)) < 10*time.Minute {
		client.spammerCacheLock.Unlock()
		time.Sleep(client.config.Timeout)
		return
	}
	client.spammerCache.Add(pid, time.Now)
	client.spammerCacheLock.Unlock()

	// Request a challenge from the client.
	challengeResponse := make(chan []byte, 1)
	client.challengeRequests <- challengeRequest{challengeResponse}
	challengeOut := <-challengeResponse
	challengeOutSize := util.EncodeBigEndianUInt32(uint32(len(challengeOut)))
	close(challengeResponse)

	// Send the challenge to the target peer.
	err := util.WriteWithTimeout(
		stream,
		append(challengeOutSize[:], challengeOut...),
		client.config.Timeout,
	)
	if err != nil {
		client.logger.Warning("Cannot send challenge to", pid, err)
		return
	}

	// Receive the size of the zero-knowledge proof from the target peer.
	proofInSize, err := util.ReadUInt32WithTimeout(
		stream,
		client.config.Timeout,
	)
	if err != nil {
		client.logger.Warning("Cannot receive zero-knowledge proof size from", pid, err)
		return
	}

	// Check if the client can buffer the zero-knowledge proof.
	if proofInSize > client.config.ProofMaxBufferSize {
		client.logger.Warningf("Cannot accept %d byte zero-knowledge proof from %v", proofInSize, pid)
		return
	}

	// Receive the zero-knowledge proof from the target peer.
	proofIn, err := util.ReadWithTimeout(
		stream,
		proofInSize,
		client.config.Timeout,
	)
	if err != nil {
		client.logger.Warning("Cannot receive zero-knowledge proof from", pid, err)
		return
	}

	// Request a verification from the client.
	verificationResponse := make(chan bool, 1)
	client.verificationRequests <- verificationRequest{proofIn, verificationResponse}
	success := <-verificationResponse
	close(verificationResponse)

	// Check if the verification was successful.
	if !success {
		client.logger.Warning("Cannot verify zero-knowledge proof from", pid, err)
		return
	}

	// Prove eligibility to the target peer.
	client.logger.Debug("Proving eligibility to", pid)

	// Receive the size of the challenge from the target peer.
	challengeInSize, err := util.ReadUInt32WithTimeout(
		stream,
		client.config.Timeout,
	)
	if err != nil {
		client.logger.Warning("Cannot receive challenge size from", pid, err)
		return
	}

	// Check if the client can buffer the challenge.
	if challengeInSize > client.config.ChallengeMaxBufferSize {
		client.logger.Warningf("Cannot accept %d byte challenge from %v", challengeInSize, pid)
		return
	}

	// Receive the challenge from the target peer.
	challengeIn, err := util.ReadWithTimeout(
		stream,
		challengeInSize,
		client.config.Timeout,
	)
	if err != nil {
		client.logger.Warning("Cannot receive challenge from", pid, err)
		return
	}

	// Request a zero-knowledge proof from the client.
	proofResponse := make(chan []byte, 1)
	client.proofRequests <- proofRequest{challengeIn, proofResponse}
	proofOut := <-proofResponse
	proofOutSize := util.EncodeBigEndianUInt32(uint32(len(proofOut)))
	close(proofResponse)

	// Send the zero-knowledge proof to the target peer.
	err = util.WriteWithTimeout(
		stream,
		append(proofOutSize[:], proofOut...),
		client.config.Timeout,
	)
	if err != nil {
		client.logger.Warning("Cannot send zero-knowledge proof to", pid, err)
		return
	}

	return

}

// Register the authentication handler.
func (client *client) registerAuthenticationService() {
	uri := client.protocol + "/authenticate"
	client.host.SetStreamHandler(uri, client.authenticateHandler)
}
