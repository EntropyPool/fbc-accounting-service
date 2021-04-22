package types

const (
	/****************** API start *********************/
	// register etcd service
	GetRegisterEtcdAPI = "/api/v0/service/register"
	// miner pledge
	GetMinerPledgeAPI = "/api/v0/miner/pledge"
	// miner dailyReward
	GetMinerDailyRewardAPI = "/api/v0/miner/daily/reward"
	// miner info
	GetMinerInfoAPI = "/api/v0/miner/info"
	//  account info include miner worker normal
	GetAccountInfoAPI = "/api/v0/account/info"
	/****************** API end *********************/
	/****************** config start *********************/
	AccountingDomain = "accounting.npool.top"

	RegisterDomain = "etcd-register.npool.top"
	RegisterPort   = "7101"
	ServerIp       = "106.74.7.5"
	UserName       = "entropytest"
	Password       = "12345679"
	/****************** config end *********************/
)
