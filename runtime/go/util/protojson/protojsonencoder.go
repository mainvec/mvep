package protojson

import (
	"github.com/mainvec/mvp/mvpgo/util/protobuf"
	enc "github.com/mainvec/ugo/oencoding"
	protojson "google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

var ()

func init() {
	proto := &ProtoJsonEncoding{}
	// Register under the short name only. Do NOT register under
	// "application/json" — that MIME type belongs to the plain JSON encoder
	// (github.com/mainvec/ugo/oencoding/json). Clients that want protojson
	// must request it explicitly by name (Content-Type: protojson).
	enc.RegisterEncoding("protojson", proto)

}

type ProtoJsonEncoding struct {
}

// Encode
func (p *ProtoJsonEncoding) Encode(v any) ([]byte, error) {
	if v == nil {
		return nil, nil
	}
	i, found := v.(proto.Message)
	if !found {
		return nil, protobuf.ErrInvalidProtoMsgEncode
	}

	b, err := protojson.Marshal(i)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// Decode
func (p *ProtoJsonEncoding) Decode(data []byte, vPtr any) error {
	if _, ok := vPtr.(*any); ok {
		return nil
	}
	i, found := vPtr.(proto.Message)
	if !found {
		return protobuf.ErrInvalidProtoMsgDecode
	}

	return protojson.Unmarshal(data, i)
}

// MimeType
func (p *ProtoJsonEncoding) MimeType() string {
	return "application/json"
}
