// Copyright (c) 2025 Matheus Degiovani
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package grpc

import (
	"context"

	"github.com/matheusd/gorpcbench/rpcbench"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type grpcClient struct {
	conn *grpc.ClientConn
	api  APIClient
}

func (c *grpcClient) Nop(ctx context.Context) error {
	_, err := c.api.Nop(ctx, nil)
	return err
}

func (c *grpcClient) Add(ctx context.Context, a int64, b int64) (int64, error) {
	res, err := c.api.Add(ctx, &AddRequest{A: a, B: b})
	if err != nil {
		return 0, err
	}
	return res.Res, nil
}

func (c *grpcClient) MultTreeValues(ctx context.Context, mult int64, tree *rpcbench.TreeNode) error {
	reqTree := new(TreeNode)
	treeToGrpc(tree, reqTree)
	res, err := c.api.MultTree(ctx, &MultTreeRequest{Mult: mult, Tree: reqTree})
	if err != nil {
		return err
	}
	return grpcToTree(res.Tree, tree)
}

func (c *grpcClient) ToHex(ctx context.Context, in, out []byte) error {
	res, err := c.api.ToHex(ctx, &ToHexRequest{In: in})
	if err != nil {
		return err
	}
	copy(out, res.Out)
	return nil
}

func newGRPCClient(ctx context.Context, addr string) (*grpcClient, error) {
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}
	conn, err := grpc.NewClient(addr, opts...)
	if err != nil {
		return nil, err
	}
	api := NewAPIClient(conn)

	go func() {
		<-ctx.Done()
		conn.Close()
	}()

	return &grpcClient{
		conn: conn,
		api:  api,
	}, nil
}
