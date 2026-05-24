package registry

import (
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

func Load(r io.Reader) (*Registry, error) {
	var raw registryYAML
	if err := yaml.NewDecoder(r).Decode(&raw); err != nil {
		return nil, err
	}
	if raw.SchemaVersion != 1 {
		return nil, registryError(ErrUnsupportedSchemaVersion, "schema_version=%d", raw.SchemaVersion)
	}

	reg := &Registry{
		SchemaVersion: raw.SchemaVersion,
		Backends:      make(map[string]Backend, len(raw.Backends)),
		Models:        raw.Models,
		Persons:       raw.Persons,
		Profiles:      raw.Profiles,
		Devices:       raw.Devices,
	}
	for id, backend := range raw.Backends {
		if backend.APIKey != "" {
			return nil, registryError(ErrInlineSecret, "backend=%q field=api_key", id)
		}
		reg.Backends[id] = Backend{
			Type:      backend.Type,
			Endpoint:  backend.Endpoint,
			APIKeyRef: backend.APIKeyRef,
		}
	}
	return reg, nil
}

type registryYAML struct {
	SchemaVersion int                    `yaml:"schema_version"`
	Backends      map[string]backendYAML `yaml:"backends"`
	Models        map[string]Model       `yaml:"models"`
	Persons       map[string]Person      `yaml:"persons"`
	Profiles      map[string]Profile     `yaml:"profiles"`
	Devices       map[string]Device      `yaml:"devices"`
}

type backendYAML struct {
	Type      string `yaml:"type"`
	Endpoint  string `yaml:"endpoint"`
	APIKeyRef string `yaml:"api_key_ref"`
	APIKey    string `yaml:"api_key"`
}

func LoadFile(path string) (*Registry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Load(f)
}
