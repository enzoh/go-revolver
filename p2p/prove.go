/**
 * File        : prove.go
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

// This type represents a function that executes when receiving a zero-knowledge
// proof request. The function can be registered as a callback using
// SetProofHandler.
type ProofHandler func(commitment []byte, challenge []byte, response chan []byte)

// This type provides the data needed to request a zero-knowledge proof.
type proofRequest struct {
	commitment []byte
	challenge  []byte
	response   chan []byte
}

// Register a zero-knowledge proof request handler.
func (client *client) SetProofHandler(handler ProofHandler) {

	notify := make(chan struct{})

	client.unsetHandlerLock.Lock()
	client.unsetProofHandler()
	client.unsetProofHandler = func() {
		close(notify)
	}
	client.unsetHandlerLock.Unlock()

	go func() {
		for {
			select {
			case <-notify:
				return
			case request := <-client.proofRequests:
				handler(request.commitment, request.challenge, request.response)
			}
		}
	}()

}

// Request a zero-knowledge proof.
func (client *client) requestProof(commitment []byte, challenge []byte) []byte {

	response := make(chan []byte, 1)
	client.proofRequests <- proofRequest{
		commitment,
		challenge,
		response,
	}
	proof := <-response
	close(response)

	return proof

}

// Send a zero-knowledge proof.
func (client *client) sendProof(stream net.Stream, proof []byte) error {

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
func (client *client) receiveProof(stream net.Stream) ([]byte, error) {

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
