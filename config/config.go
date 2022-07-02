package config

type Config struct {
	Servers map[string]*Server
	Binary  Binary
	Dns     Dns
}

type Binary struct {
	BinaryPath  string `toml:"binary_path"`
	Params      string
	ParamsOrder string `toml:"params_order"`
}

type Server struct {
	IP        string
	Password  string
	User      string
	MaxChunks int `toml:"max_chunks"`
}

type Dns struct {
	Hosts []string
}
