package election

import (
	"errors"
	"log"
	"net"
	"os"
)

type Config struct {
	ListenAddr string
	Cluster    []string
	Logger     *log.Logger
}

func DefaultConfig() *Config {
	return &Config{
		ListenAddr: "127.0.0.1:55012",
		Cluster:    []string{},
		Logger:     log.New(os.Stderr, "[Election]", log.LstdFlags),
	}
}

func (c *Config) validate() error {
	if c.ListenAddr == "" {
		return errors.New("invalid ListenAddr")
	}

	if c.ListenAddr != "" {
		if _, err := net.ResolveUDPAddr("udp", c.ListenAddr); err != nil {
			return err
		}
	}

	if c.Cluster != nil {
		for _, addr := range c.Cluster {
			if _, err := net.ResolveUDPAddr("udp", addr); err != nil {
				return err
			}
		}
	}

	if c.Logger == nil {
		return errors.New("invalid Logger")
	}

	return nil
}
