package mdcapnp

import (
	"context"

	rpc "matheusd.com/mdcapnp/capnprpc"
	ser "matheusd.com/mdcapnp/capnpser"
)

const (
	api_interfaceId       = 0x1701d
	api_nop_methodId      = 0x0001
	api_add_methodId      = 0x0002
	api_toHex_methodId    = 0x0003
	api_multTree_methodId = 0x0004
)

type testAPI rpc.CallFuture

func (api testAPI) Nop() rpc.VoidFuture {
	return rpc.VoidFuture(rpc.RemoteCall(
		rpc.CallFuture(api),
		rpc.SetupCallNoParams(rpc.CallFuture(api),
			api_interfaceId,
			api_nop_methodId,
		),
	))
}

var addRequestSize = ser.StructSize{DataSectionSize: 2, PointerSectionSize: 0}

type addRequestBuilder ser.StructBuilder

func (b *addRequestBuilder) SetA(v int64) error {
	return (*ser.StructBuilder)(b).SetInt64(0, v)
}

func (b *addRequestBuilder) SetB(v int64) error {
	return (*ser.StructBuilder)(b).SetInt64(1, v)
}

func newAddRequestBuilder(serMsg *ser.MessageBuilder) (addRequestBuilder, error) {
	return ser.NewStructBuilder[addRequestBuilder](serMsg, addRequestSize)
}

type addRequest ser.Struct

func (s *addRequest) A() int64 {
	return (*ser.Struct)(s).Int64(0)
}

func (s *addRequest) B() int64 {
	return (*ser.Struct)(s).Int64(1)
}

var addResponseSize = ser.StructSize{DataSectionSize: 1, PointerSectionSize: 0}

type addResponseBuilder ser.StructBuilder

func (b *addResponseBuilder) SetC(v int64) error {
	return (*ser.StructBuilder)(b).SetInt64(0, v)
}

type addResponse ser.Struct

func (s *addResponse) C() int64 {
	return (*ser.Struct)(s).Int64(0)
}

type futureAddResult rpc.CallFuture

func (fut futureAddResult) Wait(ctx context.Context) (res int64, err error) {
	r, rr, err := rpc.WaitShallowCopyReturnResultsStruct[addResponse](ctx, rpc.CallFuture(fut))
	if err != nil {
		return
	}
	res = r.C()
	rr.Release()
	return
}
func (api testAPI) Add(a int64, b int64) futureAddResult {
	cs, req := rpc.SetupCallWithStructParamsGeneric[addRequestBuilder](
		rpc.CallFuture(api),
		addRequestSize.TotalSize(),
		api_interfaceId,
		api_add_methodId,
		addRequestSize,
	)

	req.SetA(a)
	req.SetB(b)
	cs.WantShallowReturnCopy = true

	return futureAddResult(rpc.RemoteCall(
		rpc.CallFuture(api),
		cs,
	))
}

var hexRequestSize = ser.StructSize{DataSectionSize: 0, PointerSectionSize: 1}

type hexRequestBuilder ser.StructBuilder

func (b *hexRequestBuilder) SetData(v []byte) error {
	return (*ser.StructBuilder)(b).SetData(0, v)
}

func newHexRequestBuilder(serMsg *ser.MessageBuilder) (hexRequestBuilder, error) {
	return ser.NewStructBuilder[hexRequestBuilder](serMsg, hexRequestSize)
}

type hexRequest ser.Struct

func (s *hexRequest) Data() []byte {
	return []byte((*ser.Struct)(s).Data(0))
}

var hexResponseSize = ser.StructSize{DataSectionSize: 0, PointerSectionSize: 1}

type hexResponseBuilder ser.StructBuilder

func (b *hexResponseBuilder) SetHexData(v []byte) error {
	return (*ser.StructBuilder)(b).SetData(0, v)
}

func (b *hexResponseBuilder) NewHexData(dataLen int) ([]byte, error) {
	return (*ser.StructBuilder)(b).NewDataField(0, ser.ByteCount(dataLen))
}

type hexResponse ser.Struct

func (s *hexResponse) HexData() []byte {
	return []byte((*ser.Struct)(s).Data(0))
}

type futureHexResult rpc.CallFuture

func (fut futureHexResult) Wait(ctx context.Context) (hexResponse, rpc.ReturnResults, error) {
	return rpc.WaitShallowCopyReturnResultsStruct[hexResponse](ctx, rpc.CallFuture(fut))
}

func (api testAPI) ToHex(v []byte) futureHexResult {
	vSerSize, _ := ser.ByteCount(len(v)).StorageWordCount()
	// vSerSize *= 4
	cs, req := rpc.SetupCallWithStructParamsGeneric[hexRequestBuilder](
		rpc.CallFuture(api),
		hexRequestSize.TotalSize()+vSerSize, // + len(v)
		api_interfaceId,
		api_toHex_methodId,
		hexRequestSize,
	)

	req.SetData(v)
	cs.WantShallowReturnCopy = true

	return futureHexResult(rpc.RemoteCall(
		rpc.CallFuture(api),
		cs,
	))
}

var (
	treeNode_size = ser.StructSize{DataSectionSize: 1, PointerSectionSize: 1}
)

type treeNodeBuilder ser.StructBuilder

func (b treeNodeBuilder) SetValue(v int64) error {
	return (*ser.StructBuilder)(&b).SetInt64(0, v)
}

func (b *treeNodeBuilder) NewChildren(listLen, listCap int) (res treeNodeListBuilder, err error) {
	err = ser.NewStructListBuilderField((*ser.StructBuilder)(b), 0, treeNode_size, listLen, listCap, (*ser.StructListBuilder)(&res))
	return
}

type treeNodeListBuilder ser.StructListBuilder

func (lb *treeNodeListBuilder) Len() int { return (*ser.StructListBuilder)(lb).Len() }
func (lb *treeNodeListBuilder) At(i int) treeNodeBuilder {
	return treeNodeBuilder((*ser.StructListBuilder)(lb).At(i))
}

func (lb *treeNodeListBuilder) ReadAt(i int, res *treeNodeBuilder) {
	(*ser.StructListBuilder)(lb).ReadAt(i, (*ser.StructBuilder)(res))
}

type treeNode ser.Struct

func (s *treeNode) Value() int64 {
	return (*ser.Struct)(s).Int64(0)
}

func (s *treeNode) Children() (res treeNodeStructList, err error) {
	err = (*ser.Struct)(s).ReadStructList(0, (*ser.StructList)(&res))
	return
}

type treeNodeStructList ser.StructList

func (sl *treeNodeStructList) Len() int          { return (*ser.StructList)(sl).Len() }
func (sl *treeNodeStructList) At(i int) treeNode { return treeNode((*ser.StructList)(sl).At(i)) }

var (
	multTreeRequest_size = ser.StructSize{DataSectionSize: 1, PointerSectionSize: 1}
)

type multTreeRequestBuilder ser.StructBuilder

func (b *multTreeRequestBuilder) SetMult(v int64) error {
	return (*ser.StructBuilder)(b).SetInt64(0, v)
}

func (b *multTreeRequestBuilder) NewTree() (treeNodeBuilder, error) {
	return ser.NewStructField[multTreeRequestBuilder, treeNodeBuilder](*b, 0, treeNode_size)
}

type multTreeRequest ser.Struct

func (s *multTreeRequest) Mult() int64 {
	return (*ser.Struct)(s).Int64(0)
}

func (s *multTreeRequest) Tree() (res treeNode, err error) {
	err = (*ser.Struct)(s).ReadStruct(0, (*ser.Struct)(&res))
	return
}

type futureMultTree rpc.CallFuture

func (fut futureMultTree) Wait(ctx context.Context) (treeNode, rpc.ReturnResults, error) {
	return rpc.WaitShallowCopyReturnResultsStruct[treeNode](ctx, rpc.CallFuture(fut))
}

func (api testAPI) MultTree(sizeHint ser.WordCount) (futureMultTree, multTreeRequestBuilder) {
	cs, reqb := rpc.SetupCallWithStructParams(
		rpc.CallFuture(api),
		multTreeRequest_size.TotalSize()+sizeHint,
		api_interfaceId,
		api_multTree_methodId,
		multTreeRequest_size,
	)

	cs.WantShallowReturnCopy = true

	return futureMultTree(rpc.RemoteCall(
		rpc.CallFuture(api),
		cs,
	)), multTreeRequestBuilder(reqb)
}

func testAPIFromBootstrap(boot rpc.BootstrapFuture) testAPI {
	return testAPI(boot)
}
