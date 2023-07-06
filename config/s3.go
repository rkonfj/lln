package config

type S3Config struct {
	Endpoint        string `yaml:"endpoint"`
	Region          string `yaml:"region"`
	Bucket          string `yaml:"bucket"`
	AccessKeyID     string `yaml:"accessKeyID"`
	SecretAccessKey string `yaml:"secretAccessKey"`
}
