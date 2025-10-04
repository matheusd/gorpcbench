// Copyright (c) 2025 Matheus Degiovani
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package websocket

import (
	"context"
	"net"

	"github.com/matheusd/gorpcbench/rpcbench"
)

type wsFactory struct {
	isJson bool
}

func (f wsFactory) NewServer(l net.Listener) (rpcbench.Server, error) {
	return newWSServer(l), nil
}

func (f wsFactory) NewClient(ctx context.Context, addr string) (rpcbench.Client, error) {
	return newWSClient(ctx, addr, f.isJson)
}

func WSFactoryIniter() rpcbench.RPCFactory {
	return wsFactory{}
}

func WSJsonFactoryIniter() rpcbench.RPCFactory {
	return wsFactory{isJson: true}
}
