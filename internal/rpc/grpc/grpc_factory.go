// Copyright (c) 2025 Matheus Degiovani
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package grpc

//go:generate protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative structdef.proto

import (
	"context"
	"net"

	"github.com/matheusd/gorpcbench/rpcbench"
)

type grpcFactory struct{}

func (f grpcFactory) NewServer(l net.Listener) (rpcbench.Server, error) {
	return newGRPCServer(l), nil
}

func (f grpcFactory) NewClient(ctx context.Context, addr string) (rpcbench.Client, error) {
	return newGRPCClient(ctx, addr)
}

func GRPCFactoryIniter() rpcbench.RPCFactory {
	return grpcFactory{}
}
