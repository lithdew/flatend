package flatend

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v3"
)

type (
	EncodeFunc = func(values Values) ([]byte, error)
	DecodeFunc = func(buf []byte, values Values) error
)

type Codec struct {
	Encode EncodeFunc
	Decode DecodeFunc
	Tags   []string
}

func (c Codec) Tag() string {
	return c.Tags[0]
}

var Codecs = make(map[string]*Codec)
var CodecTypes []string

func init() {
	registerCodec(reflectCodec(json.Marshal, json.Unmarshal, "application/json", "text/json"))
	registerCodec(reflectCodec(yaml.Marshal, yaml.Unmarshal,
		"application/x-yaml", "text/x-yaml", "application/yaml", "text/yaml",
		"application/x-yml", "text/x-yml", "application/yml", "text/yml",
	))
	registerCodec(newCodec(tomlEncoder, tomlDecoder,
		"application/x-toml", "text/x-toml", "application/toml", "text/x-toml",
	))
	//registerCodec(newCodec(csvEncoder, csvDecoder, "application/csv", "text/csv"))
	registerCodec(newCodec(gobEncoder, gobDecoder, "application/x-gob", "text/x-gob"))
}

func registerCodec(codec *Codec) {
	for _, tag := range codec.Tags {
		Codecs[tag] = codec
		CodecTypes = append(CodecTypes, tag)
	}
}

func reflectCodec(me func(src interface{}) ([]byte, error), md func(buf []byte, dst interface{}) error, tags ...string) *Codec {
	return &Codec{Encode: marshalEncoder(me), Decode: unmarshalDecoder(md), Tags: tags}
}

func newCodec(encoder EncodeFunc, decoder DecodeFunc, tags ...string) *Codec {
	return &Codec{Encode: encoder, Decode: decoder, Tags: tags}
}

func unmarshalDecoder(f func(buf []byte, dst interface{}) error) DecodeFunc {
	return func(buf []byte, values Values) error {
		return f(buf, values)
	}
}

func marshalEncoder(f func(src interface{}) ([]byte, error)) EncodeFunc {
	return func(values Values) ([]byte, error) {
		return f(values)
	}
}

//func csvEncoder(values Values) ([]byte, error) {
//	records := [][]string{make([]string, 0, len(values)), make([]string, 0, len(values))}
//
//	for k, v := range values {
//		s, ok := v.(fmt.Stringer)
//		if !ok {
//			continue
//		}
//		records[0] = append(records[0], k)
//		records[1] = append(records[1], s.String())
//	}
//
//	var b bytes.Buffer
//	if err := csv.NewWriter(&b).WriteAll(records); err != nil {
//		return nil, err
//	}
//
//	return b.Bytes(), nil
//}
//
//func csvDecoder(src []byte, values Values) error {
//	r := csv.NewReader(bytes.NewReader(src))
//
//	keys, err := r.Read()
//	if err != nil {
//		return fmt.Errorf("csv: failed to read keys: %w", err)
//	}
//
//	r.FieldsPerRecord = len(keys)
//
//	vals, err := r.Read()
//	if err != nil {
//		return fmt.Errorf("csv: failed to read values: %w", err)
//	}
//
//	for i := 0; i < len(keys); i++ {
//		values[keys[i]] = vals[i]
//	}
//
//	return nil
//}

func gobEncoder(values Values) ([]byte, error) {
	var b bytes.Buffer
	if err := gob.NewEncoder(&b).Encode(values); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func gobDecoder(src []byte, values Values) error {
	return gob.NewDecoder(bytes.NewReader(src)).Decode(&values)
}

func tomlEncoder(values Values) ([]byte, error) {
	var b bytes.Buffer
	if err := toml.NewEncoder(&b).Encode(values); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func tomlDecoder(src []byte, values Values) error {
	return toml.Unmarshal(src, values)
}
