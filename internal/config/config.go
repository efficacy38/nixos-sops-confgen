package config

type Config struct {
	User   User             `yaml:"user"`
	Groups map[string]Group `mapstructure:"groups" yaml:"groups"`
	Hosts  map[string]Host  `mapstructure:"hosts"  yaml:"hosts"`
}

type User struct {
	PersonalEncGpgKey string `mapstructure:"personal_enc_pgp_key" yaml:"personal_enc_pgp_key"`
}

type Group struct {
	PathRegex string `mapstructure:"path_regex" yaml:"path_regex"`
}

type Host struct {
	Key    string   `yaml:"key"`
	Groups []string `yaml:"groups"`
}
