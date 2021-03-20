package types

// miner info
type MinerInfos struct {
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
}
