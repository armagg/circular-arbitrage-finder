





package exec

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)




const _ = grpc.SupportPackageIsVersion8

const (
	Executor_ProposePlan_FullMethodName = "/exec.Executor/ProposePlan"
)




type ExecutorClient interface {
	ProposePlan(ctx context.Context, in *Plan, opts ...grpc.CallOption) (*ProposeReply, error)
}

type executorClient struct {
	cc grpc.ClientConnInterface
}

func NewExecutorClient(cc grpc.ClientConnInterface) ExecutorClient {
	return &executorClient{cc}
}

func (c *executorClient) ProposePlan(ctx context.Context, in *Plan, opts ...grpc.CallOption) (*ProposeReply, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(ProposeReply)
	err := c.cc.Invoke(ctx, Executor_ProposePlan_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}




type ExecutorServer interface {
	ProposePlan(context.Context, *Plan) (*ProposeReply, error)
	mustEmbedUnimplementedExecutorServer()
}


type UnimplementedExecutorServer struct {
}

func (UnimplementedExecutorServer) ProposePlan(context.Context, *Plan) (*ProposeReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ProposePlan not implemented")
}
func (UnimplementedExecutorServer) mustEmbedUnimplementedExecutorServer() {}




type UnsafeExecutorServer interface {
	mustEmbedUnimplementedExecutorServer()
}

func RegisterExecutorServer(s grpc.ServiceRegistrar, srv ExecutorServer) {
	s.RegisterService(&Executor_ServiceDesc, srv)
}

func _Executor_ProposePlan_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Plan)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ExecutorServer).ProposePlan(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Executor_ProposePlan_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ExecutorServer).ProposePlan(ctx, req.(*Plan))
	}
	return interceptor(ctx, in, info, handler)
}




var Executor_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "exec.Executor",
	HandlerType: (*ExecutorServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "ProposePlan",
			Handler:    _Executor_ProposePlan_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "proto/executor.proto",
}
