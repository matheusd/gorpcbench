// Copyright (c) 2025 Matheus Degiovani
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package mdcapnp

import (
	"context"
	"fmt"
	"net"
	"sync/atomic"

	"github.com/matheusd/gorpcbench/rpcbench"
	"github.com/rs/zerolog"
	rpc "matheusd.com/mdcapnp/capnprpc"
	ser "matheusd.com/mdcapnp/capnpser"
)

type client struct {
	rv  rpc.RemoteVat
	api testAPI
}

func (c *client) Nop(ctx context.Context) error {
	return c.api.Nop().Wait(ctx)
}

func (c *client) Add(ctx context.Context, a int64, b int64) (int64, error) {
	return c.api.Add(a, b).Wait(ctx)
}

func (c *client) MultTreeValues(ctx context.Context, mult int64, fillArgs func(rpcbench.TreeNode)) (rpcbench.TreeNode, error) {
	mtFuture, reqb := c.api.MultTree(0)
	reqb.SetMult(mult)
	root, err := reqb.NewTree()
	if err != nil {
		return nil, err
	}

	// This creates a new adapter per call, which causes allocations.
	fillArgs(&tnBuilderAdapterAlt{tnb: root})

	reply, rr, err := mtFuture.Wait(ctx)
	if err != nil {
		return nil, fmt.Errorf("err after Wait(): %v", err)
	}
	reply = ser.StructWithDepthLimit(reply, 100)

	// rr.Release() call is elided because the return values outlive this
	// function. Caller could arrange to release them.
	_ = rr.Release

	return newTnReaderAdapter(reply)
}

func (c *client) ToHex(ctx context.Context, in, out []byte) error {
	res, rr, err := c.api.ToHex(in).Wait(ctx)
	if err != nil {
		return err
	}
	copy(out, res.HexData())
	rr.Release()
	return nil
}

var clientCount atomic.Uint64

func newClient(ctx context.Context, addr string) (*client, error) {
	// Try to connect.
	var dc net.Dialer
	c, err := dc.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, err
	}

	l := zerolog.Nop()
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixNano
	// l = zerolog.New(os.Stderr).With().Timestamp().Logger()

	cnb := clientCount.Add(1)

	var testVatOpts []rpc.VatOption
	testVatOpts = append(testVatOpts,
		rpc.WithName(fmt.Sprintf("client%02d", cnb)),
		rpc.WithLogger(&l),
	)
	vat := rpc.NewVat(testVatOpts...)
	go vat.Run(ctx)

	remoteName := c.RemoteAddr().String()
	ioc := rpc.NewIOTransport(remoteName, c)

	rv := vat.UseRemoteVat(ioc)

	// Fetch the bootstrap cap (API reference).
	boot := rv.Bootstrap()
	if _, err := boot.Wait(ctx); err != nil {
		return nil, err
	}

	return &client{rv: rv, api: testAPIFromBootstrap(boot)}, nil
}
