/**
 * File        : challenge.go
 * Description : Authentication submodule.
 * Copyright   : Copyright (c) 2017-2018 DFINITY Stiftung. All rights reserved.
 * Maintainer  : Enzo Haussecker <enzo@dfinity.org>
 * Stability   : Experimental
 */

package p2p

import (
	"errors"

	"gx/ipfs/QmNa31VPzC561NWwRsJLE7nGYZYuuD2QfpK2b1q9BK54J1/go-libp2p-net"

	"github.com/dfinity/go-revolver/util"
)

// This type represents a function that executes when receiving a challenge
// request.
type ChallengeHandler func(response chan []byte)

// This type provides the data needed to request a challenge.
type challengeRequest struct {
	response chan []byte
}

// Register a challenge request handler.
func (client *client) setChallengeHandler(handler ChallengeHandler) {

	notify := make(chan struct{})
	client.unsetChallengeHandler = func() {
		close(notify)
	}

	go func() {
		for {
			select {
			case <-notify:
				return
			case request := <-client.challengeRequests:
				handler(request.response)
			}
		}
	}()

}

// Handle a challenge request.
func DefaultChallengeHandler(response chan []byte) {
	response <- nil
}

// Request a challenge.
func (client *client) requestChallenge() []byte {

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
		client.logger.Debug("Cannot send challenge to", pid, err)
		return err
	}

	return nil

}

// Receive a challenge.
func (client *client) receiveChallenge(stream net.Stream) ([]byte, error) {

	size, err := util.ReadUInt32WithTimeout(
		stream,
		client.config.Timeout,
	)
	if err != nil {
		pid := stream.Conn().RemotePeer()
		client.logger.Debug("Cannot receive challenge size from", pid, err)
		return nil, err
	}

	if size > client.config.ChallengeMaxBufferSize {
		pid := stream.Conn().RemotePeer()
		client.logger.Debugf("Cannot accept %d byte challenge from %v", size, pid)
		return nil, errors.New("Challenge exceeds maximum buffer size")
	}

	challenge, err := util.ReadWithTimeout(
		stream,
		size,
		client.config.Timeout,
	)
	if err != nil {
		pid := stream.Conn().RemotePeer()
		client.logger.Debug("Cannot receive challenge from", pid, err)
		return nil, err
	}

	return challenge, nil

}
