// Code generated by protoc-gen-go.
// source: collector.proto
// DO NOT EDIT!

/*
Package collector is a generated protocol buffer package.

It is generated from these files:
	collector.proto

It has these top-level messages:
	ApplicationMessage
	RequestPayload
	ResponsePayload
*/
package databus_client

import proto "github.com/golang/protobuf/proto"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = math.Inf

type ApplicationMessage struct {
	Key              []byte `protobuf:"bytes,1,opt,name=key" json:"key,omitempty"`
	Value            []byte `protobuf:"bytes,2,req,name=value" json:"value,omitempty"`
	Codec            *int32 `protobuf:"varint,3,opt,name=codec,def=0" json:"codec,omitempty"`
	XXX_unrecognized []byte `json:"-"`
}

func (m *ApplicationMessage) Reset()         { *m = ApplicationMessage{} }
func (m *ApplicationMessage) String() string { return proto.CompactTextString(m) }
func (*ApplicationMessage) ProtoMessage()    {}

const Default_ApplicationMessage_Codec int32 = 0

func (m *ApplicationMessage) GetKey() []byte {
	if m != nil {
		return m.Key
	}
	return nil
}

func (m *ApplicationMessage) GetValue() []byte {
	if m != nil {
		return m.Value
	}
	return nil
}

func (m *ApplicationMessage) GetCodec() int32 {
	if m != nil && m.Codec != nil {
		return *m.Codec
	}
	return Default_ApplicationMessage_Codec
}

type RequestPayload struct {
	Channel          *string               `protobuf:"bytes,1,req,name=channel" json:"channel,omitempty"`
	Messages         []*ApplicationMessage `protobuf:"bytes,2,rep,name=messages" json:"messages,omitempty"`
	NeedResp         *int32                `protobuf:"varint,3,opt,name=need_resp" json:"need_resp,omitempty"`
	XXX_unrecognized []byte                `json:"-"`
}

func (m *RequestPayload) Reset()         { *m = RequestPayload{} }
func (m *RequestPayload) String() string { return proto.CompactTextString(m) }
func (*RequestPayload) ProtoMessage()    {}

func (m *RequestPayload) GetChannel() string {
	if m != nil && m.Channel != nil {
		return *m.Channel
	}
	return ""
}

func (m *RequestPayload) GetMessages() []*ApplicationMessage {
	if m != nil {
		return m.Messages
	}
	return nil
}

func (m *RequestPayload) GetNeedResp() int32 {
	if m != nil && m.NeedResp != nil {
		return *m.NeedResp
	}
	return 0
}

type ResponsePayload struct {
	Code             *int32 `protobuf:"varint,1,req,name=code" json:"code,omitempty"`
	XXX_unrecognized []byte `json:"-"`
}

func (m *ResponsePayload) Reset()         { *m = ResponsePayload{} }
func (m *ResponsePayload) String() string { return proto.CompactTextString(m) }
func (*ResponsePayload) ProtoMessage()    {}

func (m *ResponsePayload) GetCode() int32 {
	if m != nil && m.Code != nil {
		return *m.Code
	}
	return 0
}
