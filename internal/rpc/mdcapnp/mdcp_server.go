// Copyright (c) 2025 Matheus Degiovani
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package mdcapnp

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"net"

	"github.com/sourcegraph/conc/pool"
	rpc "matheusd.com/mdcapnp/capnprpc"
	ser "matheusd.com/mdcapnp/capnpser"
)

type server struct {
	v *rpc.Vat
	l net.Listener
	// skipLog bool
}

func (s *server) handleAdd(cc *rpc.CallContext) error {
	req, err := rpc.CallContextParamsStruct[addRequest](cc)
	if err != nil {
		return err
	}
	a, b := req.A(), req.B()
	res, err := rpc.RespondCallAsStruct[addResponseBuilder](cc, addResponseSize, 0)
	if err != nil {
		return err
	}
	res.SetC(a + b)
	return nil
}

func multTree(mul int64, in treeNode, out treeNodeBuilder) error {
	outValue := in.Value() * mul
	out.SetValue(outValue)
	inChildren, err := in.Children()
	if err != nil {
		return fmt.Errorf("errored getting inChildren: %v", err)
	}
	inChildrenLen := inChildren.Len()

	outChildren, err := out.NewChildren(inChildrenLen, inChildrenLen)
	if err != nil {
		return err
	}
	for i := range inChildrenLen {
		if err := multTree(mul, inChildren.At(i), outChildren.At(i)); err != nil {
			return err
		}
	}
	return nil
}

func (s *server) handleMultTree(cc *rpc.CallContext) error {
	req, err := rpc.CallContextParamsStruct[multTreeRequest](cc)
	if err != nil {
		return err
	}

	inTree, err := req.Tree()
	if err != nil {
		return err
	}
	inTree = ser.StructWithDepthLimit(inTree, 100)

	// The response is the same(ish) size of the request, so use that as a
	// size hint of the reponse.
	var resSizeHint ser.WordCount = (*ser.Struct)(&req).Arena().TotalSize()
	res, err := rpc.RespondCallAsStruct[treeNodeBuilder](cc, treeNode_size, resSizeHint)
	if err != nil {
		return err
	}

	if err := multTree(req.Mult(), inTree, res); err != nil {
		return err
	}

	return nil
}

func (s *server) handleToHex(cc *rpc.CallContext) error {
	req, err := rpc.CallContextParamsStruct[hexRequest](cc)
	if err != nil {
		return err
	}
	in := req.Data()
	resSizeHint, _ := ser.ByteCount(len(in) * 2).StorageWordCount()
	res, err := rpc.RespondCallAsStruct[hexResponseBuilder](cc, hexResponseSize, resSizeHint)
	if err != nil {
		return err
	}

	out, err := res.NewHexData(len(in) * 2)
	if err != nil {
		return err
	}

	hex.Encode(out, in)
	return nil
}

func (s *server) Call(ctx context.Context, cc *rpc.CallContext) error {
	if cc.InterfaceId() != api_interfaceId {
		return errors.New("wrong interfaceId")
	}
	switch cc.MethodId() {
	case api_nop_methodId:
		return nil
	case api_add_methodId:
		return s.handleAdd(cc)
	case api_multTree_methodId:
		return s.handleMultTree(cc)
	case api_toHex_methodId:
		return s.handleToHex(cc)
	default:
		return errors.New("unimplemented method")
	}
}

func (s *server) Run(ctx context.Context) error {
	g := pool.New().WithContext(ctx).WithCancelOnError().WithFirstError()

	g.Go(func(ctx context.Context) error {
		<-ctx.Done()
		return s.l.Close()
	})

	g.Go(s.v.Run)

	g.Go(func(ctx context.Context) error {
		var acceptErr error
		connPool := pool.New().WithContext(ctx).WithCancelOnError().WithFirstError()
		for {
			c, err := s.l.Accept()
			if err != nil {
				if !errors.Is(err, net.ErrClosed) {
					acceptErr = err
				}
				break
			}

			remoteName := c.RemoteAddr().String()
			ioc := rpc.NewIOTransport(remoteName, c)
			s.v.RunConn(ioc)
		}

		waitErr := connPool.Wait()
		switch {
		case acceptErr != nil:
			return fmt.Errorf("server Accept() errored: %w", acceptErr)
		case waitErr != nil:
			return fmt.Errorf("conn wait() errored: %w", waitErr)
		default:
			return nil
		}
	})

	err := g.Wait()
	if errors.Is(err, context.Canceled) {
		err = nil
	}
	return err
}

func newServer(l net.Listener) *server {
	// logger := zerolog.New(os.Stderr)

	s := &server{l: l}

	var testVatOpts []rpc.VatOption
	testVatOpts = append(testVatOpts,
		rpc.WithName("server"),
		// rpc.WithLogger(&logger),
		rpc.WithBootstrapHandler(s),
	)
	s.v = rpc.NewVat(testVatOpts...)

	return s
}
