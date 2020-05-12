package flatend

type Config struct {
	Codecs       map[string]*Codec
	CodecTypes   []string
	DefaultCodec string

	Handlers []Handler
}

func NewDefaultConfig() *Config {
	return &Config{
		Codecs:     Codecs,
		CodecTypes: CodecTypes,
	}
}
