package fbcpostgres

import (
	_ "encoding/json"
	"fmt"
	log "github.com/EntropyPool/entropy-logger"
	_ "github.com/google/uuid"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"golang.org/x/xerrors"
	_ "golang.org/x/xerrors"
	"strings"
	_ "time"
)

type PostgresConfig struct {
	Host    string `json:"host"`
	User    string `json:"user"`
	Passwd  string `json:"passwd"`
	DbName  string `json:"db"`
	Port    string `json:"port"`
	Sslmode string `json:"sslmode"`
}

type PostgresCli struct {
	config PostgresConfig
	url    string
	db     *gorm.DB
}

func NewPostgresCli(config PostgresConfig) *PostgresCli {
	cli := &PostgresCli{
		config: config,
		url: fmt.Sprintf("postgres://%v:%v@%v:%v/%v?sslmode=%v",
			config.User, config.Passwd, config.Host, config.Port, config.DbName, config.Sslmode),
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

// table miner_sector_infos
type MinerSectorInfos struct {
	MinerId               string `gorm:"column:miner_id"`
	SectorId              int    `gorm:"column:sector_id"`
	StateRoot             string `gorm:"column:state_root"`
	SealedCid             string `gorm:"column:sealed_cid"`
	ActivationEpoch       int32  `gorm:"column:activation_epoch"`
	ExpirationEpoch       int32  `gorm:"column:expiration_epoch"`
	DealWeight            int    `gorm:"column:deal_weight"`
	VerifiedDealWeight    int    `gorm:"column:verified_deal_weight"`
	InitialPledge         int64  `gorm:"column:initial_pledge"`
	ExpectedDayReward     int64  `gorm:"column:expected_day_reward"`
	ExpectedStoragePledge int64  `gorm:"column:expected_storage_pledge"`
	Height                int32  `gorm:"column:height"`
}

// table  miner_pre_commit_info
type MinerPreCommitInfos struct {
	MinerId                string `gorm:"column:miner_id"`
	SectorId               int    `gorm:"column:sector_id"`
	StateRoot              string `gorm:"column:state_root"`
	SealedCid              string `gorm:"column:sealed_cid"`
	SealRandEpoch          int32  `gorm:"column:seal_rand_epoch"`
	ExpirationEpoch        int32  `gorm:"column:expiration_epoch"`
	PreCommitDeposit       int64  `gorm:"column:pre_commit_deposit"`
	PreCommitEpoch         int64  `gorm:"column:pre_commit_epoch"`
	DealWeight             int    `gorm:"column:deal_weight"`
	VerifiedDealWeight     int32  `gorm:"column:verified_deal_weight"`
	IsReplaceCapacity      bool   `gorm:"column:is_replace_capacity"`
	ReplaceSectorDeadline  int32  `gorm:"column:replace_sector_deadline"`
	ReplaceSectorPartition int32  `gorm:"column:replace_sector_partition"`
	ReplaceSectorNumber    int32  `gorm:"column:replace_sector_number"`
	Height                 int64  `gorm:"column:height"`
}

// table derived_gas_outputs
type DerivedGasOutputs struct {
	Cid                string `gorm:"column:cid"`
	From               string `gorm:"column:from"`
	To                 string `gorm:"column:to"`
	Value              int64  `gorm:"column:value"`
	GasFeeCap          int64  `gorm:"column:gas_fee_cap"`
	GasLimit           int64  `gorm:"column:gas_limit"`
	SizeBytes          int    `gorm:"column:size_bytes"`
	Nonce              int    `gorm:"column:nonce"`
	Method             int    `gorm:"column:method"`
	StateRoot          string `gorm:"column:state_root"`
	ExitCode           int    `gorm:"column:exit_code"`
	GasUsed            int64  `gorm:"column:gas_used"`
	ParentBaseFee      int64  `gorm:"column:parent_base_fee"`
	BaseFeeBurn        int64  `gorm:"column:base_fee_burn"`
	OverEstimationBurn int64  `gorm:"column:over_estimation_burn"`
	MinerPenalty       int    `gorm:"column:miner_penalty"`
	MinerTip           int64  `gorm:"column:miner_tip"`
	Refund             int64  `gorm:"column:refund"`
	GasRefund          int64  `gorm:"column:gas_refund"`
	GasBurned          int64  `gorm:"column:gas_burned"`
	Height             int    `gorm:"column:height"`
	ActorName          string `gorm:"column:actor_name"`
}

func (cli *PostgresCli) QueryMinerPreCommitInfo(minerId string) (*MinerPreCommitInfos, error) {
	var info MinerPreCommitInfos
	var count int
	cli.db.Where("miner_id = ?", minerId).Find(&info).Count(&count)
	if count == 0 {
		return nil, xerrors.Errorf("cannot find any value")
	}

	return &info, nil
}

func (cli *PostgresCli) InsertMinerPreCommitInfo(info MinerPreCommitInfos) error {
	couldBeUpdated := false

	oldInfo, err := cli.QueryMinerPreCommitInfo(info.MinerId)
	if err == nil && oldInfo != nil {
		if !strings.Contains(oldInfo.MinerId, info.MinerId) {
			oldInfo.MinerId = fmt.Sprintf("%v,%v", oldInfo.MinerId, info.MinerId)
			couldBeUpdated = true
		}
	}

	var updateInfo *MinerPreCommitInfos = nil

	if couldBeUpdated {
		updateInfo = oldInfo
	}

	if updateInfo == nil {
		return xerrors.Errorf("invalid operation without maintaining mode")
	}

	rc := cli.db.Create(updateInfo)
	return rc.Error
}
