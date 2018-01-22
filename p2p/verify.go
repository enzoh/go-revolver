/**
 * File        : verify.go
 * Description : Authentication submodule.
 * Copyright   : Copyright (c) 2017-2018 DFINITY Stiftung. All rights reserved.
 * Maintainer  : Enzo Haussecker <enzo@dfinity.org>
 * Stability   : Experimental
 */

package p2p

// This type represents a function that executes when receiving a verification
// request.
type VerificationHandler func(commitment []byte, challenge []byte, proof []byte, response chan bool)

// This type provides the data needed to request a verification.
type verificationRequest struct {
	commitment []byte
	challenge  []byte
	proof      []byte
	response   chan bool
}

// Register a verification request handler.
func (client *client) setVerificationHandler(handler VerificationHandler) {

	notify := make(chan struct{})
	client.unsetVerificationHandler = func() {
		close(notify)
	}

	go func() {
		for {
			select {
			case <-notify:
				return
			case request := <-client.verificationRequests:
				handler(request.commitment, request.challenge, request.proof, request.response)
			}
		}
	}()

}

// Handle a verification request.
func DefaultVerificationHandler(commitment, challenge, proof []byte, response chan bool) {
	response <- true
}

// Request a verification.
func (client *client) requestVerification(commitment []byte, challenge []byte, proof []byte) bool {

	response := make(chan bool, 1)
	client.verificationRequests <- verificationRequest{
		commitment,
		challenge,
		proof,
		response,
	}
	success := <-response
	close(response)

	return success

}
