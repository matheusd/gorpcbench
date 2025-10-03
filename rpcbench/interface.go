// Copyright (c) 2025 Matheus Degiovani
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpcbench

import (
	"context"
	"net"
)

// Runnable is an interface to objects that can run.
type Runnable interface {
	Run(context.Context) error
}

// Server is the interface to a server implementation.
type Server interface {
	Runnable
}

// TreeNode is a simple nested data structure.
type TreeNode struct {
	Value    int64
	Children []TreeNode
}

// TotalNodes returns the total number of nodes in this tree.
func (tn *TreeNode) TotalNodes() int {
	sum := 1
	for i := range tn.Children {
		sum += tn.Children[i].TotalNodes()
	}
	return sum
}

// Client is the interface to an RPC client with specific functions. If client
// implements Runnable, the Run() method will be called after the client is
// created and before any calls are made.
type Client interface {
	// Nop is a no-op call. It is used to measure the minimum overhead
	// imposed by the RPC subsystem to calls.
	Nop(context.Context) error

	// Add is a simple unary call. Servers should sum the two arguments and
	// return their sum.
	Add(context.Context, int64, int64) (int64, error)

	// MultTreeValues is a call with a deeply nested data structure.
	// Servers should traverse the tree, multiply the value of each node in
	// the tree by the first argument and return the new modified tree. Upon
	// reply, the client should modify the input tree to match the returned
	// values.
	MultTreeValues(context.Context, int64, *TreeNode) error

	// ToHex is a call with arbitrarily large data. Servers should convert
	// the input to an hex string. Clients should fill the output slice with
	// the results returned by the server.
	ToHex(ctx context.Context, in, out []byte) error
}

// RPCFactory is the interface to a test RPC system.
type RPCFactory interface {
	// NewServer should create a new server, bound to the given network
	// listener.
	NewServer(net.Listener) (Server, error)

	// NewClient should create a new client. A client will only ever by used
	// by a single goroutine to make calls, but multiple clients may be
	// created to access the same server during a test.
	//
	// If a client is safe for concurrent access by multiple goroutines and
	// is able to multiplex multiple concurrent calls, then implementations
	// are free to return a single client on every call.
	NewClient(context.Context, string) (Client, error)
}

type FactoryIniter func() RPCFactory
