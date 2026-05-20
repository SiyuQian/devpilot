package envelope

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v5"

	"github.com/siyuqian/devpilot/internal/graph/envelope/schemas"
)

var (
	compilerOnce sync.Once
	compiler     *jsonschema.Compiler
	compileErr   error
)

func loadCompiler() (*jsonschema.Compiler, error) {
	compilerOnce.Do(func() {
		c := jsonschema.NewCompiler()
		entries, err := fs.ReadDir(schemas.FS, ".")
		if err != nil {
			compileErr = fmt.Errorf("read schemas dir: %w", err)
			return
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			data, err := fs.ReadFile(schemas.FS, e.Name())
			if err != nil {
				compileErr = fmt.Errorf("read %s: %w", e.Name(), err)
				return
			}
			if err := c.AddResource(e.Name(), bytes.NewReader(data)); err != nil {
				compileErr = fmt.Errorf("add resource %s: %w", e.Name(), err)
				return
			}
		}
		compiler = c
	})
	return compiler, compileErr
}

// Validate parses raw JSON and asserts it conforms to the schema with the given $id.
func Validate(raw []byte, schemaID string) error {
	c, err := loadCompiler()
	if err != nil {
		return err
	}
	sch, err := c.Compile(schemaID)
	if err != nil {
		return fmt.Errorf("compile %s: %w", schemaID, err)
	}
	var doc any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return fmt.Errorf("unmarshal envelope: %w", err)
	}
	if err := sch.Validate(doc); err != nil {
		return fmt.Errorf("validate %s: %w", schemaID, err)
	}
	return nil
}
