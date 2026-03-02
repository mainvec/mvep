package protobuf

import (
	"errors"

	enc "github.com/mainvec/ugo/oencoding"
	"google.golang.org/protobuf/proto"
)

var (
	ErrInvalidProtoMsgEncode = errors.New("invalid protobuf proto.Message object passed to encode")
	ErrInvalidProtoMsgDecode = errors.New("invalid protobuf proto.Message object passed to decode")
)

func init() {
	proto := &ProtobufEncoding{}
	enc.RegisterEncoding("application/x-protobuf", proto)
	enc.RegisterEncoding("application/protobuf", proto)
	enc.RegisterEncoding("protobuf", proto)

}

type ProtobufEncoding struct {
}

// Encode
func (p *ProtobufEncoding) Encode(v any) ([]byte, error) {
	if v == nil {
		return nil, nil
	}
	i, found := v.(proto.Message)
	if !found {
		return nil, ErrInvalidProtoMsgEncode
	}

	b, err := proto.Marshal(i)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// Decode
func (p *ProtobufEncoding) Decode(data []byte, vPtr any) error {
	if _, ok := vPtr.(*any); ok {
		return nil
	}
	i, found := vPtr.(proto.Message)
	if !found {
		return ErrInvalidProtoMsgDecode
	}

	return proto.Unmarshal(data, i)
}

// MimeType
func (p *ProtobufEncoding) MimeType() string {
	return "application/x-protobuf"
}
