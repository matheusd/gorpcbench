// Copyright (c) 2025 Matheus Degiovani
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package http1

import (
	"context"
	"net"

	"github.com/matheusd/gorpcbench/rpcbench"
)

type http1Factory struct{}

func (f http1Factory) NewServer(l net.Listener) (rpcbench.Server, error) {
	return newHttp1Server(l), nil
}

func (f http1Factory) NewClient(ctx context.Context, addr string) (rpcbench.Client, error) {
	return newHttp1Client(ctx, addr)
}

func HTTP1FactoryIniter() rpcbench.RPCFactory {
	return http1Factory{}
}
