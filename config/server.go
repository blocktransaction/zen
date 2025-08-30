package config

type Server struct {
	Host string
	Port int
}

var ServerConfig = new(Server)
