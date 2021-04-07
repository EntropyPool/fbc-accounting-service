package types

// miner info
type MinerInfos struct {
	Cid                   string `gorm:"column:cid"`
	MinerId               string `gorm:"column:miner_id"`
	PreCommitDeposit      string `gorm:"column:pre_commit_deposit"`
	Value                 string `gorm:"column:value"`
	BaseFeeBurn           string `gorm:"column:base_fee_burn"`
	OverEstimationBurn    string `gorm:"column:over_estimation_burn"`
	MinerPenalty          int    `gorm:"column:miner_penalty"`
	MinerTip              string `gorm:"column:miner_tip"`
	Refund                string `gorm:"column:refund"`
	GasRefund             string `gorm:"column:gas_refund"`
	GasBurned             string `gorm:"column:gas_burned"`
	Method                int    `gorm:"column:method"`
	InitialPledge         string `gorm:"column:initial_pledge"`
	ExpectedStoragePledge string `gorm:"column:expected_storage_pledge"`
	SectorId              int    `gorm:"column:sector_id"`

	//Cid string `gorm:"column:cid"`
	//From               string `gorm:"column:from"`
	//To                 string `gorm:"column:to"`
	//Value              int64  `gorm:"column:value"`
	//GasFeeCap          int64  `gorm:"column:gas_fee_cap"`
	//GasLimit           int64  `gorm:"column:gas_limit"`
	//SizeBytes          int    `gorm:"column:size_bytes"`
	//Nonce              int    `gorm:"column:nonce"`
	//Method             int    `gorm:"column:method"`
	//StateRoot          string `gorm:"column:state_root"`
	//ExitCode           int    `gorm:"column:exit_code"`
	//GasUsed            int64  `gorm:"column:gas_used"`
	//ParentBaseFee      int64  `gorm:"column:parent_base_fee"`
	//BaseFeeBurn        int64  `gorm:"column:base_fee_burn"`
	//OverEstimationBurn int64  `gorm:"column:over_estimation_burn"`
	//MinerPenalty       int    `gorm:"column:miner_penalty"`
	//MinerTip           int64  `gorm:"column:miner_tip"`
	//Refund             int64  `gorm:"column:refund"`
	//GasRefund          int64  `gorm:"column:gas_refund"`
	//GasBurned          int64  `gorm:"column:gas_burned"`
	//Height             int    `gorm:"column:height"`
	//ActorName          string `gorm:"column:actor_name"`
	//
	//MinerId               string `gorm:"column:miner_id"`
	//SectorId              int    `gorm:"column:sector_id"`
	//SealedCid             string `gorm:"column:sealed_cid"`
	//ActivationEpoch       int32  `gorm:"column:activation_epoch"`
	//ExpirationEpoch       int32  `gorm:"column:expiration_epoch"`
	//DealWeight            int    `gorm:"column:deal_weight"`
	//VerifiedDealWeight    int    `gorm:"column:verified_deal_weight"`
	//InitialPledge         int64  `gorm:"column:initial_pledge"`
	//ExpectedDayReward     int64  `gorm:"column:expected_day_reward"`
	//ExpectedStoragePledge int64  `gorm:"column:expected_storage_pledge"`
	//
	//SealRandEpoch          int32 `gorm:"column:seal_rand_epoch"`
	//PreCommitDeposit       int64 `gorm:"column:pre_commit_deposit"`
	//PreCommitEpoch         int64 `gorm:"column:pre_commit_epoch"`
	//IsReplaceCapacity      bool  `gorm:"column:is_replace_capacity"`
	//ReplaceSectorDeadline  int32 `gorm:"column:replace_sector_deadline"`
	//ReplaceSectorPartition int32 `gorm:"column:replace_sector_partition"`
	//ReplaceSectorNumber    int32 `gorm:"column:replace_sector_number"`
}

// normal account
type AccountInfo struct {
	Id          string `gorm:"column:Id"` // normalAddress workerId(workerAddress) minerId(minerAddress)
	Balance     string `gorm:"column:Balance"`
	BlockHeight int64  `gorm:"column:BlockHeight"`
	Fee         string `gorm:"column:Fee"` // 费用
	MinerTip    string `gorm:"column:MinerTip"`
	SendIn      string `gorm:"column:Send"` // 转账（入)
	SendOut     string `gorm:"column:Send"` // 转账（出)
	Send        string `gorm:"column:Send"` // 转账
}

// worker account
type WorkerInfo struct {
	AccountInfo
	PreCommitSectors   string `gorm:"column:preCommitSectors"`
	ProveCommitSectors string `gorm:"column:proveCommitSectors"`
}

// miner account
type MinerInfo struct {
	AccountInfo
	PunishFee                     string `gorm:"column:punishFee"` // 惩罚
	PreCommitDeposits             string `gorm:"column:preCommitDeposits"`
	PreCommitSectors              string `gorm:"column:preCommitSectors"`
	ProveCommitSectors            string `gorm:"column:proveCommitSectors"`
	BlockReward                   string `gorm:"column:blockReward"`
	TAG                           string `gorm:"column:tag"`
	MinerAvailableBalance         string `gorm:"column:minerAvailableBalance"`
	LockedFunds                   string `gorm:"column:lockedFunds"`
	InitialPledge                 string `gorm:"column:initialPledge"`
	BlockRewardToAvailableBalance string `gorm:"column:blockRewardToAvailableBalance"`
	BlockRewardToLockedFunds      string `gorm:"column:blockRewardToLockedFunds"`
	SubLockFunds                  string `gorm:"column:SubLockFunds"`
}

type DailyMinerInfoAvailable struct {
	TodayBlockRewards      string `gorm:"column:TodayBlockRewards"`     // 当天的奖励
	Today25PercentRewards  string `gorm:"column:Today25PercentRewards"` // 当天奖励25%的释放
	Today180PercentRewards string `gorm:"column:Today25PercentRewards"` // 当天累计1/180释放
	TotalTodayRewards      string `gorm:"column:Total25PercentRewards"` // 累计当天总释放
	PunishFee              string `gorm:"column:PunishFee"`             // 惩罚
	MinerPower             string `gorm:"column:MinerPower"`            // 矿工算力
}
