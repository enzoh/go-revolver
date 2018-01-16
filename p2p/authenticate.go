/**
 * File        : authenticate.go
 * Description : Authentication module.
 * Copyright   : Copyright (c) 2017-2018 DFINITY Stiftung. All rights reserved.
 * Maintainer  : Enzo Haussecker <enzo@dfinity.org>
 * Stability   : Experimental
 */

package p2p

import (
	"errors"
	"time"

	"gx/ipfs/QmNa31VPzC561NWwRsJLE7nGYZYuuD2QfpK2b1q9BK54J1/go-libp2p-net"
	"gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"

	"github.com/dfinity/go-revolver/util"
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

	challenge, err := client.recvChallenge(stream)
	if err != nil {
		return false, err
	}

	err = client.sendZKP(stream, client.genZKP(challenge))
	if err != nil {
		return false, err
	}

	client.logger.Debug("verifying eligibility of", pid)

	err = client.sendChallenge(stream, client.genChallenge())
	if err != nil {
		return false, err
	}

	proof, err := client.recvZKP(stream)
	if err != nil {
		return false, err
	}

	success := client.genVerification(proof)

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

	err := client.sendChallenge(stream, client.genChallenge())
	if err != nil {
		return
	}

	proof, err := client.recvZKP(stream)
	if err != nil {
		return
	}

	success := client.genVerification(proof)
	if !success {
		return
	}

	client.logger.Debug("Proving eligibility to", pid)

	challenge, err := client.recvChallenge(stream)
	if err != nil {
		return
	}

	err = client.sendZKP(stream, client.genZKP(challenge))
	if err != nil {
		return
	}

}

// Register the authentication handler.
func (client *client) registerAuthService() {
	uri := client.protocol + "/auth"
	client.host.SetStreamHandler(uri, client.authHandler)
}

// Generate a challenge.
func (client *client) genChallenge() []byte {

	response := make(chan []byte, 1)
	client.challengeRequests <- challengeRequest{response}
	challenge := <-response
	close(response)

	return challenge

}

// Send a challenge.
func (client *client) sendChallenge(stream net.Stream, challenge []byte) error {

	size := util.EncodeBigEndianUInt32(uint32(len(challenge)))

	err := util.WriteWithTimeout(
		stream,
		append(size[:], challenge...),
		client.config.Timeout,
	)
	if err != nil {
		pid := stream.Conn().RemotePeer()
		client.logger.Debug("cannot send challenge to", pid, err)
		return err
	}

	return nil

}

// Receive a challenge.
func (client *client) recvChallenge(stream net.Stream) ([]byte, error) {

	size, err := util.ReadUInt32WithTimeout(
		stream,
		client.config.Timeout,
	)
	if err != nil {
		pid := stream.Conn().RemotePeer()
		client.logger.Debug("cannot receive challenge size from", pid, err)
		return nil, err
	}

	if size > client.config.ChallengeMaxBufferSize {
		pid := stream.Conn().RemotePeer()
		client.logger.Debugf("cannot accept %d byte challenge from %v", size, pid)
		return nil, errors.New("challenge exceeds maximum buffer size")
	}

	challenge, err := util.ReadWithTimeout(
		stream,
		size,
		client.config.Timeout,
	)
	if err != nil {
		pid := stream.Conn().RemotePeer()
		client.logger.Debug("cannot receive challenge from", pid, err)
		return nil, err
	}

	return challenge, nil

}

// Generate a zero-knowledge proof.
func (client *client) genZKP(challenge []byte) []byte {

	response := make(chan []byte, 1)
	client.proofRequests <- proofRequest{challenge, response}
	proof := <-response
	close(response)

	return proof

}

// Send a zero-knowledge proof.
func (client *client) sendZKP(stream net.Stream, proof []byte) error {

	size := util.EncodeBigEndianUInt32(uint32(len(proof)))

	err := util.WriteWithTimeout(
		stream,
		append(size[:], proof...),
		client.config.Timeout,
	)
	if err != nil {
		pid := stream.Conn().RemotePeer()
		client.logger.Debug("cannot send zero-knowledge proof to", pid, err)
		return err
	}

	return nil

}

// Receive a zero-knowledge proof.
func (client *client) recvZKP(stream net.Stream) ([]byte, error) {

	size, err := util.ReadUInt32WithTimeout(
		stream,
		client.config.Timeout,
	)
	if err != nil {
		pid := stream.Conn().RemotePeer()
		client.logger.Debug("cannot receive zero-knowledge proof size from", pid, err)
		return nil, err
	}

	if size > client.config.ProofMaxBufferSize {
		pid := stream.Conn().RemotePeer()
		client.logger.Debugf("cannot accept %d byte zero-knowledge proof from %v", size, pid)
		return nil, errors.New("zero-knowledge proof exceeds maximum buffer size")
	}

	proof, err := util.ReadWithTimeout(
		stream,
		size,
		client.config.Timeout,
	)
	if err != nil {
		pid := stream.Conn().RemotePeer()
		client.logger.Debug("cannot receive zero-knowledge proof from", pid, err)
		return nil, err
	}

	return proof, nil

}

// Verify a zero knowledge proof.
func (client *client) genVerification(proof []byte) bool {

	response := make(chan bool, 1)
	client.verificationRequests <- verificationRequest{proof, response}
	success := <-response
	close(response)

	return success

}
