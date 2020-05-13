package flatend

type Config struct {
	Codecs       map[string]*Codec
	CodecTypes   []string
	DefaultCodec string

	Before []Handler
	After  []Handler
}

func NewDefaultConfig() *Config {
	return &Config{
		Codecs:     Codecs,
		CodecTypes: CodecTypes,
	}
}
