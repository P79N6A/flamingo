package msgtype

// NOTE: THIS FILE WAS PRODUCED BY THE
// MSGP CODE GENERATION TOOL (github.com/tinylib/msgp)
// DO NOT EDIT

import "github.com/tinylib/msgp/msgp"

// DecodeMsg implements msgp.Decodable
func (z *Instance) DecodeMsg(dc *msgp.Reader) (err error) {
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
		case "a":
			z.Addr, err = dc.ReadString()
			if err != nil {
				return
			}
		case "c":
			z.Cluster, err = dc.ReadString()
			if err != nil {
				return
			}
		case "w":
			z.Weight, err = dc.ReadUint16()
			if err != nil {
				return
			}
		case "u":
			z.UpdatedAt, err = dc.ReadInt64()
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
func (z *Instance) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 4
	// write "a"
	err = en.Append(0x84, 0xa1, 0x61)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Addr)
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
	// write "w"
	err = en.Append(0xa1, 0x77)
	if err != nil {
		return err
	}
	err = en.WriteUint16(z.Weight)
	if err != nil {
		return
	}
	// write "u"
	err = en.Append(0xa1, 0x75)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.UpdatedAt)
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *Instance) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 4
	// string "a"
	o = append(o, 0x84, 0xa1, 0x61)
	o = msgp.AppendString(o, z.Addr)
	// string "c"
	o = append(o, 0xa1, 0x63)
	o = msgp.AppendString(o, z.Cluster)
	// string "w"
	o = append(o, 0xa1, 0x77)
	o = msgp.AppendUint16(o, z.Weight)
	// string "u"
	o = append(o, 0xa1, 0x75)
	o = msgp.AppendInt64(o, z.UpdatedAt)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *Instance) UnmarshalMsg(bts []byte) (o []byte, err error) {
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
		case "a":
			z.Addr, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "c":
			z.Cluster, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "w":
			z.Weight, bts, err = msgp.ReadUint16Bytes(bts)
			if err != nil {
				return
			}
		case "u":
			z.UpdatedAt, bts, err = msgp.ReadInt64Bytes(bts)
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
func (z *Instance) Msgsize() (s int) {
	s = 1 + 2 + msgp.StringPrefixSize + len(z.Addr) + 2 + msgp.StringPrefixSize + len(z.Cluster) + 2 + msgp.Uint16Size + 2 + msgp.Int64Size
	return
}

// DecodeMsg implements msgp.Decodable
func (z *ServiceInstaces) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zcmr uint32
	zcmr, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for zcmr > 0 {
		zcmr--
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
		case "s":
			z.Service, err = dc.ReadString()
			if err != nil {
				return
			}
		case "n":
			z.Total, err = dc.ReadInt64()
			if err != nil {
				return
			}
		case "ii":
			var zajw uint32
			zajw, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.Instances) >= int(zajw) {
				z.Instances = (z.Instances)[:zajw]
			} else {
				z.Instances = make([]Instance, zajw)
			}
			for zbai := range z.Instances {
				err = z.Instances[zbai].DecodeMsg(dc)
				if err != nil {
					return
				}
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
func (z *ServiceInstaces) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 4
	// write "rid"
	err = en.Append(0x84, 0xa3, 0x72, 0x69, 0x64)
	if err != nil {
		return err
	}
	err = en.WriteString(z.RequestID)
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
	// write "n"
	err = en.Append(0xa1, 0x6e)
	if err != nil {
		return err
	}
	err = en.WriteInt64(z.Total)
	if err != nil {
		return
	}
	// write "ii"
	err = en.Append(0xa2, 0x69, 0x69)
	if err != nil {
		return err
	}
	err = en.WriteArrayHeader(uint32(len(z.Instances)))
	if err != nil {
		return
	}
	for zbai := range z.Instances {
		err = z.Instances[zbai].EncodeMsg(en)
		if err != nil {
			return
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *ServiceInstaces) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 4
	// string "rid"
	o = append(o, 0x84, 0xa3, 0x72, 0x69, 0x64)
	o = msgp.AppendString(o, z.RequestID)
	// string "s"
	o = append(o, 0xa1, 0x73)
	o = msgp.AppendString(o, z.Service)
	// string "n"
	o = append(o, 0xa1, 0x6e)
	o = msgp.AppendInt64(o, z.Total)
	// string "ii"
	o = append(o, 0xa2, 0x69, 0x69)
	o = msgp.AppendArrayHeader(o, uint32(len(z.Instances)))
	for zbai := range z.Instances {
		o, err = z.Instances[zbai].MarshalMsg(o)
		if err != nil {
			return
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *ServiceInstaces) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zwht uint32
	zwht, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zwht > 0 {
		zwht--
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
		case "s":
			z.Service, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "n":
			z.Total, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		case "ii":
			var zhct uint32
			zhct, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.Instances) >= int(zhct) {
				z.Instances = (z.Instances)[:zhct]
			} else {
				z.Instances = make([]Instance, zhct)
			}
			for zbai := range z.Instances {
				bts, err = z.Instances[zbai].UnmarshalMsg(bts)
				if err != nil {
					return
				}
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
func (z *ServiceInstaces) Msgsize() (s int) {
	s = 1 + 4 + msgp.StringPrefixSize + len(z.RequestID) + 2 + msgp.StringPrefixSize + len(z.Service) + 2 + msgp.Int64Size + 3 + msgp.ArrayHeaderSize
	for zbai := range z.Instances {
		s += z.Instances[zbai].Msgsize()
	}
	return
}
