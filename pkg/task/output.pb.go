// Code generated by protoc-gen-go. DO NOT EDIT.
// source: output.proto

package task

import (
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	context "golang.org/x/net/context"
	grpc "google.golang.org/grpc"
	math "math"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

type OutputReadyRequest struct {
	// Data (content) of the annotated value.
	AnnotatedValueData string `protobuf:"bytes,1,opt,name=AnnotatedValueData,proto3" json:"AnnotatedValueData,omitempty"`
	// Name of the task output that is data belongs to.
	OutputName           string   `protobuf:"bytes,2,opt,name=OutputName,proto3" json:"OutputName,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *OutputReadyRequest) Reset()         { *m = OutputReadyRequest{} }
func (m *OutputReadyRequest) String() string { return proto.CompactTextString(m) }
func (*OutputReadyRequest) ProtoMessage()    {}
func (*OutputReadyRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_0b2b3ae2e703b013, []int{0}
}

func (m *OutputReadyRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_OutputReadyRequest.Unmarshal(m, b)
}
func (m *OutputReadyRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_OutputReadyRequest.Marshal(b, m, deterministic)
}
func (m *OutputReadyRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_OutputReadyRequest.Merge(m, src)
}
func (m *OutputReadyRequest) XXX_Size() int {
	return xxx_messageInfo_OutputReadyRequest.Size(m)
}
func (m *OutputReadyRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_OutputReadyRequest.DiscardUnknown(m)
}

var xxx_messageInfo_OutputReadyRequest proto.InternalMessageInfo

func (m *OutputReadyRequest) GetAnnotatedValueData() string {
	if m != nil {
		return m.AnnotatedValueData
	}
	return ""
}

func (m *OutputReadyRequest) GetOutputName() string {
	if m != nil {
		return m.OutputName
	}
	return ""
}

type OutputReadyResponse struct {
	// ID of the published annotated value
	AnnotatedValueID     string   `protobuf:"bytes,1,opt,name=AnnotatedValueID,proto3" json:"AnnotatedValueID,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *OutputReadyResponse) Reset()         { *m = OutputReadyResponse{} }
func (m *OutputReadyResponse) String() string { return proto.CompactTextString(m) }
func (*OutputReadyResponse) ProtoMessage()    {}
func (*OutputReadyResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_0b2b3ae2e703b013, []int{1}
}

func (m *OutputReadyResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_OutputReadyResponse.Unmarshal(m, b)
}
func (m *OutputReadyResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_OutputReadyResponse.Marshal(b, m, deterministic)
}
func (m *OutputReadyResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_OutputReadyResponse.Merge(m, src)
}
func (m *OutputReadyResponse) XXX_Size() int {
	return xxx_messageInfo_OutputReadyResponse.Size(m)
}
func (m *OutputReadyResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_OutputReadyResponse.DiscardUnknown(m)
}

var xxx_messageInfo_OutputReadyResponse proto.InternalMessageInfo

func (m *OutputReadyResponse) GetAnnotatedValueID() string {
	if m != nil {
		return m.AnnotatedValueID
	}
	return ""
}

func init() {
	proto.RegisterType((*OutputReadyRequest)(nil), "task.OutputReadyRequest")
	proto.RegisterType((*OutputReadyResponse)(nil), "task.OutputReadyResponse")
}

func init() { proto.RegisterFile("output.proto", fileDescriptor_0b2b3ae2e703b013) }

var fileDescriptor_0b2b3ae2e703b013 = []byte{
	// 218 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x6c, 0x90, 0x31, 0x4b, 0xc4, 0x40,
	0x10, 0x85, 0x39, 0x11, 0xc1, 0xd1, 0x42, 0xc6, 0x26, 0x5a, 0x88, 0x5c, 0x25, 0xa2, 0xbb, 0xa0,
	0xbf, 0x20, 0xc7, 0x35, 0xd7, 0xdc, 0x41, 0x0a, 0x41, 0xbb, 0x89, 0x19, 0xcf, 0x5c, 0x72, 0x99,
	0x75, 0x77, 0xb6, 0xf0, 0xdf, 0x4b, 0x12, 0x8b, 0x84, 0xa4, 0xfd, 0xde, 0xf2, 0xbd, 0x7d, 0x03,
	0x97, 0x12, 0xd5, 0x45, 0x35, 0xce, 0x8b, 0x0a, 0x9e, 0x2a, 0x85, 0x6a, 0x59, 0x00, 0xee, 0x3a,
	0x9a, 0x31, 0x15, 0xbf, 0x19, 0xff, 0x44, 0x0e, 0x8a, 0x06, 0x30, 0x6d, 0x1a, 0x51, 0x52, 0x2e,
	0xde, 0xa8, 0x8e, 0xbc, 0x26, 0xa5, 0x64, 0x71, 0xbf, 0x78, 0x38, 0xcf, 0x66, 0x12, 0xbc, 0x03,
	0xe8, 0x2d, 0x5b, 0x3a, 0x72, 0x72, 0xd2, 0xbd, 0x1b, 0x90, 0x65, 0x0a, 0xd7, 0xa3, 0x96, 0xe0,
	0xa4, 0x09, 0x8c, 0x8f, 0x70, 0x35, 0x96, 0x6d, 0xd6, 0xff, 0x25, 0x13, 0xfe, 0xf2, 0x3e, 0x52,
	0x6c, 0x45, 0xcb, 0xaf, 0x92, 0x3d, 0xae, 0xe0, 0x62, 0x80, 0x31, 0x31, 0xed, 0x2a, 0x33, 0x9d,
	0x74, 0x7b, 0x33, 0x93, 0xf4, 0xdf, 0x58, 0x99, 0x8f, 0xa7, 0x7d, 0xa9, 0xdf, 0x31, 0x37, 0x9f,
	0x72, 0xb4, 0x69, 0x7d, 0xa0, 0xdc, 0x6f, 0x76, 0xb6, 0x12, 0xaa, 0x0f, 0xf4, 0x2c, 0x8e, 0x3d,
	0xa9, 0x78, 0xeb, 0xaa, 0xbd, 0x6d, 0x1d, 0xf9, 0x59, 0x77, 0xc0, 0xd7, 0xbf, 0x00, 0x00, 0x00,
	0xff, 0xff, 0x91, 0xad, 0xb2, 0x82, 0x50, 0x01, 0x00, 0x00,
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// OutputReadyNotifierClient is the client API for OutputReadyNotifier service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type OutputReadyNotifierClient interface {
	OutputReady(ctx context.Context, in *OutputReadyRequest, opts ...grpc.CallOption) (*OutputReadyResponse, error)
}

type outputReadyNotifierClient struct {
	cc *grpc.ClientConn
}

func NewOutputReadyNotifierClient(cc *grpc.ClientConn) OutputReadyNotifierClient {
	return &outputReadyNotifierClient{cc}
}

func (c *outputReadyNotifierClient) OutputReady(ctx context.Context, in *OutputReadyRequest, opts ...grpc.CallOption) (*OutputReadyResponse, error) {
	out := new(OutputReadyResponse)
	err := c.cc.Invoke(ctx, "/task.OutputReadyNotifier/OutputReady", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// OutputReadyNotifierServer is the server API for OutputReadyNotifier service.
type OutputReadyNotifierServer interface {
	OutputReady(context.Context, *OutputReadyRequest) (*OutputReadyResponse, error)
}

func RegisterOutputReadyNotifierServer(s *grpc.Server, srv OutputReadyNotifierServer) {
	s.RegisterService(&_OutputReadyNotifier_serviceDesc, srv)
}

func _OutputReadyNotifier_OutputReady_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(OutputReadyRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(OutputReadyNotifierServer).OutputReady(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/task.OutputReadyNotifier/OutputReady",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(OutputReadyNotifierServer).OutputReady(ctx, req.(*OutputReadyRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _OutputReadyNotifier_serviceDesc = grpc.ServiceDesc{
	ServiceName: "task.OutputReadyNotifier",
	HandlerType: (*OutputReadyNotifierServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "OutputReady",
			Handler:    _OutputReadyNotifier_OutputReady_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "output.proto",
}
