// Copyright (c) 2025 Matheus Degiovani
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package tcp

import (
	"context"
	"net"

	"github.com/matheusd/gorpcbench/rpcbench"
)

type tcpFactory struct{}

func (f tcpFactory) NewServer(l net.Listener) (rpcbench.Server, error) {
	return newTCPServer(l), nil
}

func (f tcpFactory) NewClient(ctx context.Context, addr string) (rpcbench.Client, error) {
	return newTCPClient(ctx, addr)
}

func TCPFactoryIniter() rpcbench.RPCFactory {
	return tcpFactory{}
}
