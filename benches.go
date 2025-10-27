// Copyright (c) 2025 Matheus Degiovani
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"github.com/matheusd/gorpcbench/internal/rpc/gocapnp"
	"github.com/matheusd/gorpcbench/internal/rpc/grpc"
	"github.com/matheusd/gorpcbench/internal/rpc/http1"
	"github.com/matheusd/gorpcbench/internal/rpc/mdcapnp"
	"github.com/matheusd/gorpcbench/internal/rpc/tcp"
	"github.com/matheusd/gorpcbench/internal/rpc/websocket"
	"github.com/matheusd/gorpcbench/rpcbench"
)

var allSystems = []rpcbench.RPCSystem{
	{
		Name:   "tcp",
		Initer: tcp.TCPFactoryIniter,
		Notes:  "Raw TCP-based RPC implementation",
	}, {
		Name:   "http1",
		Initer: http1.HTTP1FactoryIniter,
		Notes:  "HTTP-based RPC implementation",
	}, {
		Name:   "ws",
		Initer: websocket.WSFactoryIniter,
		Notes:  "Websockets-based RPC implementation",
	}, {
		Name:   "wsjson",
		Initer: websocket.WSJsonFactoryIniter,
		Notes:  "Websockets-based RPC implementation (JSON)",
	}, {
		Name:   "grpc",
		Initer: grpc.GRPCFactoryIniter,
		Notes:  "gRPC based implementation",
	}, {
		Name:   "gocapnp",
		Initer: gocapnp.GoCapnpIniter,
		Notes:  "go-CapNProto based implementation",
	}, {
		Name:   "mdcapnp",
		Initer: mdcapnp.MDCapNProtoFactoryIniter,
		Notes:  "MdCapNProto based implementation",
	}, {
		Name:   "mdcapl0",
		Initer: mdcapnp.MDCapNProtoLevel0FactoryIniter,
		Notes:  "Level 0 MdCapNProto based implementation",
	},
}

func fullTestMatrix() []rpcbench.BenchCase {
	calls := rpcbench.ClientCallMatrix()
	parallelCases := []bool{false, true}
	matrix := make([]rpcbench.BenchCase, 0, len(parallelCases)*len(calls)*len(allSystems))
	for _, parallel := range parallelCases {
		for _, call := range calls {
			for si := range allSystems {
				matrix = append(matrix, rpcbench.BenchCase{
					Sys:      &allSystems[si],
					Call:     call,
					Parallel: parallel,
				})
			}
		}
	}
	return matrix
}
