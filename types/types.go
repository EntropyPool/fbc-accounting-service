package types

// miner info
type MinerInfos struct {
	Cid                   string `gorm:"column:cid"`
	MinerId               string `gorm:"column:miner_id"`
	PreCommitDeposit      int64  `gorm:"column:pre_commit_deposit"`
	Value                 int64  `gorm:"column:value"`
	BaseFeeBurn           int64  `gorm:"column:base_fee_burn"`
	OverEstimationBurn    int64  `gorm:"column:over_estimation_burn"`
	MinerPenalty          int    `gorm:"column:miner_penalty"`
	MinerTip              int64  `gorm:"column:miner_tip"`
	Refund                int64  `gorm:"column:refund"`
	GasRefund             int64  `gorm:"column:gas_refund"`
	GasBurned             int64  `gorm:"column:gas_burned"`
	InitialPledge         int64  `gorm:"column:initial_pledge"`
	ExpectedStoragePledge int64  `gorm:"column:expected_storage_pledge"`
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
