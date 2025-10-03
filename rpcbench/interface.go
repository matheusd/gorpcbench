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

// TreeNode is an interface to a TreeNode structure. This is an interface (as
// opposed to a direct struct) to allow RPC systems to use their preferred
// representation for data.
//
// TreeNode values are reused across tests, but not across different clients.
type TreeNode interface {
	SetValue(v int64)
	GetValue() int64

	// InitChildren should init n new nodes as children of this node.
	InitChildren(n int)
	ChildrenCount() int

	// Child should return a reference to child i. It is ok to panic if i is
	// out of bounds.
	Child(i int) TreeNode

	// TotalNodes should return the total number of nodes across the
	// subtree.
	TotalNodes() int
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
	// the tree by the first argument and return the new modified tree.
	//
	// Clients need to call the fillArgs function in order to fill a
	// TreeNode implementation with the values for this specific call.
	//
	// This roundabound way of setting args is necessary to ensure every RPC
	// implementation has a chance to cache the arguments (if possible) and
	// to be fair(ish) and ensure every implementation sets fields in at
	// least one structure prior to writing the message to the server.
	//
	// The assumption held here is that in production code, the argument
	// (TreeNode) will already be filled by the app with data from a
	// database, some computation, a prior RPC call, and so on. But
	// different RPC implementations may be using different structures to
	// hold this value, therefore, for benchmarking purposes, we ensure
	// every implementation requires setting one structure.
	//
	// The resulting TreeNode returned by this function MAY be the same one
	// passed to the fillArgs call (if the implementation is able to reuse
	// it).
	MultTreeValues(ctx context.Context, mult int64, fillArgs func(TreeNode)) (TreeNode, error)

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
