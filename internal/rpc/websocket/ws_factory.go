// Copyright (c) 2025 Matheus Degiovani
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package websocket

import (
	"context"
	"net"

	"github.com/matheusd/gorpcbench/rpcbench"
)

type wsFactory struct{}

func (f wsFactory) NewServer(l net.Listener) (rpcbench.Server, error) {
	return newWSServer(l), nil
}

func (f wsFactory) NewClient(ctx context.Context, addr string) (rpcbench.Client, error) {
	return newWSClient(ctx, addr)
}

func WSFactoryIniter() rpcbench.RPCFactory {
	return wsFactory{}
}
