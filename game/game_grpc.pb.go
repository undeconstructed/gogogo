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

// RGameClient is the client API for RGame service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type RGameClient interface {
	Load(ctx context.Context, in *RLoadRequest, opts ...grpc.CallOption) (*RLoadResponse, error)
	Init(ctx context.Context, in *RInitRequest, opts ...grpc.CallOption) (*RInitResponse, error)
	AddPlayer(ctx context.Context, in *RAddPlayerRequest, opts ...grpc.CallOption) (*RAddPlayerResponse, error)
	Start(ctx context.Context, in *RStartRequest, opts ...grpc.CallOption) (*RStartResponse, error)
	Play(ctx context.Context, in *RPlayRequest, opts ...grpc.CallOption) (*RPlayResponse, error)
	Destroy(ctx context.Context, in *RDestroyRequest, opts ...grpc.CallOption) (*RDestroyResponse, error)
}

type rGameClient struct {
	cc grpc.ClientConnInterface
}

func NewRGameClient(cc grpc.ClientConnInterface) RGameClient {
	return &rGameClient{cc}
}

func (c *rGameClient) Load(ctx context.Context, in *RLoadRequest, opts ...grpc.CallOption) (*RLoadResponse, error) {
	out := new(RLoadResponse)
	err := c.cc.Invoke(ctx, "/game.RGame/Load", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *rGameClient) Init(ctx context.Context, in *RInitRequest, opts ...grpc.CallOption) (*RInitResponse, error) {
	out := new(RInitResponse)
	err := c.cc.Invoke(ctx, "/game.RGame/Init", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *rGameClient) AddPlayer(ctx context.Context, in *RAddPlayerRequest, opts ...grpc.CallOption) (*RAddPlayerResponse, error) {
	out := new(RAddPlayerResponse)
	err := c.cc.Invoke(ctx, "/game.RGame/AddPlayer", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *rGameClient) Start(ctx context.Context, in *RStartRequest, opts ...grpc.CallOption) (*RStartResponse, error) {
	out := new(RStartResponse)
	err := c.cc.Invoke(ctx, "/game.RGame/Start", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *rGameClient) Play(ctx context.Context, in *RPlayRequest, opts ...grpc.CallOption) (*RPlayResponse, error) {
	out := new(RPlayResponse)
	err := c.cc.Invoke(ctx, "/game.RGame/Play", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *rGameClient) Destroy(ctx context.Context, in *RDestroyRequest, opts ...grpc.CallOption) (*RDestroyResponse, error) {
	out := new(RDestroyResponse)
	err := c.cc.Invoke(ctx, "/game.RGame/Destroy", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// RGameServer is the server API for RGame service.
// All implementations must embed UnimplementedRGameServer
// for forward compatibility
type RGameServer interface {
	Load(context.Context, *RLoadRequest) (*RLoadResponse, error)
	Init(context.Context, *RInitRequest) (*RInitResponse, error)
	AddPlayer(context.Context, *RAddPlayerRequest) (*RAddPlayerResponse, error)
	Start(context.Context, *RStartRequest) (*RStartResponse, error)
	Play(context.Context, *RPlayRequest) (*RPlayResponse, error)
	Destroy(context.Context, *RDestroyRequest) (*RDestroyResponse, error)
	mustEmbedUnimplementedRGameServer()
}

// UnimplementedRGameServer must be embedded to have forward compatible implementations.
type UnimplementedRGameServer struct {
}

func (UnimplementedRGameServer) Load(context.Context, *RLoadRequest) (*RLoadResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Load not implemented")
}
func (UnimplementedRGameServer) Init(context.Context, *RInitRequest) (*RInitResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Init not implemented")
}
func (UnimplementedRGameServer) AddPlayer(context.Context, *RAddPlayerRequest) (*RAddPlayerResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AddPlayer not implemented")
}
func (UnimplementedRGameServer) Start(context.Context, *RStartRequest) (*RStartResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Start not implemented")
}
func (UnimplementedRGameServer) Play(context.Context, *RPlayRequest) (*RPlayResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Play not implemented")
}
func (UnimplementedRGameServer) Destroy(context.Context, *RDestroyRequest) (*RDestroyResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Destroy not implemented")
}
func (UnimplementedRGameServer) mustEmbedUnimplementedRGameServer() {}

// UnsafeRGameServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to RGameServer will
// result in compilation errors.
type UnsafeRGameServer interface {
	mustEmbedUnimplementedRGameServer()
}

func RegisterRGameServer(s grpc.ServiceRegistrar, srv RGameServer) {
	s.RegisterService(&RGame_ServiceDesc, srv)
}

func _RGame_Load_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RLoadRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RGameServer).Load(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/game.RGame/Load",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RGameServer).Load(ctx, req.(*RLoadRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RGame_Init_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RInitRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RGameServer).Init(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/game.RGame/Init",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RGameServer).Init(ctx, req.(*RInitRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RGame_AddPlayer_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RAddPlayerRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RGameServer).AddPlayer(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/game.RGame/AddPlayer",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RGameServer).AddPlayer(ctx, req.(*RAddPlayerRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RGame_Start_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RStartRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RGameServer).Start(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/game.RGame/Start",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RGameServer).Start(ctx, req.(*RStartRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RGame_Play_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RPlayRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RGameServer).Play(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/game.RGame/Play",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RGameServer).Play(ctx, req.(*RPlayRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RGame_Destroy_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RDestroyRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RGameServer).Destroy(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/game.RGame/Destroy",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RGameServer).Destroy(ctx, req.(*RDestroyRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// RGame_ServiceDesc is the grpc.ServiceDesc for RGame service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var RGame_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "game.RGame",
	HandlerType: (*RGameServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Load",
			Handler:    _RGame_Load_Handler,
		},
		{
			MethodName: "Init",
			Handler:    _RGame_Init_Handler,
		},
		{
			MethodName: "AddPlayer",
			Handler:    _RGame_AddPlayer_Handler,
		},
		{
			MethodName: "Start",
			Handler:    _RGame_Start_Handler,
		},
		{
			MethodName: "Play",
			Handler:    _RGame_Play_Handler,
		},
		{
			MethodName: "Destroy",
			Handler:    _RGame_Destroy_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "game/game.proto",
}
