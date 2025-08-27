





package md

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)




const _ = grpc.SupportPackageIsVersion8

const (
	OrderBookFeed_StreamBooks_FullMethodName = "/md.OrderBookFeed/StreamBooks"
)




type OrderBookFeedClient interface {
	StreamBooks(ctx context.Context, opts ...grpc.CallOption) (OrderBookFeed_StreamBooksClient, error)
}

type orderBookFeedClient struct {
	cc grpc.ClientConnInterface
}

func NewOrderBookFeedClient(cc grpc.ClientConnInterface) OrderBookFeedClient {
	return &orderBookFeedClient{cc}
}

func (c *orderBookFeedClient) StreamBooks(ctx context.Context, opts ...grpc.CallOption) (OrderBookFeed_StreamBooksClient, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	stream, err := c.cc.NewStream(ctx, &OrderBookFeed_ServiceDesc.Streams[0], OrderBookFeed_StreamBooks_FullMethodName, cOpts...)
	if err != nil {
		return nil, err
	}
	x := &orderBookFeedStreamBooksClient{ClientStream: stream}
	return x, nil
}

type OrderBookFeed_StreamBooksClient interface {
	Send(*StreamRequest) error
	Recv() (*OrderBookDelta, error)
	grpc.ClientStream
}

type orderBookFeedStreamBooksClient struct {
	grpc.ClientStream
}

func (x *orderBookFeedStreamBooksClient) Send(m *StreamRequest) error {
	return x.ClientStream.SendMsg(m)
}

func (x *orderBookFeedStreamBooksClient) Recv() (*OrderBookDelta, error) {
	m := new(OrderBookDelta)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}




type OrderBookFeedServer interface {
	StreamBooks(OrderBookFeed_StreamBooksServer) error
	mustEmbedUnimplementedOrderBookFeedServer()
}


type UnimplementedOrderBookFeedServer struct {
}

func (UnimplementedOrderBookFeedServer) StreamBooks(OrderBookFeed_StreamBooksServer) error {
	return status.Errorf(codes.Unimplemented, "method StreamBooks not implemented")
}
func (UnimplementedOrderBookFeedServer) mustEmbedUnimplementedOrderBookFeedServer() {}




type UnsafeOrderBookFeedServer interface {
	mustEmbedUnimplementedOrderBookFeedServer()
}

func RegisterOrderBookFeedServer(s grpc.ServiceRegistrar, srv OrderBookFeedServer) {
	s.RegisterService(&OrderBookFeed_ServiceDesc, srv)
}

func _OrderBookFeed_StreamBooks_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(OrderBookFeedServer).StreamBooks(&orderBookFeedStreamBooksServer{ServerStream: stream})
}

type OrderBookFeed_StreamBooksServer interface {
	Send(*OrderBookDelta) error
	Recv() (*StreamRequest, error)
	grpc.ServerStream
}

type orderBookFeedStreamBooksServer struct {
	grpc.ServerStream
}

func (x *orderBookFeedStreamBooksServer) Send(m *OrderBookDelta) error {
	return x.ServerStream.SendMsg(m)
}

func (x *orderBookFeedStreamBooksServer) Recv() (*StreamRequest, error) {
	m := new(StreamRequest)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}




var OrderBookFeed_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "md.OrderBookFeed",
	HandlerType: (*OrderBookFeedServer)(nil),
	Methods:     []grpc.MethodDesc{},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "StreamBooks",
			Handler:       _OrderBookFeed_StreamBooks_Handler,
			ServerStreams: true,
			ClientStreams: true,
		},
	},
	Metadata: "proto/marketdata.proto",
}

const (
	OrderBookIngress_PushDeltas_FullMethodName = "/md.OrderBookIngress/PushDeltas"
)




type OrderBookIngressClient interface {
	PushDeltas(ctx context.Context, opts ...grpc.CallOption) (OrderBookIngress_PushDeltasClient, error)
}

type orderBookIngressClient struct {
	cc grpc.ClientConnInterface
}

func NewOrderBookIngressClient(cc grpc.ClientConnInterface) OrderBookIngressClient {
	return &orderBookIngressClient{cc}
}

func (c *orderBookIngressClient) PushDeltas(ctx context.Context, opts ...grpc.CallOption) (OrderBookIngress_PushDeltasClient, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	stream, err := c.cc.NewStream(ctx, &OrderBookIngress_ServiceDesc.Streams[0], OrderBookIngress_PushDeltas_FullMethodName, cOpts...)
	if err != nil {
		return nil, err
	}
	x := &orderBookIngressPushDeltasClient{ClientStream: stream}
	return x, nil
}

type OrderBookIngress_PushDeltasClient interface {
	Send(*OrderBookDelta) error
	CloseAndRecv() (*Ack, error)
	grpc.ClientStream
}

type orderBookIngressPushDeltasClient struct {
	grpc.ClientStream
}

func (x *orderBookIngressPushDeltasClient) Send(m *OrderBookDelta) error {
	return x.ClientStream.SendMsg(m)
}

func (x *orderBookIngressPushDeltasClient) CloseAndRecv() (*Ack, error) {
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	m := new(Ack)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}




type OrderBookIngressServer interface {
	PushDeltas(OrderBookIngress_PushDeltasServer) error
	mustEmbedUnimplementedOrderBookIngressServer()
}


type UnimplementedOrderBookIngressServer struct {
}

func (UnimplementedOrderBookIngressServer) PushDeltas(OrderBookIngress_PushDeltasServer) error {
	return status.Errorf(codes.Unimplemented, "method PushDeltas not implemented")
}
func (UnimplementedOrderBookIngressServer) mustEmbedUnimplementedOrderBookIngressServer() {}




type UnsafeOrderBookIngressServer interface {
	mustEmbedUnimplementedOrderBookIngressServer()
}

func RegisterOrderBookIngressServer(s grpc.ServiceRegistrar, srv OrderBookIngressServer) {
	s.RegisterService(&OrderBookIngress_ServiceDesc, srv)
}

func _OrderBookIngress_PushDeltas_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(OrderBookIngressServer).PushDeltas(&orderBookIngressPushDeltasServer{ServerStream: stream})
}

type OrderBookIngress_PushDeltasServer interface {
	SendAndClose(*Ack) error
	Recv() (*OrderBookDelta, error)
	grpc.ServerStream
}

type orderBookIngressPushDeltasServer struct {
	grpc.ServerStream
}

func (x *orderBookIngressPushDeltasServer) SendAndClose(m *Ack) error {
	return x.ServerStream.SendMsg(m)
}

func (x *orderBookIngressPushDeltasServer) Recv() (*OrderBookDelta, error) {
	m := new(OrderBookDelta)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}




var OrderBookIngress_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "md.OrderBookIngress",
	HandlerType: (*OrderBookIngressServer)(nil),
	Methods:     []grpc.MethodDesc{},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "PushDeltas",
			Handler:       _OrderBookIngress_PushDeltas_Handler,
			ClientStreams: true,
		},
	},
	Metadata: "proto/marketdata.proto",
}
