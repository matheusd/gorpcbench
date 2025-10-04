// Copyright (c) 2025 Matheus Degiovani
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package gocapnp

import (
	context "context"
	"net"

	"github.com/matheusd/gorpcbench/rpcbench"
)

// NOTE: To run the following generate directive requires go-capnp to be checked
// out in a package in this location (../go-capnp from the root of gorpcbench
// root).

//go:generate capnp compile -I../../../../go-capnp/std -ogo structdef.capnp

type gocapnpFactory struct{}

func (g gocapnpFactory) NewServer(l net.Listener) (rpcbench.Server, error) {
	return newGoCapnpServer(l)
}

func (g gocapnpFactory) NewClient(ctx context.Context, addr string) (rpcbench.Client, error) {
	return newGoCapnpClient(ctx, addr)
}

func GoCapnpIniter() rpcbench.RPCFactory {
	return gocapnpFactory{}
}
