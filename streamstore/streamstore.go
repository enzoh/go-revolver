/**
 * File        : streamstore.go
 * Description : High-level stream store interface.
 * Copyright   : Copyright (c) 2017-2018 DFINITY Stiftung. All rights reserved.
 * Maintainer  : Enzo Haussecker <enzo@dfinity.org>
 * Stability   : Stable
 */

package streamstore

import (
	"errors"
	"io"
	"math"
	"sort"
	"sync"

	"github.com/enzoh/go-logging"
	"gx/ipfs/QmNa31VPzC561NWwRsJLE7nGYZYuuD2QfpK2b1q9BK54J1/go-libp2p-net"
	"gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"

	"github.com/dfinity/go-revolver/routingtable"
)

// Streamstore is a thread-safe collection of peer-stream pairs.
type Streamstore interface {

	// Add a stream to the stream store.  The `outbound` flag specifies if the
	// stream is an outbound stream.
	Add(peerID peer.ID, stream net.Stream, outbound bool) bool

	// Remove a stream from the stream store.
	Remove(peer.ID)

	// Remove all streams from the stream store.
	Purge()

	// Apply a function to a subset of streams in the stream store except
	// those specified in a sorted exclude list.
	Apply(func(peer.ID, io.Writer) error, peer.IDSlice) map[peer.ID]chan error

	// Apply a function to every stream in the stream store except
	// those specified in a sorted exclude list.
	ApplyAll(func(peer.ID, io.Writer) error, peer.IDSlice) map[peer.ID]chan error

	// Get the peers associated with inbound streams.
	InboundPeers() []peer.ID

	// Get the peers associated with the outbound streams.
	OutboundPeers() []peer.ID

	// Get the inbound capacity of the stream store.
	InboundCapacity() int

	// Get the outbound capacity of the stream store.
	OutboundCapacity() int

	// Get the current number of inbound streams.
	InboundSize() int

	// Get the current number of outbound streams.
	OutboundSize() int
}

type streamstore struct {
	inboundCapacity  int
	outboundCapacity int

	peers        map[peer.ID]peerctx
	routingTable routingtable.RoutingTable

	txQueueSize int
	*logging.Logger
	sync.RWMutex
}

type peerctx struct {
	outbound bool
	queue    chan transaction
	stream   net.Stream
}

// Release resources associated with this context.
func (p *peerctx) Close() error {
	close(p.queue)
	return p.stream.Close()
}

type transaction struct {
	query  func(peer.ID, io.Writer) error
	result map[peer.ID]chan error
	*sync.Mutex
}

// New creates a stream store.
func New(inboundCapacity, outboundCapacity, txQueueSize int, probe routingtable.LatencyProbeFn) Streamstore {
	return &streamstore{
		inboundCapacity:  inboundCapacity,
		outboundCapacity: outboundCapacity,
		peers:            make(map[peer.ID]peerctx),
		routingTable:     routingtable.NewRingsRoutingTable(routingtable.NewDefaultRingsConfig(probe)),
		txQueueSize:      txQueueSize,
		Logger:           logging.MustGetLogger("streamstore"),
		RWMutex:          sync.RWMutex{},
	}
}

func (ss *streamstore) Add(pid peer.ID, stream net.Stream, outbound bool) bool {
	ss.Lock()
	defer ss.Unlock()

	ctx, exists := ss.peers[pid]
	if exists {
		ss.Debug("Removing", pid, "from stream store")
		ctx.Close()
		delete(ss.peers, pid)
	}

	if outbound && ss.outboundSize() >= ss.OutboundCapacity() {
		ss.Debug("Cannot add", pid, "to stream store: too many outbound connections")
		return false
	}

	if !outbound && ss.inboundSize() >= ss.InboundCapacity() {
		ss.Debug("Cannot add", pid, "to stream store: too many inbound connections")
		return false
	}

	ctx = peerctx{
		outbound: outbound,
		queue:    make(chan transaction, ss.txQueueSize),
		stream:   stream,
	}

	go func() {
		for {
			select {
			case tx, ok := <-ctx.queue:
				if !ok {
					return
				}
				ss.Debug("Processing transaction for", pid)
				err := tx.query(pid, ctx.stream)
				ss.Debug("Recording result for", pid)
				tx.Lock()
				tx.result[pid] <- err
				tx.Unlock()
			}
		}
	}()
	ss.Debug("Adding stream", pid, "to stream store")
	ss.peers[pid] = ctx
	ss.routingTable.Add(pid)
	return true
}

func (ss *streamstore) Apply(f func(peer.ID, io.Writer) error, exclude peer.IDSlice) map[peer.ID]chan error {
	// Apply the function to Sqrt(N) streams where N is the total capacity of
	// the stream store.
	pids := ss.routingTable.Recommend(int(math.Sqrt(float64(ss.InboundCapacity()+ss.OutboundCapacity()))), exclude)
	return ss.apply(f, exclude, pids)
}

func (ss *streamstore) ApplyAll(f func(peer.ID, io.Writer) error, exclude peer.IDSlice) map[peer.ID]chan error {
	var pids []peer.ID
	for pid := range ss.peers {
		pids = append(pids, pid)
	}
	return ss.apply(f, exclude, pids)
}

func (ss *streamstore) apply(f func(peer.ID, io.Writer) error, exclude peer.IDSlice, peers peer.IDSlice) map[peer.ID]chan error {
	ss.Lock()
	defer ss.Unlock()
	tx := transaction{
		f,
		make(map[peer.ID]chan error),
		&sync.Mutex{},
	}
	var group sync.WaitGroup
	for _, pid := range peers {
		pid := pid
		ctx := ss.peers[pid]
		i := sort.Search(len(exclude), func(i int) bool {
			return exclude[i] >= pid
		})
		if i < len(exclude) && exclude[i] == pid {
			continue
		}
		group.Add(1)
		go func() {
			defer group.Done()
			ss.Debug("Preparing result for", pid)
			tx.Lock()
			tx.result[pid] = make(chan error, 1)
			tx.Unlock()
			ss.Debug("Queueing transaction for", pid)
			select {
			case ctx.queue <- tx:
			default:
				ss.Debug("Cannot queue transaction for", pid)
				tx.Lock()
				tx.result[pid] <- errors.New("transaction queue is full")
				tx.Unlock()
			}
		}()
	}
	group.Wait()
	return tx.result
}

func (ss *streamstore) InboundCapacity() int {
	return ss.inboundCapacity
}

func (ss *streamstore) OutboundCapacity() int {
	return ss.outboundCapacity
}

func (ss *streamstore) InboundPeers() []peer.ID {
	ss.RLock()
	defer ss.RUnlock()
	var peers []peer.ID
	for pid, ctx := range ss.peers {
		if !ctx.outbound {
			peers = append(peers, pid)
		}
	}
	return peers
}

func (ss *streamstore) OutboundPeers() []peer.ID {
	ss.RLock()
	defer ss.RUnlock()
	var peers []peer.ID
	for pid, ctx := range ss.peers {
		if ctx.outbound {
			peers = append(peers, pid)
		}
	}
	return peers
}

func (ss *streamstore) Purge() {
	ss.Lock()
	defer ss.Unlock()

	for pid, ctx := range ss.peers {
		ss.Debug("Removing stream", pid, "from stream store")
		ctx.Close()
	}

	ss.peers = make(map[peer.ID]peerctx)
}

func (ss *streamstore) Remove(pid peer.ID) {
	ss.Lock()
	defer ss.Unlock()

	if ctx, exists := ss.peers[pid]; exists {
		ss.Debug("Removing stream", pid, "from stream store")
		ctx.Close()
		delete(ss.peers, pid)
	}

	ss.routingTable.Remove(pid)
}

func (ss *streamstore) InboundSize() int {
	ss.RLock()
	defer ss.RUnlock()

	return ss.inboundSize()
}

func (ss *streamstore) inboundSize() int {
	var count int
	for _, ctx := range ss.peers {
		if !ctx.outbound {
			count++
		}
	}
	return count
}

func (ss *streamstore) OutboundSize() int {
	ss.RLock()
	defer ss.RUnlock()

	return ss.outboundSize()
}

func (ss *streamstore) outboundSize() int {
	var count int
	for _, ctx := range ss.peers {
		if ctx.outbound {
			count++
		}
	}
	return count
}
