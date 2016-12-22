package config

type Config struct {
	Cpu, MaxBuf, Reads int
	Uniq               bool
}

func NewConfig(cpu, maxBuf, reads int, uniq bool) *Config {
	return &Config{cpu, maxBuf, reads, uniq}
}
