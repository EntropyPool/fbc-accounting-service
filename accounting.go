package main

import (
	"github.com/NpoolDevOps/fbc-accounting-service/fbcpostgres"
)

type AccountingConfig struct {
	PostgresConfig PostgresConfig `json:"postgres"`
	Port           int
}

type AccountingServer struct {
	config      AccountingConfig
	PostgresCli *PostgresCli
}

func NewAccountingServer(configFile string) *AccountingServer {
	server := &AccountingServer{}
	return server
}
