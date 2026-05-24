package registry

import (
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

func Load(r io.Reader) (*Registry, error) {
	var reg Registry
	if err := yaml.NewDecoder(r).Decode(&reg); err != nil {
		return nil, err
	}
	if reg.SchemaVersion != 1 {
		return nil, registryError(ErrUnsupportedSchemaVersion, "schema_version=%d", reg.SchemaVersion)
	}
	return &reg, nil
}

func LoadFile(path string) (*Registry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Load(f)
}
