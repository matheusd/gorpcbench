// Copyright (c) 2025 Matheus Degiovani
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package grpc

import (
	"context"
	"encoding/hex"
	"net"

	grpc "google.golang.org/grpc"
)

type grpcServer struct {
	UnimplementedAPIServer
	gs *grpc.Server
	l  net.Listener
}

func (s *grpcServer) Nop(context.Context, *VoidData) (*VoidData, error) {
	return nil, nil
}

func (s *grpcServer) Add(_ context.Context, req *AddRequest) (*AddResult, error) {
	return &AddResult{
		Res: req.A + req.B,
	}, nil
}

func multTree(mul int64, t *TreeNode) {
	t.Value *= mul
	for _, c := range t.Children {
		multTree(mul, c)
	}
}

func (s *grpcServer) MultTree(_ context.Context, req *MultTreeRequest) (*MultTreeResponse, error) {
	resTree := req.Tree
	multTree(req.Mult, resTree)
	return &MultTreeResponse{
		Tree: resTree,
	}, nil
}

func (s *grpcServer) ToHex(_ context.Context, req *ToHexRequest) (*ToHexResponse, error) {
	buf := make([]byte, len(req.In)*2)
	hex.Encode(buf, req.In)
	return &ToHexResponse{Out: buf}, nil
}

func (s *grpcServer) Run(ctx context.Context) error {
	go func() {
		<-ctx.Done()
		s.gs.GracefulStop()
	}()
	return s.gs.Serve(s.l)
}

func newGRPCServer(l net.Listener) *grpcServer {
	s := &grpcServer{
		l: l,
	}
	s.gs = grpc.NewServer()
	RegisterAPIServer(s.gs, s)
	return s
}
