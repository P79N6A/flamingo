package msgtype

// NOTE: THIS FILE WAS PRODUCED BY THE
// MSGP CODE GENERATION TOOL (github.com/tinylib/msgp)
// DO NOT EDIT

import (
	"github.com/tinylib/msgp/msgp"
)

// DecodeMsg implements msgp.Decodable
func (z *Request) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zxvk uint32
	zxvk, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for zxvk > 0 {
		zxvk--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "rid":
			z.RequestID, err = dc.ReadString()
			if err != nil {
				return
			}
		case "op":
			z.Op, err = dc.ReadUint8()
			if err != nil {
				return
			}
		case "s":
			z.Service, err = dc.ReadString()
			if err != nil {
				return
			}
		case "c":
			z.Cluster, err = dc.ReadString()
			if err != nil {
				return
			}
		case "nc":
			z.NoCompress, err = dc.ReadBool()
			if err != nil {
				return
			}
		case "ss":
			z.SingleShot, err = dc.ReadBool()
			if err != nil {
				return
			}
		case "l":
			z.Limit, err = dc.ReadUint16()
			if err != nil {
				return
			}
		default:
			err = dc.Skip()
			if err != nil {
				return
			}
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z *Request) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 7
	// write "rid"
	err = en.Append(0x87, 0xa3, 0x72, 0x69, 0x64)
	if err != nil {
		return err
	}
	err = en.WriteString(z.RequestID)
	if err != nil {
		return
	}
	// write "op"
	err = en.Append(0xa2, 0x6f, 0x70)
	if err != nil {
		return err
	}
	err = en.WriteUint8(z.Op)
	if err != nil {
		return
	}
	// write "s"
	err = en.Append(0xa1, 0x73)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Service)
	if err != nil {
		return
	}
	// write "c"
	err = en.Append(0xa1, 0x63)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Cluster)
	if err != nil {
		return
	}
	// write "nc"
	err = en.Append(0xa2, 0x6e, 0x63)
	if err != nil {
		return err
	}
	err = en.WriteBool(z.NoCompress)
	if err != nil {
		return
	}
	// write "ss"
	err = en.Append(0xa2, 0x73, 0x73)
	if err != nil {
		return err
	}
	err = en.WriteBool(z.SingleShot)
	if err != nil {
		return
	}
	// write "l"
	err = en.Append(0xa1, 0x6c)
	if err != nil {
		return err
	}
	err = en.WriteUint16(z.Limit)
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *Request) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 7
	// string "rid"
	o = append(o, 0x87, 0xa3, 0x72, 0x69, 0x64)
	o = msgp.AppendString(o, z.RequestID)
	// string "op"
	o = append(o, 0xa2, 0x6f, 0x70)
	o = msgp.AppendUint8(o, z.Op)
	// string "s"
	o = append(o, 0xa1, 0x73)
	o = msgp.AppendString(o, z.Service)
	// string "c"
	o = append(o, 0xa1, 0x63)
	o = msgp.AppendString(o, z.Cluster)
	// string "nc"
	o = append(o, 0xa2, 0x6e, 0x63)
	o = msgp.AppendBool(o, z.NoCompress)
	// string "ss"
	o = append(o, 0xa2, 0x73, 0x73)
	o = msgp.AppendBool(o, z.SingleShot)
	// string "l"
	o = append(o, 0xa1, 0x6c)
	o = msgp.AppendUint16(o, z.Limit)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *Request) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zbzg uint32
	zbzg, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zbzg > 0 {
		zbzg--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "rid":
			z.RequestID, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "op":
			z.Op, bts, err = msgp.ReadUint8Bytes(bts)
			if err != nil {
				return
			}
		case "s":
			z.Service, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "c":
			z.Cluster, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "nc":
			z.NoCompress, bts, err = msgp.ReadBoolBytes(bts)
			if err != nil {
				return
			}
		case "ss":
			z.SingleShot, bts, err = msgp.ReadBoolBytes(bts)
			if err != nil {
				return
			}
		case "l":
			z.Limit, bts, err = msgp.ReadUint16Bytes(bts)
			if err != nil {
				return
			}
		default:
			bts, err = msgp.Skip(bts)
			if err != nil {
				return
			}
		}
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *Request) Msgsize() (s int) {
	s = 1 + 4 + msgp.StringPrefixSize + len(z.RequestID) + 3 + msgp.Uint8Size + 2 + msgp.StringPrefixSize + len(z.Service) + 2 + msgp.StringPrefixSize + len(z.Cluster) + 3 + msgp.BoolSize + 3 + msgp.BoolSize + 2 + msgp.Uint16Size
	return
}
