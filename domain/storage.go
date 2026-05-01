package domain

type StorageConfig struct {
	Path   string               `yaml:"path"`
	Tables map[string]*TableDef `yaml:"tables"`
	Type   string               `yaml:"type"`
}

type TableDef struct {
	Columns []string `yaml:"columns"`
	Source  string   `yaml:"source"`
}
