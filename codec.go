package flatend

import (
	"bytes"
	"encoding/csv"
	"encoding/gob"
	"encoding/json"
	"encoding/xml"
	"fmt"
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
	registerCodec(reflectCodec(yaml.Marshal, yaml.Unmarshal, "application/x-yaml", "text/x-yaml"))
	registerCodec(reflectCodec(xml.Marshal, xml.Unmarshal, "application/xml", "text/xml"))
	registerCodec(newCodec(csvEncoder, csvDecoder, "application/csv", "text/csv"))
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

func csvEncoder(values Values) ([]byte, error) {
	records := [][]string{make([]string, 0, len(values)), make([]string, 0, len(values))}
	keys, vals := records[0], records[1]

	for k, v := range values {
		keys = append(keys, k)
		vals = append(vals, fmt.Sprint(v))
	}

	var b bytes.Buffer
	if err := csv.NewWriter(&b).WriteAll(records); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

func csvDecoder(src []byte, values Values) error {
	r := csv.NewReader(bytes.NewReader(src))

	keys, err := r.Read()
	if err != nil {
		return fmt.Errorf("csv: failed to read keys: %w", err)
	}

	r.FieldsPerRecord = len(keys)

	vals, err := r.Read()
	if err != nil {
		return fmt.Errorf("csv: failed to read values: %w", err)
	}

	for i := 0; i < len(keys); i++ {
		values[keys[i]] = vals[i]
	}

	return nil
}

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
