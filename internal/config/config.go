package config

import (
	"github.com/kelseyhightower/envconfig"
	"time"
)

type Config struct {
	DBHost           string        `envconfig:"DB_HOST" default:"127.0.0.1"`
	DBPort           string        `envconfig:"DB_PORT" default:"5432"`
	DBName           string        `envconfig:"DB_NAME" default:"mesh_group"`
	DBUser           string        `envconfig:"DB_USER" default:"postgres"`
	DBPassword       string        `envconfig:"DB_PASSWORD" default:"postgres"`
	ConnURI          string        `envconfig:"CONN_URI" default:"http://bsm.api.iql.ru/ords/bsm/segmentation/get_segmentation"`
	ConnAuthLoginPwd string        `envconfig:"CONN_AUTH_LOGIN_PWD" default:"4Dfddf5:jKlljHGH"`
	ConnUserAgent    string        `envconfig:"CONN_USER_AGENT" default:"spacecount-test"`
	ConnTimeout      time.Duration `envconfig:"CONN_TIMEOUT" default:"5s"`
	ConnInterval     time.Duration `envconfig:"CONN_INTERVAL" default:"1500ms"`
	ImportBatchSize  int           `envconfig:"IMPORT_BATCH_SIZE" default:"50"`
	LogCleanupMaxAge int           `envconfig:"LOG_CLEANUP_MAX_AGE" default:"7"`
}

func LoadConfig() (*Config, error) {
	var cfg Config
	err := envconfig.Process("", &cfg)
	return &cfg, err
}
