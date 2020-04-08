package flatend

import (
	"fmt"
	"go.starlark.net/starlark"
)

type Config struct {
	main     *starlark.Thread
	vars     starlark.StringDict
	handlers map[string]func()

	program *starlark.Program
}

func (c *Config) declareBuiltins() {
	c.vars["sql"] = starlark.NewBuiltin("sql", SQL)
	c.vars["http_get"] = starlark.NewBuiltin("http_get", GET)
}

func LoadConfig(src interface{}) (*Config, error) {
	cfg := &Config{
		main:     &starlark.Thread{Name: "main"},
		vars:     make(starlark.StringDict),
		handlers: make(map[string]func()),
	}

	cfg.declareBuiltins()

	_, program, err := starlark.SourceProgram("", src, cfg.vars.Has)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	cfg.program = program

	_, err = program.Init(cfg.main, cfg.vars)
	if err != nil {
		return nil, fmt.Errorf("failed to init config: %w", err)
	}

	return cfg, nil
}
