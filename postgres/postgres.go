package fbcpostgres

import (
	"encoding/json"
	"fmt"
	log "github.com/EntropyPool/entropy-logger"
	"github.com/google/uuid"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"golang.org/x/xerrors"
	"time"
)

type PostgresConfig struct {
	Host   string `json:"host"`
	User   string `json:"user"`
	Passwd string `json:"passwd"`
	DbName string `json:"db"`
}

type PostgresCli struct {
	config PostgresConfig
	url    string
	db     *gorm.DB
}

func NewPostgresCli(config PostgresConfig) *PostgresCli {
	cli := &PostgresCli{
		config: config,
		url: fmt.Sprintf("postgres://%v:%v@%v/%v",
			config.User, config.Passwd, config.Host, config.DbName),
	}

	log.Infof(log.Fields{}, "open postgres db %v", cli.url)
	db, err := gorm.Open("postgres", cli.url)
	if err != nil {
		log.Errorf(log.Fields{}, "cannot open %v: %v", cli.url, err)
		return nil
	}

	log.Infof(log.Fields{}, "successful to create postgres db %v", cli.url)
	db.SingularTable(true)
	cli.db = db

	return cli
}

func (cli *PostgresCli) Delete() {
	cli.db.Close()
}
