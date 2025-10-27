// Copyright (c) 2025 Matheus Degiovani
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package mdcapnp

import (
	"context"
	"net"

	"github.com/matheusd/gorpcbench/rpcbench"
)

type mdcapFactory struct {
	level0 bool
}

func (f mdcapFactory) NewServer(l net.Listener) (rpcbench.Server, error) {
	return newServer(l), nil
}

func (f mdcapFactory) NewClient(ctx context.Context, addr string) (rpcbench.Client, error) {
	if f.level0 {
		return newclientLevel0(ctx, addr)
	}
	return newClient(ctx, addr)
}

func MDCapNProtoFactoryIniter() rpcbench.RPCFactory {
	return mdcapFactory{}
}

func MDCapNProtoLevel0FactoryIniter() rpcbench.RPCFactory {
	return mdcapFactory{level0: true}
}
