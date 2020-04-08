package flatend

import (
	"fmt"
	"go.starlark.net/starlark"
)

type Config struct {
	handlers map[string]func()
}

func (c *Config) declareStarlarkBuiltins() starlark.StringDict {
	return starlark.StringDict{
		"sql":      starlark.NewBuiltin("sql", SQL),
		"http_get": starlark.NewBuiltin("http_get", GET),
	}
}

func (c *Config) parseStarlark(src interface{}) error {
	vars := c.declareStarlarkBuiltins()

	_, program, err := starlark.SourceProgram("", src, vars.Has)
	if err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	_, err = program.Init(&starlark.Thread{Name: "main"}, vars)
	if err != nil {
		return fmt.Errorf("failed to init config: %w", err)
	}

	return nil
}

func LoadConfig(src interface{}) (*Config, error) {
	cfg := &Config{handlers: make(map[string]func())}
	return cfg, cfg.parseStarlark(src)
}
