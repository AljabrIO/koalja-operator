// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: agent_api.proto

package pipeline // import "github.com/AljabrIO/koalja-operator/pkg/agent/pipeline"

import proto "github.com/gogo/protobuf/proto"
import fmt "fmt"
import math "math"
import v1alpha1 "github.com/AljabrIO/koalja-operator/pkg/apis/koalja/v1alpha1"
import event "github.com/AljabrIO/koalja-operator/pkg/event"
import types "github.com/gogo/protobuf/types"
import empty "github.com/golang/protobuf/ptypes/empty"
import _ "google.golang.org/genproto/googleapis/api/annotations"

import (
	context "golang.org/x/net/context"
	grpc "google.golang.org/grpc"
)

import io "io"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion2 // please upgrade the proto package

type RegisterLinkRequest struct {
	// Name of the link
	LinkName string `protobuf:"bytes,1,opt,name=LinkName,proto3" json:"LinkName,omitempty"`
	// URI of the link
	URI string `protobuf:"bytes,2,opt,name=URI,proto3" json:"URI,omitempty"`
}

func (m *RegisterLinkRequest) Reset()         { *m = RegisterLinkRequest{} }
func (m *RegisterLinkRequest) String() string { return proto.CompactTextString(m) }
func (*RegisterLinkRequest) ProtoMessage()    {}
func (*RegisterLinkRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_agent_api_ab421025fe7e89fb, []int{0}
}
func (m *RegisterLinkRequest) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *RegisterLinkRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_RegisterLinkRequest.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalTo(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (dst *RegisterLinkRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_RegisterLinkRequest.Merge(dst, src)
}
func (m *RegisterLinkRequest) XXX_Size() int {
	return m.Size()
}
func (m *RegisterLinkRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_RegisterLinkRequest.DiscardUnknown(m)
}

var xxx_messageInfo_RegisterLinkRequest proto.InternalMessageInfo

func (m *RegisterLinkRequest) GetLinkName() string {
	if m != nil {
		return m.LinkName
	}
	return ""
}

func (m *RegisterLinkRequest) GetURI() string {
	if m != nil {
		return m.URI
	}
	return ""
}

type RegisterTaskRequest struct {
	// Name of the task
	TaskName string `protobuf:"bytes,1,opt,name=TaskName,proto3" json:"TaskName,omitempty"`
	// URI of the task
	URI string `protobuf:"bytes,2,opt,name=URI,proto3" json:"URI,omitempty"`
}

func (m *RegisterTaskRequest) Reset()         { *m = RegisterTaskRequest{} }
func (m *RegisterTaskRequest) String() string { return proto.CompactTextString(m) }
func (*RegisterTaskRequest) ProtoMessage()    {}
func (*RegisterTaskRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_agent_api_ab421025fe7e89fb, []int{1}
}
func (m *RegisterTaskRequest) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *RegisterTaskRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_RegisterTaskRequest.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalTo(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (dst *RegisterTaskRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_RegisterTaskRequest.Merge(dst, src)
}
func (m *RegisterTaskRequest) XXX_Size() int {
	return m.Size()
}
func (m *RegisterTaskRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_RegisterTaskRequest.DiscardUnknown(m)
}

var xxx_messageInfo_RegisterTaskRequest proto.InternalMessageInfo

func (m *RegisterTaskRequest) GetTaskName() string {
	if m != nil {
		return m.TaskName
	}
	return ""
}

func (m *RegisterTaskRequest) GetURI() string {
	if m != nil {
		return m.URI
	}
	return ""
}

type OutputEventsRequest struct {
	// EventIDs is a list of event IDs.
	// The response will include only events that are (indirectly) related
	// to any of these event IDs.
	EventIDs []string `protobuf:"bytes,1,rep,name=EventIDs" json:"EventIDs,omitempty"`
	// TaskNames is a list of names of tasks.
	// The response will include only events that are created by one of these tasks.
	TaskNames []string `protobuf:"bytes,2,rep,name=TaskNames" json:"TaskNames,omitempty"`
	// If set, only events created after this timestamp are returned.
	CreatedAfter *types.Timestamp `protobuf:"bytes,3,opt,name=CreatedAfter" json:"CreatedAfter,omitempty"`
	// If set, only events created before this timestamp are returned.
	CreatedBefore *types.Timestamp `protobuf:"bytes,4,opt,name=CreatedBefore" json:"CreatedBefore,omitempty"`
}

func (m *OutputEventsRequest) Reset()         { *m = OutputEventsRequest{} }
func (m *OutputEventsRequest) String() string { return proto.CompactTextString(m) }
func (*OutputEventsRequest) ProtoMessage()    {}
func (*OutputEventsRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_agent_api_ab421025fe7e89fb, []int{2}
}
func (m *OutputEventsRequest) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *OutputEventsRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_OutputEventsRequest.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalTo(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (dst *OutputEventsRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_OutputEventsRequest.Merge(dst, src)
}
func (m *OutputEventsRequest) XXX_Size() int {
	return m.Size()
}
func (m *OutputEventsRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_OutputEventsRequest.DiscardUnknown(m)
}

var xxx_messageInfo_OutputEventsRequest proto.InternalMessageInfo

func (m *OutputEventsRequest) GetEventIDs() []string {
	if m != nil {
		return m.EventIDs
	}
	return nil
}

func (m *OutputEventsRequest) GetTaskNames() []string {
	if m != nil {
		return m.TaskNames
	}
	return nil
}

func (m *OutputEventsRequest) GetCreatedAfter() *types.Timestamp {
	if m != nil {
		return m.CreatedAfter
	}
	return nil
}

func (m *OutputEventsRequest) GetCreatedBefore() *types.Timestamp {
	if m != nil {
		return m.CreatedBefore
	}
	return nil
}

type OutputEvents struct {
	// The selected events
	Events []*event.Event `protobuf:"bytes,1,rep,name=Events" json:"Events,omitempty"`
}

func (m *OutputEvents) Reset()         { *m = OutputEvents{} }
func (m *OutputEvents) String() string { return proto.CompactTextString(m) }
func (*OutputEvents) ProtoMessage()    {}
func (*OutputEvents) Descriptor() ([]byte, []int) {
	return fileDescriptor_agent_api_ab421025fe7e89fb, []int{3}
}
func (m *OutputEvents) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *OutputEvents) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_OutputEvents.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalTo(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (dst *OutputEvents) XXX_Merge(src proto.Message) {
	xxx_messageInfo_OutputEvents.Merge(dst, src)
}
func (m *OutputEvents) XXX_Size() int {
	return m.Size()
}
func (m *OutputEvents) XXX_DiscardUnknown() {
	xxx_messageInfo_OutputEvents.DiscardUnknown(m)
}

var xxx_messageInfo_OutputEvents proto.InternalMessageInfo

func (m *OutputEvents) GetEvents() []*event.Event {
	if m != nil {
		return m.Events
	}
	return nil
}

func init() {
	proto.RegisterType((*RegisterLinkRequest)(nil), "pipeline.RegisterLinkRequest")
	proto.RegisterType((*RegisterTaskRequest)(nil), "pipeline.RegisterTaskRequest")
	proto.RegisterType((*OutputEventsRequest)(nil), "pipeline.OutputEventsRequest")
	proto.RegisterType((*OutputEvents)(nil), "pipeline.OutputEvents")
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// AgentRegistryClient is the client API for AgentRegistry service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type AgentRegistryClient interface {
	// Register an instance of a link agent
	RegisterLink(ctx context.Context, in *RegisterLinkRequest, opts ...grpc.CallOption) (*empty.Empty, error)
	// Register an instance of a task agent
	RegisterTask(ctx context.Context, in *RegisterTaskRequest, opts ...grpc.CallOption) (*empty.Empty, error)
}

type agentRegistryClient struct {
	cc *grpc.ClientConn
}

func NewAgentRegistryClient(cc *grpc.ClientConn) AgentRegistryClient {
	return &agentRegistryClient{cc}
}

func (c *agentRegistryClient) RegisterLink(ctx context.Context, in *RegisterLinkRequest, opts ...grpc.CallOption) (*empty.Empty, error) {
	out := new(empty.Empty)
	err := c.cc.Invoke(ctx, "/pipeline.AgentRegistry/RegisterLink", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *agentRegistryClient) RegisterTask(ctx context.Context, in *RegisterTaskRequest, opts ...grpc.CallOption) (*empty.Empty, error) {
	out := new(empty.Empty)
	err := c.cc.Invoke(ctx, "/pipeline.AgentRegistry/RegisterTask", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// AgentRegistryServer is the server API for AgentRegistry service.
type AgentRegistryServer interface {
	// Register an instance of a link agent
	RegisterLink(context.Context, *RegisterLinkRequest) (*empty.Empty, error)
	// Register an instance of a task agent
	RegisterTask(context.Context, *RegisterTaskRequest) (*empty.Empty, error)
}

func RegisterAgentRegistryServer(s *grpc.Server, srv AgentRegistryServer) {
	s.RegisterService(&_AgentRegistry_serviceDesc, srv)
}

func _AgentRegistry_RegisterLink_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RegisterLinkRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AgentRegistryServer).RegisterLink(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/pipeline.AgentRegistry/RegisterLink",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AgentRegistryServer).RegisterLink(ctx, req.(*RegisterLinkRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _AgentRegistry_RegisterTask_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RegisterTaskRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AgentRegistryServer).RegisterTask(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/pipeline.AgentRegistry/RegisterTask",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AgentRegistryServer).RegisterTask(ctx, req.(*RegisterTaskRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _AgentRegistry_serviceDesc = grpc.ServiceDesc{
	ServiceName: "pipeline.AgentRegistry",
	HandlerType: (*AgentRegistryServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "RegisterLink",
			Handler:    _AgentRegistry_RegisterLink_Handler,
		},
		{
			MethodName: "RegisterTask",
			Handler:    _AgentRegistry_RegisterTask_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "agent_api.proto",
}

// FrontendClient is the client API for Frontend service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type FrontendClient interface {
	// GetPipeline returns the pipeline resource.
	GetPipeline(ctx context.Context, in *empty.Empty, opts ...grpc.CallOption) (*v1alpha1.PipelineSpec, error)
	// GetOutputEvents returns all events (resulting from task outputs that
	// are not connected to inputs of other tasks) that match the given filter.
	GetOutputEvents(ctx context.Context, in *OutputEventsRequest, opts ...grpc.CallOption) (*OutputEvents, error)
}

type frontendClient struct {
	cc *grpc.ClientConn
}

func NewFrontendClient(cc *grpc.ClientConn) FrontendClient {
	return &frontendClient{cc}
}

func (c *frontendClient) GetPipeline(ctx context.Context, in *empty.Empty, opts ...grpc.CallOption) (*v1alpha1.PipelineSpec, error) {
	out := new(v1alpha1.PipelineSpec)
	err := c.cc.Invoke(ctx, "/pipeline.Frontend/GetPipeline", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *frontendClient) GetOutputEvents(ctx context.Context, in *OutputEventsRequest, opts ...grpc.CallOption) (*OutputEvents, error) {
	out := new(OutputEvents)
	err := c.cc.Invoke(ctx, "/pipeline.Frontend/GetOutputEvents", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// FrontendServer is the server API for Frontend service.
type FrontendServer interface {
	// GetPipeline returns the pipeline resource.
	GetPipeline(context.Context, *empty.Empty) (*v1alpha1.PipelineSpec, error)
	// GetOutputEvents returns all events (resulting from task outputs that
	// are not connected to inputs of other tasks) that match the given filter.
	GetOutputEvents(context.Context, *OutputEventsRequest) (*OutputEvents, error)
}

func RegisterFrontendServer(s *grpc.Server, srv FrontendServer) {
	s.RegisterService(&_Frontend_serviceDesc, srv)
}

func _Frontend_GetPipeline_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(empty.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FrontendServer).GetPipeline(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/pipeline.Frontend/GetPipeline",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FrontendServer).GetPipeline(ctx, req.(*empty.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _Frontend_GetOutputEvents_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(OutputEventsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FrontendServer).GetOutputEvents(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/pipeline.Frontend/GetOutputEvents",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FrontendServer).GetOutputEvents(ctx, req.(*OutputEventsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _Frontend_serviceDesc = grpc.ServiceDesc{
	ServiceName: "pipeline.Frontend",
	HandlerType: (*FrontendServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetPipeline",
			Handler:    _Frontend_GetPipeline_Handler,
		},
		{
			MethodName: "GetOutputEvents",
			Handler:    _Frontend_GetOutputEvents_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "agent_api.proto",
}

func (m *RegisterLinkRequest) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalTo(dAtA)
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *RegisterLinkRequest) MarshalTo(dAtA []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	if len(m.LinkName) > 0 {
		dAtA[i] = 0xa
		i++
		i = encodeVarintAgentApi(dAtA, i, uint64(len(m.LinkName)))
		i += copy(dAtA[i:], m.LinkName)
	}
	if len(m.URI) > 0 {
		dAtA[i] = 0x12
		i++
		i = encodeVarintAgentApi(dAtA, i, uint64(len(m.URI)))
		i += copy(dAtA[i:], m.URI)
	}
	return i, nil
}

func (m *RegisterTaskRequest) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalTo(dAtA)
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *RegisterTaskRequest) MarshalTo(dAtA []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	if len(m.TaskName) > 0 {
		dAtA[i] = 0xa
		i++
		i = encodeVarintAgentApi(dAtA, i, uint64(len(m.TaskName)))
		i += copy(dAtA[i:], m.TaskName)
	}
	if len(m.URI) > 0 {
		dAtA[i] = 0x12
		i++
		i = encodeVarintAgentApi(dAtA, i, uint64(len(m.URI)))
		i += copy(dAtA[i:], m.URI)
	}
	return i, nil
}

func (m *OutputEventsRequest) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalTo(dAtA)
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *OutputEventsRequest) MarshalTo(dAtA []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	if len(m.EventIDs) > 0 {
		for _, s := range m.EventIDs {
			dAtA[i] = 0xa
			i++
			l = len(s)
			for l >= 1<<7 {
				dAtA[i] = uint8(uint64(l)&0x7f | 0x80)
				l >>= 7
				i++
			}
			dAtA[i] = uint8(l)
			i++
			i += copy(dAtA[i:], s)
		}
	}
	if len(m.TaskNames) > 0 {
		for _, s := range m.TaskNames {
			dAtA[i] = 0x12
			i++
			l = len(s)
			for l >= 1<<7 {
				dAtA[i] = uint8(uint64(l)&0x7f | 0x80)
				l >>= 7
				i++
			}
			dAtA[i] = uint8(l)
			i++
			i += copy(dAtA[i:], s)
		}
	}
	if m.CreatedAfter != nil {
		dAtA[i] = 0x1a
		i++
		i = encodeVarintAgentApi(dAtA, i, uint64(m.CreatedAfter.Size()))
		n1, err := m.CreatedAfter.MarshalTo(dAtA[i:])
		if err != nil {
			return 0, err
		}
		i += n1
	}
	if m.CreatedBefore != nil {
		dAtA[i] = 0x22
		i++
		i = encodeVarintAgentApi(dAtA, i, uint64(m.CreatedBefore.Size()))
		n2, err := m.CreatedBefore.MarshalTo(dAtA[i:])
		if err != nil {
			return 0, err
		}
		i += n2
	}
	return i, nil
}

func (m *OutputEvents) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalTo(dAtA)
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *OutputEvents) MarshalTo(dAtA []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	if len(m.Events) > 0 {
		for _, msg := range m.Events {
			dAtA[i] = 0xa
			i++
			i = encodeVarintAgentApi(dAtA, i, uint64(msg.Size()))
			n, err := msg.MarshalTo(dAtA[i:])
			if err != nil {
				return 0, err
			}
			i += n
		}
	}
	return i, nil
}

func encodeVarintAgentApi(dAtA []byte, offset int, v uint64) int {
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return offset + 1
}
func (m *RegisterLinkRequest) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.LinkName)
	if l > 0 {
		n += 1 + l + sovAgentApi(uint64(l))
	}
	l = len(m.URI)
	if l > 0 {
		n += 1 + l + sovAgentApi(uint64(l))
	}
	return n
}

func (m *RegisterTaskRequest) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.TaskName)
	if l > 0 {
		n += 1 + l + sovAgentApi(uint64(l))
	}
	l = len(m.URI)
	if l > 0 {
		n += 1 + l + sovAgentApi(uint64(l))
	}
	return n
}

func (m *OutputEventsRequest) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if len(m.EventIDs) > 0 {
		for _, s := range m.EventIDs {
			l = len(s)
			n += 1 + l + sovAgentApi(uint64(l))
		}
	}
	if len(m.TaskNames) > 0 {
		for _, s := range m.TaskNames {
			l = len(s)
			n += 1 + l + sovAgentApi(uint64(l))
		}
	}
	if m.CreatedAfter != nil {
		l = m.CreatedAfter.Size()
		n += 1 + l + sovAgentApi(uint64(l))
	}
	if m.CreatedBefore != nil {
		l = m.CreatedBefore.Size()
		n += 1 + l + sovAgentApi(uint64(l))
	}
	return n
}

func (m *OutputEvents) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if len(m.Events) > 0 {
		for _, e := range m.Events {
			l = e.Size()
			n += 1 + l + sovAgentApi(uint64(l))
		}
	}
	return n
}

func sovAgentApi(x uint64) (n int) {
	for {
		n++
		x >>= 7
		if x == 0 {
			break
		}
	}
	return n
}
func sozAgentApi(x uint64) (n int) {
	return sovAgentApi(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *RegisterLinkRequest) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowAgentApi
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: RegisterLinkRequest: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: RegisterLinkRequest: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field LinkName", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowAgentApi
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= (uint64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthAgentApi
			}
			postIndex := iNdEx + intStringLen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.LinkName = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field URI", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowAgentApi
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= (uint64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthAgentApi
			}
			postIndex := iNdEx + intStringLen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.URI = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipAgentApi(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthAgentApi
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *RegisterTaskRequest) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowAgentApi
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: RegisterTaskRequest: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: RegisterTaskRequest: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field TaskName", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowAgentApi
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= (uint64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthAgentApi
			}
			postIndex := iNdEx + intStringLen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.TaskName = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field URI", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowAgentApi
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= (uint64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthAgentApi
			}
			postIndex := iNdEx + intStringLen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.URI = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipAgentApi(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthAgentApi
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *OutputEventsRequest) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowAgentApi
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: OutputEventsRequest: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: OutputEventsRequest: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field EventIDs", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowAgentApi
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= (uint64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthAgentApi
			}
			postIndex := iNdEx + intStringLen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.EventIDs = append(m.EventIDs, string(dAtA[iNdEx:postIndex]))
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field TaskNames", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowAgentApi
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= (uint64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthAgentApi
			}
			postIndex := iNdEx + intStringLen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.TaskNames = append(m.TaskNames, string(dAtA[iNdEx:postIndex]))
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field CreatedAfter", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowAgentApi
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthAgentApi
			}
			postIndex := iNdEx + msglen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.CreatedAfter == nil {
				m.CreatedAfter = &types.Timestamp{}
			}
			if err := m.CreatedAfter.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field CreatedBefore", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowAgentApi
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthAgentApi
			}
			postIndex := iNdEx + msglen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.CreatedBefore == nil {
				m.CreatedBefore = &types.Timestamp{}
			}
			if err := m.CreatedBefore.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipAgentApi(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthAgentApi
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *OutputEvents) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowAgentApi
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: OutputEvents: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: OutputEvents: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Events", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowAgentApi
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthAgentApi
			}
			postIndex := iNdEx + msglen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Events = append(m.Events, &event.Event{})
			if err := m.Events[len(m.Events)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipAgentApi(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthAgentApi
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func skipAgentApi(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowAgentApi
			}
			if iNdEx >= l {
				return 0, io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		wireType := int(wire & 0x7)
		switch wireType {
		case 0:
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowAgentApi
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if dAtA[iNdEx-1] < 0x80 {
					break
				}
			}
			return iNdEx, nil
		case 1:
			iNdEx += 8
			return iNdEx, nil
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowAgentApi
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				length |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			iNdEx += length
			if length < 0 {
				return 0, ErrInvalidLengthAgentApi
			}
			return iNdEx, nil
		case 3:
			for {
				var innerWire uint64
				var start int = iNdEx
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return 0, ErrIntOverflowAgentApi
					}
					if iNdEx >= l {
						return 0, io.ErrUnexpectedEOF
					}
					b := dAtA[iNdEx]
					iNdEx++
					innerWire |= (uint64(b) & 0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				innerWireType := int(innerWire & 0x7)
				if innerWireType == 4 {
					break
				}
				next, err := skipAgentApi(dAtA[start:])
				if err != nil {
					return 0, err
				}
				iNdEx = start + next
			}
			return iNdEx, nil
		case 4:
			return iNdEx, nil
		case 5:
			iNdEx += 4
			return iNdEx, nil
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
	}
	panic("unreachable")
}

var (
	ErrInvalidLengthAgentApi = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowAgentApi   = fmt.Errorf("proto: integer overflow")
)

func init() { proto.RegisterFile("agent_api.proto", fileDescriptor_agent_api_ab421025fe7e89fb) }

var fileDescriptor_agent_api_ab421025fe7e89fb = []byte{
	// 571 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x8c, 0x94, 0x41, 0x6b, 0x13, 0x41,
	0x14, 0xc7, 0x3b, 0x89, 0x94, 0x74, 0x92, 0x5a, 0x9d, 0x4a, 0x58, 0x96, 0xb8, 0x84, 0xe0, 0xa1,
	0x08, 0xce, 0x90, 0x54, 0x04, 0x15, 0xc4, 0xb4, 0xc6, 0x12, 0x29, 0xb6, 0xac, 0xf5, 0xe2, 0xa5,
	0x4c, 0xda, 0x97, 0xed, 0x36, 0x9b, 0x9d, 0x71, 0x77, 0x12, 0xc8, 0xd5, 0x8b, 0x57, 0xc1, 0xb3,
	0xdf, 0xc7, 0x63, 0xc5, 0x8b, 0x47, 0x49, 0xfc, 0x18, 0x1e, 0x64, 0x76, 0x77, 0x9a, 0x0d, 0xa6,
	0xb4, 0x97, 0x65, 0xde, 0xfc, 0xe7, 0xfd, 0x98, 0xf7, 0xde, 0x7f, 0x16, 0x6f, 0x70, 0x0f, 0x42,
	0x75, 0xcc, 0xa5, 0x4f, 0x65, 0x24, 0x94, 0x20, 0x25, 0xe9, 0x4b, 0x08, 0xfc, 0x10, 0xec, 0x7d,
	0xcf, 0x57, 0x67, 0xa3, 0x1e, 0x3d, 0x11, 0x43, 0xd6, 0x0e, 0xce, 0x79, 0x2f, 0xea, 0x1e, 0xb0,
	0x81, 0xe0, 0xc1, 0x39, 0x7f, 0x24, 0x24, 0x44, 0x5c, 0x89, 0x88, 0xc9, 0x81, 0xc7, 0xb8, 0xf4,
	0xe3, 0x4c, 0x60, 0xe3, 0x26, 0x0f, 0xe4, 0x19, 0x6f, 0x32, 0x0f, 0x42, 0x7d, 0x04, 0x4e, 0x53,
	0xae, 0xfd, 0x3c, 0x47, 0xf3, 0x44, 0xc0, 0x43, 0x8f, 0x25, 0x42, 0x6f, 0xd4, 0x67, 0x52, 0x4d,
	0x24, 0xc4, 0x4c, 0xf9, 0x43, 0x88, 0x15, 0x1f, 0xca, 0xf9, 0x2a, 0x4b, 0xde, 0xbe, 0x3e, 0x19,
	0x86, 0x52, 0x4d, 0xd2, 0x6f, 0x96, 0xf4, 0xf4, 0xa6, 0xf7, 0x87, 0x31, 0x84, 0x2a, 0xfd, 0x66,
	0xa9, 0x35, 0x4f, 0x08, 0x2f, 0x00, 0x5d, 0x19, 0xe3, 0x61, 0x28, 0x14, 0x57, 0xbe, 0x08, 0xe3,
	0x54, 0x6d, 0xec, 0xe2, 0x4d, 0x17, 0x3c, 0x3f, 0x56, 0x10, 0xed, 0xfb, 0xe1, 0xc0, 0x85, 0x8f,
	0x23, 0x88, 0x15, 0xb1, 0x71, 0x49, 0x87, 0x6f, 0xf9, 0x10, 0x2c, 0x54, 0x47, 0x5b, 0x6b, 0xee,
	0x65, 0x4c, 0xee, 0xe0, 0xe2, 0x7b, 0xb7, 0x6b, 0x15, 0x92, 0x6d, 0xbd, 0xcc, 0x43, 0x8e, 0x78,
	0x9c, 0x87, 0xe8, 0x30, 0x0f, 0x31, 0xf1, 0x12, 0xc8, 0x0f, 0x84, 0x37, 0x0f, 0x46, 0x4a, 0x8e,
	0x54, 0x47, 0xdf, 0x3e, 0xce, 0x51, 0x92, 0x8d, 0xee, 0xab, 0xd8, 0x42, 0xf5, 0xa2, 0xa6, 0x98,
	0x98, 0xd4, 0xf0, 0x9a, 0x21, 0xc6, 0x56, 0x21, 0x11, 0xe7, 0x1b, 0xe4, 0x05, 0xae, 0xec, 0x46,
	0xa0, 0xe7, 0xd6, 0xee, 0x2b, 0x88, 0xac, 0x62, 0x1d, 0x6d, 0x95, 0x5b, 0x36, 0x4d, 0x1b, 0x42,
	0x4d, 0xd7, 0xe9, 0x91, 0x99, 0x90, 0xbb, 0x70, 0x9e, 0xbc, 0xc4, 0xeb, 0x59, 0xbc, 0x03, 0x7d,
	0x11, 0x81, 0x75, 0xeb, 0x5a, 0xc0, 0x62, 0x42, 0xe3, 0x31, 0xae, 0xe4, 0x4b, 0x22, 0x0f, 0xf0,
	0x6a, 0xba, 0x4a, 0x2a, 0x29, 0xb7, 0x2a, 0x34, 0x9d, 0x54, 0xb2, 0xe9, 0x66, 0x5a, 0xeb, 0x1b,
	0xc2, 0xeb, 0x6d, 0x6d, 0xe5, 0xb4, 0xa9, 0xd1, 0x84, 0x74, 0x70, 0x25, 0x3f, 0x25, 0x72, 0x9f,
	0x1a, 0x67, 0xd3, 0x25, 0xd3, 0xb3, 0xab, 0xff, 0xdd, 0xb0, 0xa3, 0xbd, 0x94, 0xc7, 0xe8, 0x2e,
	0x2d, 0xc3, 0xe4, 0xe6, 0x77, 0x15, 0xa6, 0xf5, 0x17, 0xe1, 0xd2, 0xeb, 0x48, 0x84, 0x0a, 0xc2,
	0x53, 0xf2, 0x19, 0xe1, 0xf2, 0x1e, 0xa8, 0xc3, 0x0c, 0x44, 0xae, 0x48, 0xb2, 0xdf, 0xd0, 0xb9,
	0x85, 0xa9, 0xb1, 0x30, 0x4d, 0x2d, 0x7c, 0x6c, 0x2c, 0x4c, 0xe5, 0xc0, 0xa3, 0xfa, 0x09, 0x66,
	0x02, 0x35, 0x4f, 0x90, 0x1a, 0xfe, 0x3b, 0x09, 0x27, 0x8d, 0x7b, 0x9f, 0x7e, 0xfe, 0xf9, 0x5a,
	0xb8, 0x4d, 0x2a, 0x6c, 0xdc, 0x64, 0xa6, 0x04, 0xd2, 0xc7, 0x1b, 0x7b, 0xa0, 0x16, 0xfa, 0x9d,
	0x2b, 0x70, 0x89, 0xb5, 0xec, 0xea, 0x72, 0xb9, 0x51, 0x4b, 0xf8, 0xd5, 0xc6, 0x5d, 0xcd, 0x17,
	0x89, 0x92, 0x3e, 0xa9, 0xf8, 0x19, 0x7a, 0xb8, 0x73, 0xf8, 0x7d, 0xea, 0xa0, 0x8b, 0xa9, 0x83,
	0x7e, 0x4f, 0x1d, 0xf4, 0x65, 0xe6, 0xac, 0x5c, 0xcc, 0x9c, 0x95, 0x5f, 0x33, 0x67, 0xe5, 0xc3,
	0x93, 0x1b, 0xff, 0x65, 0xf4, 0x74, 0x2f, 0x6f, 0xde, 0x5b, 0x4d, 0x7a, 0xb5, 0xfd, 0x2f, 0x00,
	0x00, 0xff, 0xff, 0xec, 0x46, 0x49, 0xe9, 0xc1, 0x04, 0x00, 0x00,
}
