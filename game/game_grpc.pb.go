// Code generated by protoc-gen-go-grpc. DO NOT EDIT.

package game

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// InstanceClient is the client API for Instance service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type InstanceClient interface {
	// Load means find game data and load it.
	Load(ctx context.Context, in *RLoadRequest, opts ...grpc.CallOption) (*RLoadResponse, error)
	// Init means create a new game here.
	Init(ctx context.Context, in *RInitRequest, opts ...grpc.CallOption) (*RInitResponse, error)
	// AddPlayer adds a player. It may be not allowed after start.
	AddPlayer(ctx context.Context, in *RAddPlayerRequest, opts ...grpc.CallOption) (*RAddPlayerResponse, error)
	// Start starts the game.
	Start(ctx context.Context, in *RStartRequest, opts ...grpc.CallOption) (*RStartResponse, error)
	// Play submits something that should be done in the context of a current turn.
	Play(ctx context.Context, in *RPlayRequest, opts ...grpc.CallOption) (*RPlayResponse, error)
	// Destroy terminates the game and removes all data.
	Destroy(ctx context.Context, in *RDestroyRequest, opts ...grpc.CallOption) (*RDestroyResponse, error)
}

type instanceClient struct {
	cc grpc.ClientConnInterface
}

func NewInstanceClient(cc grpc.ClientConnInterface) InstanceClient {
	return &instanceClient{cc}
}

func (c *instanceClient) Load(ctx context.Context, in *RLoadRequest, opts ...grpc.CallOption) (*RLoadResponse, error) {
	out := new(RLoadResponse)
	err := c.cc.Invoke(ctx, "/game.Instance/Load", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *instanceClient) Init(ctx context.Context, in *RInitRequest, opts ...grpc.CallOption) (*RInitResponse, error) {
	out := new(RInitResponse)
	err := c.cc.Invoke(ctx, "/game.Instance/Init", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *instanceClient) AddPlayer(ctx context.Context, in *RAddPlayerRequest, opts ...grpc.CallOption) (*RAddPlayerResponse, error) {
	out := new(RAddPlayerResponse)
	err := c.cc.Invoke(ctx, "/game.Instance/AddPlayer", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *instanceClient) Start(ctx context.Context, in *RStartRequest, opts ...grpc.CallOption) (*RStartResponse, error) {
	out := new(RStartResponse)
	err := c.cc.Invoke(ctx, "/game.Instance/Start", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *instanceClient) Play(ctx context.Context, in *RPlayRequest, opts ...grpc.CallOption) (*RPlayResponse, error) {
	out := new(RPlayResponse)
	err := c.cc.Invoke(ctx, "/game.Instance/Play", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *instanceClient) Destroy(ctx context.Context, in *RDestroyRequest, opts ...grpc.CallOption) (*RDestroyResponse, error) {
	out := new(RDestroyResponse)
	err := c.cc.Invoke(ctx, "/game.Instance/Destroy", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// InstanceServer is the server API for Instance service.
// All implementations must embed UnimplementedInstanceServer
// for forward compatibility
type InstanceServer interface {
	// Load means find game data and load it.
	Load(context.Context, *RLoadRequest) (*RLoadResponse, error)
	// Init means create a new game here.
	Init(context.Context, *RInitRequest) (*RInitResponse, error)
	// AddPlayer adds a player. It may be not allowed after start.
	AddPlayer(context.Context, *RAddPlayerRequest) (*RAddPlayerResponse, error)
	// Start starts the game.
	Start(context.Context, *RStartRequest) (*RStartResponse, error)
	// Play submits something that should be done in the context of a current turn.
	Play(context.Context, *RPlayRequest) (*RPlayResponse, error)
	// Destroy terminates the game and removes all data.
	Destroy(context.Context, *RDestroyRequest) (*RDestroyResponse, error)
	mustEmbedUnimplementedInstanceServer()
}

// UnimplementedInstanceServer must be embedded to have forward compatible implementations.
type UnimplementedInstanceServer struct {
}

func (UnimplementedInstanceServer) Load(context.Context, *RLoadRequest) (*RLoadResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Load not implemented")
}
func (UnimplementedInstanceServer) Init(context.Context, *RInitRequest) (*RInitResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Init not implemented")
}
func (UnimplementedInstanceServer) AddPlayer(context.Context, *RAddPlayerRequest) (*RAddPlayerResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AddPlayer not implemented")
}
func (UnimplementedInstanceServer) Start(context.Context, *RStartRequest) (*RStartResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Start not implemented")
}
func (UnimplementedInstanceServer) Play(context.Context, *RPlayRequest) (*RPlayResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Play not implemented")
}
func (UnimplementedInstanceServer) Destroy(context.Context, *RDestroyRequest) (*RDestroyResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Destroy not implemented")
}
func (UnimplementedInstanceServer) mustEmbedUnimplementedInstanceServer() {}

// UnsafeInstanceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to InstanceServer will
// result in compilation errors.
type UnsafeInstanceServer interface {
	mustEmbedUnimplementedInstanceServer()
}

func RegisterInstanceServer(s grpc.ServiceRegistrar, srv InstanceServer) {
	s.RegisterService(&Instance_ServiceDesc, srv)
}

func _Instance_Load_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RLoadRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(InstanceServer).Load(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/game.Instance/Load",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(InstanceServer).Load(ctx, req.(*RLoadRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Instance_Init_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RInitRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(InstanceServer).Init(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/game.Instance/Init",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(InstanceServer).Init(ctx, req.(*RInitRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Instance_AddPlayer_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RAddPlayerRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(InstanceServer).AddPlayer(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/game.Instance/AddPlayer",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(InstanceServer).AddPlayer(ctx, req.(*RAddPlayerRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Instance_Start_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RStartRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(InstanceServer).Start(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/game.Instance/Start",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(InstanceServer).Start(ctx, req.(*RStartRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Instance_Play_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RPlayRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(InstanceServer).Play(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/game.Instance/Play",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(InstanceServer).Play(ctx, req.(*RPlayRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Instance_Destroy_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RDestroyRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(InstanceServer).Destroy(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/game.Instance/Destroy",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(InstanceServer).Destroy(ctx, req.(*RDestroyRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// Instance_ServiceDesc is the grpc.ServiceDesc for Instance service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Instance_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "game.Instance",
	HandlerType: (*InstanceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Load",
			Handler:    _Instance_Load_Handler,
		},
		{
			MethodName: "Init",
			Handler:    _Instance_Init_Handler,
		},
		{
			MethodName: "AddPlayer",
			Handler:    _Instance_AddPlayer_Handler,
		},
		{
			MethodName: "Start",
			Handler:    _Instance_Start_Handler,
		},
		{
			MethodName: "Play",
			Handler:    _Instance_Play_Handler,
		},
		{
			MethodName: "Destroy",
			Handler:    _Instance_Destroy_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "game/game.proto",
}
