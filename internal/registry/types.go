package registry

// Registry is the schema v1 local YAML registry.
type Registry struct {
	SchemaVersion int                `yaml:"schema_version"`
	Backends      map[string]Backend `yaml:"backends"`
	Models        map[string]Model   `yaml:"models"`
	Persons       map[string]Person  `yaml:"persons"`
	Profiles      map[string]Profile `yaml:"profiles"`
	Devices       map[string]Device  `yaml:"devices"`
}

type Backend struct {
	Type      string `yaml:"type"`
	Endpoint  string `yaml:"endpoint"`
	APIKeyRef string `yaml:"api_key_ref"`
}

type Model struct {
	Backend string `yaml:"backend"`
	Name    string `yaml:"name"`
}

type Person struct {
	DisplayName string `yaml:"display_name"`
}

type Profile struct {
	Person       string `yaml:"person"`
	Model        string `yaml:"model"`
	SystemPrompt string `yaml:"system_prompt"`
}

type Device struct {
	Label          string                  `yaml:"label"`
	DefaultPerson  string                  `yaml:"default_person"`
	DefaultProfile string                  `yaml:"default_profile"`
	Aliases        map[string]AliasBinding `yaml:"aliases"`
}

type AliasBinding struct {
	Person  string `yaml:"person"`
	Profile string `yaml:"profile"`
}

type ResolvedContext struct {
	DeviceID string
	Alias    string

	PersonID  string
	Person    Person
	ProfileID string
	Profile   Profile
	ModelID   string
	Model     Model
	BackendID string
	Backend   Backend
}
