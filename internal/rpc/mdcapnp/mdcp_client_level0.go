// Copyright (c) 2025 Matheus Degiovani
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package mdcapnp

import (
	"context"
	"fmt"
	"net"

	"github.com/matheusd/gorpcbench/rpcbench"
	rpc "matheusd.com/mdcapnp/capnprpc"
	ser "matheusd.com/mdcapnp/capnpser"
)

type clientLevel0 struct {
	c   *rpc.Level0ClientVat
	api testAPI

	buildTree tnBuilderAdapterAlt
	readTree  tnReaderAdapterAlt
}

func (c *clientLevel0) Nop(ctx context.Context) error {
	return c.api.Nop().Wait(ctx)
}

func (c *clientLevel0) Add(ctx context.Context, a int64, b int64) (int64, error) {
	return c.api.Add(a, b).Wait(ctx)
}

func (c *clientLevel0) MultTreeValues(ctx context.Context, mult int64, fillArgs func(rpcbench.TreeNode)) (rpcbench.TreeNode, error) {
	// Use a high estimate for the max message size (max number of nodes)
	// that will be sent.
	var sizeHint ser.WordCount = 65000

	mtFuture, reqb := c.api.MultTree(sizeHint)
	reqb.SetMult(mult)
	root, err := reqb.NewTree()
	if err != nil {
		return nil, err
	}

	buildTree := &c.buildTree
	buildTree.tnb = root
	fillArgs(buildTree)

	res, rr, err := mtFuture.Wait(ctx)
	if err != nil {
		return nil, fmt.Errorf("err after Wait(): %v", err)
	}
	res = ser.StructWithDepthLimit(res, 100)

	// rr.Release() call is elided because the return values outlive this
	// function.
	//
	// Additionally, level 0 clients can only make synchronous calls,
	// therefore the next API call will naturally reuse the backing storage,
	// without the need of calling release().
	_ = rr.Release

	c.readTree.tn = res
	if err := c.readTree.reset(); err != nil {
		return nil, fmt.Errorf("resetting readTree errored: %v", err)
	}

	return &c.readTree, nil
}
func (c *clientLevel0) ToHex(ctx context.Context, in, out []byte) error {
	res, rr, err := c.api.ToHex(in).Wait(ctx)
	if err != nil {
		return err
	}
	resHex := res.HexData()
	copy(out, resHex)
	rr.Release()
	return nil
}

func newclientLevel0(ctx context.Context, addr string) (*clientLevel0, error) {
	// Try to connect.
	var dc net.Dialer
	c, err := dc.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, err
	}

	remoteName := c.RemoteAddr().String()
	ioc := rpc.NewIOTransport(remoteName, c)
	cfg := rpc.Level0ClientCfg{
		Conn: ioc,
	}
	vat := rpc.NewLevel0ClientVat(cfg)

	// Wait for bootstrap.
	boot := vat.Bootstrap()
	if _, err := boot.Wait(ctx); err != nil {
		return nil, err
	}
	api := testAPIFromBootstrap(boot)
	return &clientLevel0{c: vat, api: api}, nil
}
