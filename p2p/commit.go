/**
 * File        : commit.go
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

// This type represents a function that executes when receiving a commitment
// request. The function can be registered as a callback using
// SetCommitmentHandler.
type CommitmentHandler func(response chan []byte)

// This type provides the data needed to request a commitment.
type commitmentRequest struct {
	response chan []byte
}

// Register a commitment request handler.
func (client *client) SetCommitmentHandler(handler CommitmentHandler) {

	notify := make(chan struct{})

	client.unsetHandlerLock.Lock()
	client.unsetCommitmentHandler()
	client.unsetCommitmentHandler = func() {
		close(notify)
	}
	client.unsetHandlerLock.Unlock()

	go func() {
		for {
			select {
			case <-notify:
				return
			case request := <-client.commitmentRequests:
				handler(request.response)
			}
		}
	}()

}

// Request a commitment.
func (client *client) requestCommitment() []byte {

	response := make(chan []byte, 1)
	client.commitmentRequests <- commitmentRequest{response}
	commitment := <-response
	close(response)

	return commitment

}

// Send a commitment.
func (client *client) sendCommitment(stream net.Stream, commitment []byte) error {

	size := util.EncodeBigEndianUInt32(uint32(len(commitment)))

	err := util.WriteWithTimeout(
		stream,
		append(size[:], commitment...),
		client.config.Timeout,
	)
	if err != nil {
		pid := stream.Conn().RemotePeer()
		client.logger.Debug("cannot send commitment to", pid, err)
		return err
	}

	return nil

}

// Receive a commitment.
func (client *client) receiveCommitment(stream net.Stream) ([]byte, error) {

	size, err := util.ReadUInt32WithTimeout(
		stream,
		client.config.Timeout,
	)
	if err != nil {
		pid := stream.Conn().RemotePeer()
		client.logger.Debug("cannot receive commitment size from", pid, err)
		return nil, err
	}

	if size > client.config.CommitmentMaxBufferSize {
		pid := stream.Conn().RemotePeer()
		client.logger.Debugf("cannot accept %d byte commitment from %v", size, pid)
		return nil, errors.New("commitment exceeds maximum buffer size")
	}

	commitment, err := util.ReadWithTimeout(
		stream,
		size,
		client.config.Timeout,
	)
	if err != nil {
		pid := stream.Conn().RemotePeer()
		client.logger.Debug("cannot receive commitment from", pid, err)
		return nil, err
	}

	return commitment, nil

}
