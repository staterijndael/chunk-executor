package config

type Config struct {
	Servers map[string]*Server
}

type Server struct {
	IP        string
	Password  string
	User      string
	MaxChunks int `toml:"max_chunks"`
}
