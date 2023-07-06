package config

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

var (
	Conf *Config
)

type Config struct {
	Listen string      `yaml:"listen"`
	OIDC   []*OIDC     `yaml:"oidc"`
	State  StateConfig `yaml:"state"`
	Model  ModelConfig `yaml:"model"`
	Admins []string    `yaml:"admins"`
	S3     S3Config    `yaml:"s3"`
}

type StateConfig struct {
	Etcd *EtcdConfig `yaml:"etcd"`
}
type EtcdConfig struct {
	Endpoints     []string `yaml:"endpoints"`
	CertFile      string   `yaml:"certFile,omitempty"`
	KeyFile       string   `yaml:"keyFile,omitempty"`
	TrustedCAFile string   `yaml:"trustedCAFile,omitempty"`
}

// LoadConfig init config package and export `config.Conf`
func LoadConfig(configPath string) error {
	logrus.Info("loading config from ", configPath)
	Conf = &Config{}
	b, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal([]byte(os.ExpandEnv(string(b))), Conf)
	if err != nil {
		return err
	}

	if Conf.State.Etcd == nil {
		Conf.State.Etcd = &EtcdConfig{Endpoints: []string{"http://127.0.0.1:2379"}}
	}

	initModel()

	initOpenIDConnect()
	return err
}

type Restriction struct {
	Status StatusConfig `json:"status"`
}

func GetRestriction(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(Restriction{
		Status: Conf.Model.Status,
	})
}
