package server

import (
	"os"
	"testing"

	. "github.com/onsi/gomega"
)

func TestLoadEmptyConfig(t *testing.T) {
	RegisterTestingT(t)

	rpConf := EmptyConfig()
	err := LoadConfig(rpConf)
	Ω(err).ShouldNot(HaveOccurred())
}

func TestLoadConfigWithParameters(t *testing.T) {
	os.Setenv("RP_PARAMETERS_PARAM", "env_value")

	rpConf := struct {
		*Config
		Param string `env:"RP_PARAMETERS_PARAM"`
	}{Config: EmptyConfig()}

	err := LoadConfig(&rpConf)
	Ω(err).ShouldNot(HaveOccurred())

	if "env_value" != rpConf.Param {
		t.Error("Config parser fails")
	}
}

func TestLoadConfigNonExisting(t *testing.T) {
	rpConf := EmptyConfig()
	err := LoadConfig(rpConf)
	Ω(err).ShouldNot(HaveOccurred())

	if 8080 != rpConf.Port {
		t.Error("Should not return empty string for default config")
	}
}

func TestLoadConfigIncorrectFormat(t *testing.T) {
	rpConf := EmptyConfig()
	err := LoadConfig(rpConf)
	Ω(err).ShouldNot(HaveOccurred())

	if 8080 != rpConf.Port {
		t.Error("Should return empty string for default config")
	}
}
