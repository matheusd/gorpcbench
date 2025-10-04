// Copyright (c) 2025 Matheus Degiovani
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package gocapnp

import (
	context "context"
	"net"

	"capnproto.org/go/capnp/v3/rpc"
	"github.com/matheusd/gorpcbench/rpcbench"
)

type gocapnpClient struct {
	api API
}

func (c *gocapnpClient) Nop(ctx context.Context) error {
	nopFuture, release := c.api.Nop(ctx, nil)
	defer release()
	select {
	case <-nopFuture.Done():
	case <-ctx.Done():
		return ctx.Err()
	}
	return nil
}

func (c *gocapnpClient) Add(ctx context.Context, a int64, b int64) (int64, error) {
	addFuture, release := c.api.Add(ctx, func(args API_add_Params) error {
		args.SetA(a)
		args.SetB(b)
		return nil
	})
	defer release()

	res, err := addFuture.Struct()
	if err != nil {
		return 0, err
	}

	return res.Res(), nil
}

func (c *gocapnpClient) MultTreeValues(ctx context.Context, mult int64, fillArgs func(rpcbench.TreeNode)) (rpcbench.TreeNode, error) {
	treeFuture, release := c.api.MultTree(ctx, func(args API_multTree_Params) error {
		args.SetMult(mult)
		tree, err := args.NewTree()
		if err != nil {
			return err
		}

		fillArgs(tree)
		return nil
	})

	// Release cannot be called on defer() here because the resulting
	// TreeNode is used by the caller. In production code, this means the
	// caller would have to arrange to release the results.
	_ = release
	// defer release()

	res, err := treeFuture.Struct()
	if err != nil {
		return nil, err
	}

	resTree, err := res.Res()
	if err != nil {
		return nil, err
	}

	return resTree, nil
}

func (c *gocapnpClient) ToHex(ctx context.Context, in []byte, out []byte) error {
	hexFuture, release := c.api.ToHex(ctx, func(args API_toHex_Params) error {
		args.SetIn(in)
		return nil
	})
	defer release()

	res, err := hexFuture.Struct()
	if err != nil {
		return err
	}

	outRes, err := res.Out()
	if err != nil {
		return err
	}

	copy(out, outRes)

	return nil
}

func newGoCapnpClient(ctx context.Context, addr string) (*gocapnpClient, error) {
	// Try to connect.
	var dc net.Dialer
	c, err := dc.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, err
	}

	// Convert net.Conn into a capnp Conn.
	capConn := rpc.NewConn(rpc.NewStreamTransport(c), nil) // nil sets defau

	// Get the "bootstrap" interface. This is an instance of an API server.
	// Wait until the server returns the cap (this is the "handshake" with
	// server). Not strictly necessary, but easier to reduce latency for the
	// next calls.
	api := API(capConn.Bootstrap(ctx))
	if err := api.Resolve(ctx); err != nil {
		return nil, err
	}

	go func() {
		<-ctx.Done()
		api.Release()
		capConn.Close()
		c.Close()
	}()

	return &gocapnpClient{
		api: api,
	}, nil
}
