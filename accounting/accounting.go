package accounting

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	log "github.com/EntropyPool/entropy-logger"
	fbcpostgres "github.com/EntropyPool/fbc-accounting-service/postgres"
	filrpc "github.com/EntropyPool/fbc-accounting-service/rpc"
	types "github.com/EntropyPool/fbc-accounting-service/types"
	utils "github.com/EntropyPool/fbc-accounting-service/utils"
	httpdaemon "github.com/NpoolRD/http-daemon"
	"github.com/tealeg/xlsx"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	_ "strings"
	"sync/atomic"
	"time"
	_ "time"
)

type AccountingConfig struct {
	PostgresConfig fbcpostgres.PostgresConfig `json:"postgres"`
	Port           int
}

type AccountingServer struct {
	config         AccountingConfig
	PostgresClient *fbcpostgres.PostgresCli
}

func NewAccountingServer(configFile string) *AccountingServer {

	buf, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Errorf(log.Fields{}, "cannot read file %v: %v", configFile, err)
		return nil
	}

	config := AccountingConfig{}
	err = json.Unmarshal(buf, &config)
	if err != nil {
		log.Errorf(log.Fields{}, "cannot parse file %v: %v", configFile, err)
		return nil
	}

	log.Infof(log.Fields{}, "create mysql cli: %v", config.PostgresConfig)
	postgreCli := fbcpostgres.NewPostgresCli(config.PostgresConfig)
	if postgreCli == nil {
		log.Errorf(log.Fields{}, "cannot create postgresql client %v: %v", config.PostgresConfig, err)
		return nil
	}

	server := &AccountingServer{
		config:         config,
		PostgresClient: postgreCli,
	}

	log.Infof(log.Fields{}, "successful to create devops server")

	return server
}

func (s *AccountingServer) Run() error {
	httpdaemon.RegisterRouter(httpdaemon.HttpRouter{
		Location: types.GetMinerPledgeAPI,
		Method:   "GET",
		Handler: func(w http.ResponseWriter, req *http.Request) (interface{}, string, int) {
			return s.GetMinerPledgeRequest(w, req)
		},
	})
	httpdaemon.RegisterRouter(httpdaemon.HttpRouter{
		Location: types.GetMinerDailyRewardAPI,
		Method:   "GET",
		Handler: func(w http.ResponseWriter, req *http.Request) (interface{}, string, int) {
			return s.GetMinerDailyRewardRequest(w, req)
		},
	})
	httpdaemon.RegisterRouter(httpdaemon.HttpRouter{
		Location: types.GetMinerInfoAPI,
		Method:   "GET",
		Handler: func(w http.ResponseWriter, req *http.Request) (interface{}, string, int) {
			return s.GetMinerInfoRequest(w, req)
		},
	})

	httpdaemon.RegisterRouter(httpdaemon.HttpRouter{
		Location: types.GetAccountInfoAPI,
		Method:   "GET",
		Handler: func(w http.ResponseWriter, req *http.Request) (interface{}, string, int) {
			return s.GetAccountInfoRequest(w, req)
		},
	})

	log.Infof(log.Fields{}, "start http daemon at %v", s.config.Port)
	httpdaemon.Run(s.config.Port)
	return nil
}

// for digfil to find miner InitialPledge
func (s *AccountingServer) GetMinerPledgeRequest(w http.ResponseWriter, req *http.Request) (interface{}, string, int) {

	query := req.URL.Query()
	account := string(query["account"][0])
	//minerPreCommitInfo, _ := s.PostgresClient.QueryMinerPreCommitInfo(minerId)
	result := filrpc.ChainHead()
	var totalInitialPledge interface{}
	var initialPledge interface{}
	var preCommitDeposits interface{}
	var resultMap map[string]interface{}
	if err := json.Unmarshal([]byte(result), &resultMap); err == nil {
		cid := resultMap["result"].(map[string]interface{})["Cids"].([]interface{})[0].(map[string]interface{})["/"]
		cidStr := cid.(string)
		stateReadStateResult := filrpc.StateReadState(account, cidStr)
		var stateReadStateMap map[string]interface{}
		if err := json.Unmarshal([]byte(stateReadStateResult), &stateReadStateMap); err == nil {
			initialPledge = stateReadStateMap["result"].(map[string]interface{})["State"].(map[string]interface{})["InitialPledge"]
			preCommitDeposits = stateReadStateMap["result"].(map[string]interface{})["State"].(map[string]interface{})["PreCommitDeposits"]
		} else {
			return err, "error", 1
		}
	}
	totalInitialPledge = utils.BigIntAddStr(initialPledge.(string), preCommitDeposits.(string))

	return totalInitialPledge, "success", 0
}

//  to find miner dailyReward
func (s *AccountingServer) GetMinerDailyRewardRequest(writer http.ResponseWriter, request *http.Request) (interface{}, string, int) {

	query := request.URL.Query()
	// 账户
	account := string(query["account"][0])
	// 初始高度0 出块时间 1598306400 后面每一个加30s 北京时间戳
	startTime := string(query["startTime"][0])
	startTimestamp, _ := strconv.ParseInt(startTime, 10, 32)
	// beijing time
	startHeightInt64 := atomic.AddInt64(&startTimestamp, -1598306400)
	realStartHeight := startHeightInt64 / 30
	endTime := string(query["endTime"][0])
	endTimestamp, _ := strconv.ParseInt(endTime, 10, 32)
	endHeightInt64 := atomic.AddInt64(&endTimestamp, -1598306400)
	realEndHeight := endHeightInt64 / 30
	fmt.Printf("account: %s,realStartHeight: %s,realEndHeight: %s", account, realStartHeight, realEndHeight)
	var DailyRewardInfos types.DailyMinerInfoAvailable
	var infos []types.MinerInfo
	TodayBlockRewards := "0"
	Today25PercentRewards := "0"
	TotalTodayRewards := "0"
	var MinerPower interface{}
	stateMinerPowerResult := filrpc.StateMinerPower(account, "", false)
	var stateMinerPowerMap map[string]interface{}
	if err := json.Unmarshal([]byte(stateMinerPowerResult), &stateMinerPowerMap); err == nil {
		MinerPower = stateMinerPowerMap["result"].(map[string]interface{})["MinerPower"].(map[string]interface{})["RawBytePower"]
	}

	totalPunishFees := "0"
	var minerAvailableBalance interface{}
	var preCommitDeposits interface{}
	var lockedFunds interface{}
	var initialPledge interface{}
	var minerBalance interface{}
	totalSubLockFunds := "0"

	for i := realStartHeight; i <= realEndHeight; i++ {
		// 统计fee minertip totalvalue
		// table derived_gas_outputs
		iStr := strconv.FormatInt(i, 10)
		var resultMap map[string]interface{}
		result := filrpc.GetMinerInfoByMinerIdAndHeight(account, iStr)
		if err := json.Unmarshal([]byte(result), &resultMap); err == nil {
			cid := resultMap["result"].(map[string]interface{})["Cids"].([]interface{})[0].(map[string]interface{})["/"]
			reallyBlockNo := resultMap["result"].(map[string]interface{})["Height"]
			int64ReallyBlockNo := int64(reallyBlockNo.(float64))
			var flagBlockIsNull = false
			if int64ReallyBlockNo == i-1 {
				flagBlockIsNull = true
			}
			cidStr := cid.(string)
			var stateGetActorMinerMap map[string]interface{}
			stateGetActorMinerResult := filrpc.StateGetActor(account, cidStr)
			if err := json.Unmarshal([]byte(stateGetActorMinerResult), &stateGetActorMinerMap); err == nil {
				minerBalance = stateGetActorMinerMap["result"].(map[string]interface{})["Balance"]
				minerAvailableBalanceResult := filrpc.StateMinerAvailableBalance(account, cidStr)
				var minerAvailableBalanceMap map[string]interface{}
				if err := json.Unmarshal([]byte(minerAvailableBalanceResult), &minerAvailableBalanceMap); err == nil {
					minerAvailableBalance = minerAvailableBalanceMap["result"]

					stateReadStateResult := filrpc.StateReadState(account, cidStr)
					var stateReadStateMap map[string]interface{}
					if err := json.Unmarshal([]byte(stateReadStateResult), &stateReadStateMap); err == nil {
						preCommitDeposits = stateReadStateMap["result"].(map[string]interface{})["State"].(map[string]interface{})["PreCommitDeposits"]
						lockedFunds = stateReadStateMap["result"].(map[string]interface{})["State"].(map[string]interface{})["LockedFunds"]
						initialPledge = stateReadStateMap["result"].(map[string]interface{})["State"].(map[string]interface{})["InitialPledge"]
					}
				}
			}
			var info = types.MinerInfo{}
			info.Id = account
			info.BlockHeight = i
			totalBurnFee := "0"
			totalMinerTip := "0"
			totalSend := "0"
			totalSendIn := "0"
			totalSendOut := "0"
			totalPreCommitSectors := "0"
			totalProveCommitSectors := "0"
			totalWithdrawBalance := "0"

			derivedGasOutputs, _ := s.PostgresClient.QueryDerivedGasOutputs(account, i)
			if len(derivedGasOutputs) > 0 {
				for j := 0; j < len(derivedGasOutputs); j++ {
					// 出账才有手续费
					if strings.EqualFold(derivedGasOutputs[j].From, account) {
						totalBurnFee = utils.BigIntAddStr(totalBurnFee, derivedGasOutputs[j].BaseFeeBurn)
						totalBurnFee = utils.BigIntAddStr(totalBurnFee, derivedGasOutputs[j].OverEstimationBurn)
						totalMinerTip = utils.BigIntAddStr(totalMinerTip, derivedGasOutputs[j].MinerTip)
					}
					if strings.EqualFold(derivedGasOutputs[j].From, account) && derivedGasOutputs[j].Method == 0 { // sub
						totalSend = utils.BigIntReduceStr(totalSend, derivedGasOutputs[j].Value)
						totalSendOut = utils.BigIntReduceStr(totalSendOut, derivedGasOutputs[j].Value)
					} else if strings.EqualFold(derivedGasOutputs[j].To, account) && derivedGasOutputs[j].Method == 0 { // add
						totalSend = utils.BigIntAddStr(totalSend, derivedGasOutputs[j].Value)
						totalSendIn = utils.BigIntAddStr(totalSendIn, derivedGasOutputs[j].Value)
					}
					if derivedGasOutputs[j].Method == 6 {
						totalPreCommitSectors = utils.BigIntAddStr(totalPreCommitSectors, derivedGasOutputs[j].Value)
					}
					if derivedGasOutputs[j].Method == 7 {
						totalProveCommitSectors = utils.BigIntAddStr(totalProveCommitSectors, derivedGasOutputs[j].Value)
					}
					// miner withdrawBalance
					if derivedGasOutputs[j].Method == 16 {
						stateReplayResult := filrpc.StateReplay(cidStr, derivedGasOutputs[j].Cid)
						if stateReplayResult != "" {
							var stateReadStateMap map[string]interface{}
							if err := json.Unmarshal([]byte(stateReplayResult), &stateReadStateMap); err == nil {
								withdrawBalance := stateReadStateMap["result"].(map[string]interface{})["ExecutionTrace"].(map[string]interface{})["Subcalls"].([]interface{})[0].(map[string]interface{})["Msg"].(map[string]interface{})["Value"]
								totalWithdrawBalance = utils.BigIntAddStr(totalWithdrawBalance, withdrawBalance.(string))
							}
						}
					}
				}
			}

			info.Fee = totalBurnFee
			info.MinerTip = totalMinerTip
			info.Send = totalSend // sub + add
			info.SendIn = totalSendIn
			info.SendOut = totalSendOut
			info.PreCommitSectors = totalPreCommitSectors
			info.ProveCommitSectors = totalProveCommitSectors

			info.WithdrawBalance = totalWithdrawBalance
			info.BlockReward = "0"
			info.FlagBlockIsNull = flagBlockIsNull
			info.InitialPledge = initialPledge.(string)
			info.PreCommitDeposits = preCommitDeposits.(string)
			info.MinerAvailableBalance = minerAvailableBalance.(string)
			info.LockedFunds = lockedFunds.(string)
			info.Balance = minerBalance.(string)
			infos = append(infos, info)
			var k = len(infos)
			totalPreCommitSectorss := "0"
			if k >= 2 {
				// 两个高度的余额差
				subBalance := utils.BigIntReduceStr(infos[k-1].Balance, infos[k-2].Balance)
				// 上一个高度的pre + pro
				totalPreCommitSectorss = utils.BigIntAddStr(infos[k-2].PreCommitSectors, infos[k-2].ProveCommitSectors)
				// 区块奖励预判 = 两个高度的余额差 - （上一个高度的pre + pro）
				infos[k-2].BlockReward = utils.BigIntReduceStr(subBalance, totalPreCommitSectorss)
				blockReward, _ := strconv.ParseInt(infos[k-2].BlockReward, 10, 64)
				if blockReward > 0 {
					// 区块奖励实际 = 区块奖励预判 - 上一个高度的收支send
					infos[k-2].BlockReward = utils.BigIntReduceStr(infos[k-2].BlockReward, infos[k-2].Send)
					infos[k-2].BlockReward = utils.BigIntAddStr(infos[k-2].BlockReward, infos[k-2].WithdrawBalance)
					// 特殊情况 : 下一个高度是空块 作差也会大于0 因为空块余额不变化，而且没将上一个高度的pre 和pro 加进来
					// 特殊情况： 下一个高度是空块，而此时出块了，此时计算到出块的奖励应该给到 infos[k-3].blockReward
					k3totalPreCommitSectors := utils.BigIntAddStr(infos[k-3].PreCommitSectors, infos[k-3].ProveCommitSectors)
					var p = utils.BigIntReduceStr(infos[k-2].BlockReward, k3totalPreCommitSectors)
					//// TODO block is null
					//if strings.EqualFold(p, "0") && !strings.EqualFold(k3totalPreCommitSectors, "0") {
					//	infos[k-2].BlockReward = "0"
					//	infos[k-2].TAG = "block is null"
					//}
					// block is null
					if infos[k-2].FlagBlockIsNull && strings.EqualFold(p, "0") && !strings.EqualFold(k3totalPreCommitSectors, "0") {
						infos[k-3].BlockReward = infos[k-2].BlockReward //将值赋给上一层
						infos[k-2].BlockReward = "0"
						infos[k-2].TAG = "block is null"
					}
					// 累计每天的奖励
					TodayBlockRewards = utils.BigIntAddStr(TodayBlockRewards, infos[k-2].BlockReward)
					//fmt.Printf("--blockNo:" + strconv.FormatInt(i, 10) + "--BlockReward:" + infos[k-2].BlockReward + "\n")

				} else if blockReward < 0 {
					// maybe this block is null or other unknown bug
					infos[k-2].BlockReward = "0"
				}

				// TODO 线性释放金额
				// 求某个高度 线性释放的金额  = （下一高度的可用余额-上一个高度可用余额）+ withdrawBalance -(上一个高度的(pre + pro + InitialPledge + PreCommitDeposits + send) - 下一个高度的（PreCommitDeposits+InitialPledge）)
				// 连续三个高度中间出现空块的情况
				if k >= 3 {
					if !infos[k-1].FlagBlockIsNull && infos[k-2].FlagBlockIsNull && !infos[k-3].FlagBlockIsNull {
						add1 := utils.BigIntAddStr(infos[k-1].PreCommitDeposits, infos[k-1].InitialPledge)
						add1 = utils.BigIntAddStr(add1, infos[k-1].MinerAvailableBalance)
						add3 := utils.BigIntAddStr(infos[k-3].PreCommitDeposits, infos[k-3].InitialPledge)
						add3 = utils.BigIntAddStr(add3, infos[k-3].Send)
						// k3 totalPreCommitSector
						totalPreCommitSector := utils.BigIntAddStr(infos[k-3].PreCommitSectors, infos[k-3].ProveCommitSectors)
						add3 = utils.BigIntAddStr(totalPreCommitSector, add3)
						add3 = utils.BigIntAddStr(infos[k-3].MinerAvailableBalance, add3)
						if !strings.EqualFold(infos[k-3].WithdrawBalance, "0") { // miner withdrawBalance
							add3 = utils.BigIntReduceStr(add3, infos[k-3].WithdrawBalance)
						}
						subBlockRewardAvalible := utils.BigIntReduceStr(add1, add3)
						TotalTodayRewards = utils.BigIntAddStr(subBlockRewardAvalible, TotalTodayRewards)
						// 连续三个高度不出现空块的情况
					} else if !infos[k-1].FlagBlockIsNull && !infos[k-2].FlagBlockIsNull && !infos[k-3].FlagBlockIsNull {
						if k == 3 {
							add2 := utils.BigIntAddStr(infos[k-2].PreCommitDeposits, infos[k-2].InitialPledge)
							add2 = utils.BigIntAddStr(add2, infos[k-2].MinerAvailableBalance)
							add3 := utils.BigIntAddStr(infos[k-3].PreCommitDeposits, infos[k-3].InitialPledge)
							add3 = utils.BigIntAddStr(add3, infos[k-3].Send)
							totalPreCommitSector := utils.BigIntAddStr(infos[k-3].PreCommitSectors, infos[k-3].ProveCommitSectors)
							add3 = utils.BigIntAddStr(totalPreCommitSector, add3)
							add3 = utils.BigIntAddStr(infos[k-3].MinerAvailableBalance, add3)
							if !strings.EqualFold(infos[k-3].WithdrawBalance, "0") { // miner withdrawBalance
								add3 = utils.BigIntReduceStr(add3, infos[k-3].WithdrawBalance)
							}
							subBlockRewardAvalible := utils.BigIntReduceStr(add2, add3)
							TotalTodayRewards = utils.BigIntAddStr(subBlockRewardAvalible, TotalTodayRewards)
						} else {
							add1 := utils.BigIntAddStr(infos[k-1].PreCommitDeposits, infos[k-1].InitialPledge)
							add1 = utils.BigIntAddStr(add1, infos[k-1].MinerAvailableBalance)
							add2 := utils.BigIntAddStr(infos[k-2].PreCommitDeposits, infos[k-2].InitialPledge)
							add2 = utils.BigIntAddStr(add2, infos[k-2].Send)
							add2 = utils.BigIntAddStr(totalPreCommitSectorss, add2)
							add2 = utils.BigIntAddStr(infos[k-2].MinerAvailableBalance, add2)
							if !strings.EqualFold(infos[k-2].WithdrawBalance, "0") { // miner withdrawBalance
								add2 = utils.BigIntReduceStr(add2, infos[k-2].WithdrawBalance)
							}
							subBlockRewardAvalible := utils.BigIntReduceStr(add1, add2)
							TotalTodayRewards = utils.BigIntAddStr(subBlockRewardAvalible, TotalTodayRewards)
						}
					}
				}

				// TODO lockFunds
				subLockFunds := "0"
				if k >= 3 {
					// 中间出现空块
					if !infos[k-1].FlagBlockIsNull && infos[k-2].FlagBlockIsNull && !infos[k-3].FlagBlockIsNull {
						subLockFunds = utils.BigIntReduceStr(infos[k-1].LockedFunds, infos[k-3].LockedFunds)
						if !strings.Contains(subLockFunds, "-") {
							totalSubLockFunds = utils.BigIntAddStr(subLockFunds, totalSubLockFunds)
						}
						// 连续不出现空块
					} else if !infos[k-1].FlagBlockIsNull && !infos[k-2].FlagBlockIsNull && !infos[k-3].FlagBlockIsNull {
						if k == 3 {
							subLockFunds = utils.BigIntReduceStr(infos[k-2].LockedFunds, infos[k-3].LockedFunds)
							if !strings.Contains(subLockFunds, "-") {
								totalSubLockFunds = utils.BigIntAddStr(subLockFunds, totalSubLockFunds)
							}
							subLockFunds = utils.BigIntReduceStr(infos[k-1].LockedFunds, infos[k-2].LockedFunds)
							if !strings.Contains(subLockFunds, "-") {
								totalSubLockFunds = utils.BigIntAddStr(subLockFunds, totalSubLockFunds)
							}
						} else {
							subLockFunds = utils.BigIntReduceStr(infos[k-1].LockedFunds, infos[k-2].LockedFunds)
							if !strings.Contains(subLockFunds, "-") {
								totalSubLockFunds = utils.BigIntAddStr(subLockFunds, totalSubLockFunds)
							}
						}

					}

				}

				// TODO 惩罚
				// 惩罚 = 上一个高度的 （pre+ pro + blockreward + 收支send）- 两个高度的余额差
				addBalance := utils.BigIntAddStr(totalPreCommitSectorss, infos[k-2].BlockReward)
				addBalance = utils.BigIntAddStr(addBalance, infos[k-2].Send)

				if !strings.EqualFold(addBalance, subBalance) {
					if !strings.EqualFold(infos[k-2].TAG, "block is null") {
						infos[k-2].TAG = "惩罚(销毁)"
						infos[k-2].PunishFee = utils.BigIntReduceStr(addBalance, subBalance)
						if !strings.EqualFold(infos[k-2].WithdrawBalance, "0") { // 去掉提现的部分
							//fmt.Printf("--blockNo:" + strconv.FormatInt(i, 10) + "---totalWithdrawBalance-------:" + infos[k-2].WithdrawBalance + "\n")
							infos[k-2].PunishFee = utils.BigIntReduceStr(infos[k-2].PunishFee, infos[k-2].WithdrawBalance)
						}
						totalPunishFees = utils.BigIntAddStr(totalPunishFees, infos[k-2].PunishFee)
						//fmt.Printf("--blockNo:" + strconv.FormatInt(i, 10) + "---burn-PunishFee-------:" + infos[k-2].PunishFee + "---totalPunishFees:" + totalPunishFees + "\n")
					} else {
						// 将空块 误认为是惩罚的 减掉
						totalPunishFees = utils.BigIntReduceStr(totalPunishFees, infos[k-3].PunishFee)
						//fmt.Printf("--blockNo:" + strconv.FormatInt(i, 10) + "---return---PunishFee----:" + infos[k-3].PunishFee + "---totalPunishFees:" + totalPunishFees + "\n")
						infos[k-3].PunishFee = "0"
						infos[k-3].TAG = ""
					}
					// 或者 subBalance=addBalance 且 lockFunds 不相等
				} else if strings.EqualFold(infos[k-1].Balance, infos[k-2].Balance) && !strings.EqualFold(subLockFunds, "0") {
					// maybe 惩罚
					//if strings.Contains(subLockFunds, "-") { //小于0
					//	totalPunishFees = utils.BigIntAddStr(totalPunishFees, subLockFunds)
					//} else {
					//	totalPunishFees = utils.BigIntReduceStr(totalPunishFees, subLockFunds)
					//}
					fmt.Printf("--blockNo:" + strconv.FormatInt(i, 10) + "---lockFunds不相等 :" + subLockFunds + "\n")
				}

			}

		}
	}
	// SubLockFunds 含有 PunishFee
	Total := utils.BigIntAddStr(TotalTodayRewards, totalSubLockFunds)
	// 除了正常出块，剩下的为线性释放
	Today180PercentRewards := utils.BigIntReduceStr(Total, TodayBlockRewards)

	DailyRewardInfos.TodayBlockRewards = TodayBlockRewards
	DailyRewardInfos.PunishFee = totalPunishFees
	DailyRewardInfos.TotalTodayRewards = TotalTodayRewards
	percentReward := utils.BigIntMulStr(TodayBlockRewards, "25")
	Today25PercentRewards = utils.BigIntDivStr(percentReward, "100")
	// 25% pencent rewards one day
	DailyRewardInfos.Today25PercentRewards = Today25PercentRewards
	DailyRewardInfos.Today180PercentRewards = Today180PercentRewards
	DailyRewardInfos.MinerPower = MinerPower.(string)
	fmt.Println("result: \n TodayBlockRewards:", TodayBlockRewards, "---", "totalSubLockFunds:", totalSubLockFunds, "---", "TotalTodayRewards:", TotalTodayRewards, "---",
		"PunishFee:", totalPunishFees, "---", "Today180PercentRewards:", Today180PercentRewards)
	return DailyRewardInfos, "", 0
}

// get MinerInfo detail
func (s *AccountingServer) GetMinerInfoRequest(w http.ResponseWriter, req *http.Request) (interface{}, string, int) {
	minerInfo := types.MinerInfos{}

	query := req.URL.Query()
	minerId := string(query["minerId"][0])
	//beginHeight := string(query["beginHeight"][0])
	//begin, _ := strconv.ParseInt(beginHeight, 10, 32)
	//endHeight := string(query["endHeight"][0])
	//end, _ := strconv.ParseInt(endHeight, 10, 32)
	// 初始高度0 出块时间 1598306400 后面每一个加30s
	startTime := string(query["startTime"][0])
	startTimestamp, _ := strconv.ParseInt(startTime, 10, 32)
	//startHeightInt64 := atomic.AddInt64(&startTimestamp, -1598306400)
	// beijing time
	startHeightInt64 := atomic.AddInt64(&startTimestamp, -1598306400)
	realStartHeight := startHeightInt64 / 30
	fmt.Printf("realStartHeight%s", realStartHeight)

	endTime := string(query["endTime"][0])
	endTimestamp, _ := strconv.ParseInt(endTime, 10, 32)
	endHeightInt64 := atomic.AddInt64(&endTimestamp, -1598306400)
	realEndHeight := endHeightInt64 / 30
	fmt.Printf("realEndHeight%s", realEndHeight)

	to := minerId

	var file *xlsx.File
	var sheet *xlsx.Sheet
	var row *xlsx.Row
	var cell *xlsx.Cell
	file = xlsx.NewFile()
	sheet, _ = file.AddSheet("Sheet1")
	row = sheet.AddRow()
	//	w.Write([]string{"MinerId", "height", "PreCommitDeposit", "Value", "BaseFeeBurn", "OverEstimationBurn",
	//		"MinerPenalty", "MinerTip", "Refund", "GasRefund", "InitialPledge", "ExpectedStoragePledge",
	//		"minerAvailableBalance", "preCommitDeposits", "lockedFunds", "initialPledge"})
	cell = row.AddCell()
	cell.Value = "Cid"
	cell = row.AddCell()
	cell.Value = "MinerId"
	cell = row.AddCell()
	cell.Value = "height"
	cell = row.AddCell()
	cell.Value = "Value"
	cell = row.AddCell()
	cell.Value = "BaseFeeBurn"
	cell = row.AddCell()
	cell.Value = "OverEstimationBurn"
	cell = row.AddCell()
	cell.Value = "MinerPenalty"
	cell = row.AddCell()
	cell.Value = "MinerTip"
	cell = row.AddCell()
	cell.Value = "Refund"
	cell = row.AddCell()
	cell.Value = "GasRefund"
	cell = row.AddCell()
	cell.Value = "Method"
	cell = row.AddCell()
	cell.Value = "InitialPledge"
	cell = row.AddCell()
	cell.Value = "ExpectedStoragePledge"
	cell = row.AddCell()
	cell.Value = "PreCommitDeposit"
	cell = row.AddCell()
	cell.Value = "SectorId"
	cell = row.AddCell()
	cell.Value = "minerAvailableBalance"
	cell = row.AddCell()
	cell.Value = "preCommitDeposits"
	cell = row.AddCell()
	cell.Value = "lockedFunds"
	cell = row.AddCell()
	cell.Value = "initialPledge"
	cell = row.AddCell()
	cell.Value = "workerBalance"
	cell = row.AddCell()
	cell.Value = "minerBalance"

	cell = row.AddCell()
	cell.Value = "totalCostFee"

	for i := realStartHeight; i < realEndHeight; i++ {

		iStr := strconv.FormatInt(i, 10)
		result := filrpc.GetMinerInfoByMinerIdAndHeight(minerId, iStr)
		var resultMap map[string]interface{}
		var minerAvailableBalance interface{}
		var preCommitDeposits interface{}
		var lockedFunds interface{}
		var initialPledge interface{}
		var balance interface{}
		var minerBalance interface{}
		cidStr := ""
		if err := json.Unmarshal([]byte(result), &resultMap); err == nil {
			cid := resultMap["result"].(map[string]interface{})["Cids"].([]interface{})[0].(map[string]interface{})["/"]
			cidStr = cid.(string)
			minerAvailableBalanceResult := filrpc.StateMinerAvailableBalance(minerId, cidStr)
			var minerAvailableBalanceMap map[string]interface{}
			if err := json.Unmarshal([]byte(minerAvailableBalanceResult), &minerAvailableBalanceMap); err == nil {
				minerAvailableBalance = minerAvailableBalanceMap["result"]

				stateReadStateResult := filrpc.StateReadState(minerId, cidStr)
				var stateReadStateMap map[string]interface{}
				if err := json.Unmarshal([]byte(stateReadStateResult), &stateReadStateMap); err == nil {
					preCommitDeposits = stateReadStateMap["result"].(map[string]interface{})["State"].(map[string]interface{})["PreCommitDeposits"]
					lockedFunds = stateReadStateMap["result"].(map[string]interface{})["State"].(map[string]interface{})["LockedFunds"]
					initialPledge = stateReadStateMap["result"].(map[string]interface{})["State"].(map[string]interface{})["InitialPledge"]
				}
				var stateGetActorMinerMap map[string]interface{}
				stateGetActorMinerResult := filrpc.StateGetActor(minerId, cidStr)
				if err := json.Unmarshal([]byte(stateGetActorMinerResult), &stateGetActorMinerMap); err == nil {
					minerBalance = stateGetActorMinerMap["result"].(map[string]interface{})["Balance"]
				}
				var stateMinerInfoMap map[string]interface{}
				stateMinerInfoResult := filrpc.StateMinerInfo(minerId, cidStr)
				if err := json.Unmarshal([]byte(stateMinerInfoResult), &stateMinerInfoMap); err == nil {
					workerId := stateMinerInfoMap["result"].(map[string]interface{})["Worker"]
					workerIdStr := workerId.(string)
					var stateGetActorMap map[string]interface{}
					stateGetActorResult := filrpc.StateGetActor(workerIdStr, cidStr)
					if err := json.Unmarshal([]byte(stateGetActorResult), &stateGetActorMap); err == nil {
						balance = stateGetActorMap["result"].(map[string]interface{})["Balance"]
					}
				}

			}

		} else {
			fmt.Println(err)
		}
		row = sheet.AddRow()

		// table derived_gas_outputs
		derivedGasOutputs, _ := s.PostgresClient.QueryDerivedGasOutputs(to, i)

		// table miner_sector_infos
		minerSectorInfos, _ := s.PostgresClient.QueryMinerSectorInfos(minerId, i)
		// table miner_pre_commit_infos
		if derivedGasOutputs != nil && minerSectorInfos != nil {
			var w = 0
			if len(derivedGasOutputs) > len(minerSectorInfos) {
				w = len(derivedGasOutputs)
			} else {
				w = len(minerSectorInfos)
			}
			var dg = len(derivedGasOutputs)
			var ms = len(minerSectorInfos)
			var totalCostFee string = "0"
			for j := 0; j <= w-1; j++ {

				if j >= 0 && dg-1 >= j {
					minerInfo.Value = derivedGasOutputs[j].Value
					minerInfo.BaseFeeBurn = derivedGasOutputs[j].BaseFeeBurn
					minerInfo.OverEstimationBurn = derivedGasOutputs[j].OverEstimationBurn
					minerInfo.MinerPenalty = derivedGasOutputs[j].MinerPenalty
					minerInfo.MinerTip = derivedGasOutputs[j].MinerTip
					minerInfo.Refund = derivedGasOutputs[j].Refund
					minerInfo.GasRefund = derivedGasOutputs[j].GasRefund
					minerInfo.Cid = derivedGasOutputs[j].Cid
					minerInfo.Method = derivedGasOutputs[j].Method
					//totalCostFee = totalCostFee + uint64(minerInfo.BaseFeeBurn) + uint64(minerInfo.OverEstimationBurn) + uint64(minerInfo.MinerTip)
					totalCostFee = utils.BigIntAddStr(totalCostFee, minerInfo.BaseFeeBurn)
					totalCostFee = utils.BigIntAddStr(totalCostFee, minerInfo.OverEstimationBurn)
					totalCostFee = utils.BigIntAddStr(totalCostFee, minerInfo.MinerTip)

					row = sheet.AddRow()
					cell = row.AddCell()
					cell.Value = minerInfo.Cid
					cell = row.AddCell()
					cell.Value = minerId
					cell = row.AddCell()
					cell.Value = strconv.FormatInt(i, 10)
					cell = row.AddCell()
					//cell.Value = strconv.FormatInt(minerInfo.Value, 10)
					cell.Value = minerInfo.Value
					cell = row.AddCell()
					cell.Value = minerInfo.BaseFeeBurn
					cell = row.AddCell()
					cell.Value = minerInfo.OverEstimationBurn
					cell = row.AddCell()
					cell.Value = strconv.Itoa(minerInfo.MinerPenalty)
					cell = row.AddCell()
					cell.Value = minerInfo.MinerTip
					cell = row.AddCell()
					cell.Value = minerInfo.Refund
					cell = row.AddCell()
					cell.Value = minerInfo.GasRefund
					cell = row.AddCell()
					cell.Value = strconv.Itoa(minerInfo.Method)
					if ms-1 >= j {
						minerInfo.InitialPledge = minerSectorInfos[j].InitialPledge
						minerInfo.ExpectedStoragePledge = minerSectorInfos[j].ExpectedStoragePledge
						// 存在数据库同步异常 没有记录的情况
						minerPreCommitInfo, _ := s.PostgresClient.QueryMinerPreCommitInfoAndSectorId(minerId, minerSectorInfos[j].SectorId)
						if minerPreCommitInfo != nil {
							minerInfo.PreCommitDeposit = minerPreCommitInfo.PreCommitDeposit
						} else {
							stateSectorPreCommitInfoResult := filrpc.StateSectorPreCommitInfo(minerId, strconv.Itoa(minerSectorInfos[j].SectorId), cidStr)
							var stateSectorPreCommitInfoMap map[string]interface{}
							if err := json.Unmarshal([]byte(stateSectorPreCommitInfoResult), &stateSectorPreCommitInfoMap); err == nil {
								minerInfo.PreCommitDeposit = stateSectorPreCommitInfoMap["result"].(map[string]interface{})["PreCommitDeposit"].(string)
							} else {
								minerInfo.PreCommitDeposit = "-1"
							}
						}
						minerInfo.SectorId = minerSectorInfos[j].SectorId
					} else {
						minerInfo.InitialPledge = "0"
						minerInfo.ExpectedStoragePledge = "0"
						minerInfo.PreCommitDeposit = "0"
						minerInfo.SectorId = 0
					}
					cell = row.AddCell()
					//cell.Value = strconv.FormatInt(minerInfo.InitialPledge, 10)
					cell.Value = minerInfo.InitialPledge
					cell = row.AddCell()
					cell.Value = minerInfo.ExpectedStoragePledge
					cell = row.AddCell()
					cell.Value = minerInfo.PreCommitDeposit
					cell = row.AddCell()
					cell.Value = strconv.Itoa(minerInfo.SectorId)

					cell = row.AddCell()
					cell.Value = minerAvailableBalance.(string)
					cell = row.AddCell()
					cell.Value = preCommitDeposits.(string)
					cell = row.AddCell()
					cell.Value = lockedFunds.(string)
					cell = row.AddCell()
					cell.Value = initialPledge.(string)
					cell = row.AddCell()
					cell.Value = balance.(string)
					cell = row.AddCell()
					cell.Value = minerBalance.(string)
					if dg-1 == j {
						cell = row.AddCell()
						cell.Value = totalCostFee
						fmt.Printf("totalCostFee: %s:\n", totalCostFee)
					} else {
						cell = row.AddCell()
						cell.Value = strconv.FormatInt(0, 10)
					}
				}
			}
		} else {
			minerInfo.MinerId = minerId
			minerInfo.PreCommitDeposit = "0"
			minerInfo.Value = "0"
			minerInfo.BaseFeeBurn = "0"
			minerInfo.OverEstimationBurn = "0"
			minerInfo.MinerPenalty = 0
			minerInfo.MinerTip = "0"
			minerInfo.Refund = "0"
			minerInfo.GasRefund = "0"
			minerInfo.Method = 0
			minerInfo.InitialPledge = "0"
			minerInfo.ExpectedStoragePledge = "0"
			minerInfo.SectorId = 0

			row = sheet.AddRow()
			cell = row.AddCell()
			cell.Value = "" // cid
			cell = row.AddCell()
			cell.Value = minerInfo.MinerId
			cell = row.AddCell()
			cell.Value = strconv.FormatInt(i, 10)
			cell = row.AddCell()
			cell.Value = minerInfo.Value
			cell = row.AddCell()
			cell.Value = minerInfo.BaseFeeBurn
			cell = row.AddCell()
			cell.Value = minerInfo.OverEstimationBurn
			cell = row.AddCell()
			cell.Value = strconv.Itoa(minerInfo.MinerPenalty)
			cell = row.AddCell()
			cell.Value = minerInfo.MinerTip
			cell = row.AddCell()
			cell.Value = minerInfo.Refund
			cell = row.AddCell()
			cell.Value = minerInfo.GasRefund
			cell = row.AddCell()
			cell.Value = strconv.Itoa(minerInfo.Method)
			cell = row.AddCell()
			cell.Value = minerInfo.InitialPledge
			cell = row.AddCell()
			cell.Value = minerInfo.ExpectedStoragePledge
			cell = row.AddCell()
			cell.Value = minerInfo.PreCommitDeposit
			cell = row.AddCell()
			cell.Value = strconv.Itoa(minerInfo.SectorId)
			cell = row.AddCell()
			cell.Value = minerAvailableBalance.(string)
			cell = row.AddCell()
			cell.Value = preCommitDeposits.(string)
			cell = row.AddCell()
			cell.Value = lockedFunds.(string)
			cell = row.AddCell()
			cell.Value = initialPledge.(string)
			cell = row.AddCell()
			cell.Value = balance.(string)
			cell = row.AddCell()
			cell.Value = minerBalance.(string)
			cell = row.AddCell()
			cell.Value = "0" // totalCostFee

		}

		file.Save(minerId + "File.xlsx")
	}

	return minerInfo, "", 0
}

// get account bills
func (s *AccountingServer) GetAccountInfoRequest(writer http.ResponseWriter, request *http.Request) (interface{}, string, int) {

	query := request.URL.Query()
	// 账户
	account := string(query["account"][0])
	// 初始高度0 出块时间 1598306400 后面每一个加30s
	startTime := string(query["startTime"][0])
	startTimestamp, _ := strconv.ParseInt(startTime, 10, 32)
	//startHeightInt64 := atomic.AddInt64(&startTimestamp, -1598306400)
	// beijing time
	startHeightInt64 := atomic.AddInt64(&startTimestamp, -1598306400)
	realStartHeight := startHeightInt64 / 30
	endTime := string(query["endTime"][0])
	endTimestamp, _ := strconv.ParseInt(endTime, 10, 32)
	endHeightInt64 := atomic.AddInt64(&endTimestamp, -1598306400)
	realEndHeight := endHeightInt64 / 30
	fmt.Printf("account: %s,realStartHeight: %s,realEndHeight: %s", account, realStartHeight, realEndHeight)
	pageSizeStr := string(query["pageSize"][0]) // 分页大小
	pageSize, _ := strconv.Atoi(pageSizeStr)
	currentPageStr := string(query["currentPage"][0]) // 当前第几页
	currentPage, _ := strconv.Atoi(currentPageStr)

	var accountType = ""
	if strings.HasPrefix(account, "f1") {
		accountType = "normal"
	} else if strings.HasPrefix(account, "f3") {
		accountType = "worker" // worker address
	} else if strings.HasPrefix(account, "f0") { // if begin f0 maybe miner id  or f1 f2 f3' address id
		// check account
		result := filrpc.ChainHead()
		var resultMap map[string]interface{}
		if err := json.Unmarshal([]byte(result), &resultMap); err == nil {
			cid := resultMap["result"].(map[string]interface{})["Cids"].([]interface{})[0].(map[string]interface{})["/"]
			cidStr := cid.(string)
			resultLookupID := filrpc.StateLookupID(account, cidStr)
			var resulLookupIDMap map[string]interface{}
			json.Unmarshal([]byte(resultLookupID), &resulLookupIDMap)
			resultId := resulLookupIDMap["result"]
			if strings.HasPrefix(resultId.(string), "f0") {
				// TODO check
				accountType = "miner"
			} else if strings.HasPrefix(resultId.(string), "f1") {
				accountType = "normal"
				// change to address because of database
				account = resultId.(string)
			} else {
				accountType = "worker"
				account = resultId.(string)
			}
		}
	} else {
		return nil, "no support f2 address", 0
	}

	if strings.EqualFold(accountType, "normal") {

		infos := s.findAccountInfoByAccountAndBlockNo(account, realStartHeight, realEndHeight, pageSize, currentPage)
		return infos, "", 0

	} else if strings.EqualFold(accountType, "worker") {
		infos := s.findWorkerInfoByAccountAndBlockNo(account, realStartHeight, realEndHeight, pageSize, currentPage)
		return infos, "", 0

	} else if strings.EqualFold(accountType, "miner") {
		infos := s.findMinerInfoByAccountAndBlockNo(account, realStartHeight, realEndHeight, pageSize, currentPage)
		return infos, "", 0
	}
	return nil, "not found anything", 0
}

// normal account
func (s *AccountingServer) findAccountInfoByAccountAndBlockNo(account string, realStartHeight int64, realEndHeight int64, pageSize int, currentPage int) map[string]interface{} {
	res := make(map[string]interface{})
	if pageSize != 0 && pageSize != 0 {
		rsCount := realEndHeight - realStartHeight
		res = utils.Paginator(currentPage, pageSize, rsCount)
		realEndHeight = realStartHeight + int64(currentPage)*int64(pageSize)
		realStartHeight = realStartHeight + (int64(currentPage)-1)*int64(pageSize)
	}

	var infos []types.AccountInfo
	var accountBalance interface{}
	for i := realStartHeight; i < realEndHeight; i++ {
		// 统计fee minertip totalvalue
		iStr := strconv.FormatInt(i, 10)
		var resultMap map[string]interface{}
		result := filrpc.GetMinerInfoByMinerIdAndHeight(account, iStr)
		if err := json.Unmarshal([]byte(result), &resultMap); err == nil {
			cid := resultMap["result"].(map[string]interface{})["Cids"].([]interface{})[0].(map[string]interface{})["/"]
			cidStr := cid.(string)
			var stateGetActorMinerMap map[string]interface{}
			stateGetActorMinerResult := filrpc.StateGetActor(account, cidStr)
			if err := json.Unmarshal([]byte(stateGetActorMinerResult), &stateGetActorMinerMap); err == nil {
				accountBalance = stateGetActorMinerMap["result"].(map[string]interface{})["Balance"]
			}

			infos[i].Id = account
			infos[i].Balance = accountBalance.(string)
			infos[i].BlockHeight = i
			totalBurnFee := "0"
			totalMinerTip := "0"
			totalSend := "0"
			totalSendIn := "0"
			totalSendOut := "0"
			// table derived_gas_outputs
			derivedGasOutputs, _ := s.PostgresClient.QueryDerivedGasOutputs(account, i)
			if len(derivedGasOutputs) > 0 {
				for j := 0; j < len(derivedGasOutputs); j++ {
					// 出账才有手续费
					if strings.EqualFold(derivedGasOutputs[j].From, account) {
						totalBurnFee = utils.BigIntAddStr(totalBurnFee, derivedGasOutputs[i].BaseFeeBurn)
						totalBurnFee = utils.BigIntAddStr(totalBurnFee, derivedGasOutputs[i].OverEstimationBurn)
						totalMinerTip = utils.BigIntAddStr(totalMinerTip, derivedGasOutputs[i].MinerTip)
					}
					if strings.EqualFold(derivedGasOutputs[i].From, account) && derivedGasOutputs[j].Method == 0 { // sub
						totalSend = utils.BigIntReduceStr(totalSend, derivedGasOutputs[i].Value)
						totalSendOut = utils.BigIntReduceStr(totalSendOut, derivedGasOutputs[i].Value)
					} else if strings.EqualFold(derivedGasOutputs[i].To, account) && derivedGasOutputs[j].Method == 0 { // add
						totalSend = utils.BigIntAddStr(totalSend, derivedGasOutputs[i].Value)
						totalSendIn = utils.BigIntAddStr(totalSendIn, derivedGasOutputs[i].Value)
					}
				}
			}
			infos[i].Fee = totalBurnFee
			infos[i].MinerTip = totalMinerTip
			infos[i].Send = totalSend // sub + add
			infos[i].SendIn = totalSendIn
			infos[i].SendOut = totalSendOut
		}
	}
	res["data"] = infos
	return res
}

// worker
func (s *AccountingServer) findWorkerInfoByAccountAndBlockNo(account string, realStartHeight int64, realEndHeight int64, pageSize int, currentPage int) map[string]interface{} {

	t := time.Now()
	res := make(map[string]interface{})
	if pageSize != 0 && pageSize != 0 {
		rsCount := realEndHeight - realStartHeight
		res = utils.Paginator(currentPage, pageSize, rsCount)
		realEndHeight = realStartHeight + int64(currentPage)*int64(pageSize)
		realStartHeight = realStartHeight + (int64(currentPage)-1)*int64(pageSize)
	}

	var infos []types.WorkerInfo
	var workerBalance interface{}
	for i := realStartHeight; i < realEndHeight; i++ {
		// 统计fee minertip totalvalue

		iStr := strconv.FormatInt(i, 10)
		var resultMap map[string]interface{}
		result := filrpc.GetMinerInfoByMinerIdAndHeight(account, iStr)
		if err := json.Unmarshal([]byte(result), &resultMap); err == nil {
			cid := resultMap["result"].(map[string]interface{})["Cids"].([]interface{})[0].(map[string]interface{})["/"]
			cidStr := cid.(string)
			var stateGetActorMinerMap map[string]interface{}
			stateGetActorMinerResult := filrpc.StateGetActor(account, cidStr)
			if err := json.Unmarshal([]byte(stateGetActorMinerResult), &stateGetActorMinerMap); err == nil {
				workerBalance = stateGetActorMinerMap["result"].(map[string]interface{})["Balance"]
			}
			var info = types.WorkerInfo{}
			info.Id = account
			info.Balance = workerBalance.(string)
			info.BlockHeight = i
			totalBurnFee := "0"
			totalMinerTip := "0"
			totalSend := "0"
			totalSendIn := "0"
			totalSendOut := "0"
			totalPreCommitSectors := "0"
			totalProveCommitSectors := "0"
			// table derived_gas_outputs
			derivedGasOutputs, _ := s.PostgresClient.QueryDerivedGasOutputs(account, i)
			if len(derivedGasOutputs) > 0 {
				for j := 0; j < len(derivedGasOutputs); j++ {
					// 出账才有手续费
					if strings.EqualFold(derivedGasOutputs[j].From, account) {
						totalBurnFee = utils.BigIntAddStr(totalBurnFee, derivedGasOutputs[j].BaseFeeBurn)
						totalBurnFee = utils.BigIntAddStr(totalBurnFee, derivedGasOutputs[j].OverEstimationBurn)
						totalMinerTip = utils.BigIntAddStr(totalMinerTip, derivedGasOutputs[j].MinerTip)
					}
					if strings.EqualFold(derivedGasOutputs[j].From, account) && derivedGasOutputs[j].Method == 0 { // sub
						totalSend = utils.BigIntReduceStr(totalSend, derivedGasOutputs[j].Value)
						totalSendOut = utils.BigIntReduceStr(totalSendOut, derivedGasOutputs[j].Value)
					} else if strings.EqualFold(derivedGasOutputs[j].To, account) && derivedGasOutputs[j].Method == 0 { // add
						totalSend = utils.BigIntAddStr(totalSend, derivedGasOutputs[j].Value)
						totalSendIn = utils.BigIntAddStr(totalSendIn, derivedGasOutputs[j].Value)
					}
					if derivedGasOutputs[j].Method == 6 {
						totalPreCommitSectors = utils.BigIntAddStr(totalPreCommitSectors, derivedGasOutputs[j].Value)
					}
					if derivedGasOutputs[j].Method == 7 {
						totalProveCommitSectors = utils.BigIntAddStr(totalProveCommitSectors, derivedGasOutputs[j].Value)
					}
				}
			}
			info.Fee = totalBurnFee
			info.MinerTip = totalMinerTip
			info.Send = totalSend // sub + add
			info.SendIn = totalSendIn
			info.SendOut = totalSendOut
			info.PreCommitSectors = totalPreCommitSectors
			info.ProveCommitSectors = totalProveCommitSectors
			infos = append(infos, info)
		}
	}
	f, err := os.Create("./" + account + "filecoin-" + strconv.FormatInt(realStartHeight, 10) + "-" + strconv.FormatInt(realEndHeight, 10) + ".csv")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	f.WriteString("\xEF\xBB\xBF") // 写入UTF-8 BOM
	w := csv.NewWriter(f)
	w.Write([]string{"Id", "Balance", "BlockHeight", "Fee", "MinerTip", "SendIn", "SendOut", "Send", "PreCommitSectors",
		"ProveCommitSectors"})
	for i := 0; i < len(infos); i++ {
		w.Write([]string{infos[i].Id + "\t", infos[i].Balance + "\t", strconv.FormatInt(infos[i].BlockHeight, 10) + "\t", infos[i].Fee + "\t",
			infos[i].MinerTip + "\t", infos[i].SendIn + "\t", infos[i].SendOut + "\t",
			infos[i].Send + "\t", infos[i].PreCommitSectors + "\t", infos[i].ProveCommitSectors + "\t"})
	}
	w.Flush()
	res["data"] = infos
	spendtime := time.Since(t)
	fmt.Println("spend time:", spendtime)
	return res
}

// miner
func (s *AccountingServer) findMinerInfoByAccountAndBlockNo(account string, realStartHeight int64, realEndHeight int64, pageSize int, currentPage int) map[string]interface{} {

	t := time.Now()

	if pageSize < 0 {
		pageSize = 20
	}
	if currentPage < 0 {
		currentPage = 1
	}
	res := make(map[string]interface{})
	if pageSize != 0 && pageSize != 0 {
		rsCount := realEndHeight - realStartHeight
		res = utils.Paginator(currentPage, pageSize, rsCount)
		realEndHeight = realStartHeight + int64(currentPage)*int64(pageSize)
		realStartHeight = realStartHeight + (int64(currentPage)-1)*int64(pageSize)
	}

	var infos []types.MinerInfo
	var minerAvailableBalance interface{}
	var preCommitDeposits interface{}
	var lockedFunds interface{}
	var initialPledge interface{}
	var minerBalance interface{}
	totalSubLockFunds := "0"

	for i := realStartHeight; i <= realEndHeight; i++ {
		// 统计fee minertip totalvalue
		// table derived_gas_outputs
		iStr := strconv.FormatInt(i, 10)
		var resultMap map[string]interface{}
		result := filrpc.GetMinerInfoByMinerIdAndHeight(account, iStr)
		if err := json.Unmarshal([]byte(result), &resultMap); err == nil {
			cid := resultMap["result"].(map[string]interface{})["Cids"].([]interface{})[0].(map[string]interface{})["/"]
			reallyBlockNo := resultMap["result"].(map[string]interface{})["Height"]
			int64ReallyBlockNo := int64(reallyBlockNo.(float64))
			var flagBlockIsNull = false
			if int64ReallyBlockNo == i-1 {
				flagBlockIsNull = true
			}
			cidStr := cid.(string)
			var stateGetActorMinerMap map[string]interface{}
			stateGetActorMinerResult := filrpc.StateGetActor(account, cidStr)
			if err := json.Unmarshal([]byte(stateGetActorMinerResult), &stateGetActorMinerMap); err == nil {
				minerBalance = stateGetActorMinerMap["result"].(map[string]interface{})["Balance"]
				minerAvailableBalanceResult := filrpc.StateMinerAvailableBalance(account, cidStr)
				var minerAvailableBalanceMap map[string]interface{}
				if err := json.Unmarshal([]byte(minerAvailableBalanceResult), &minerAvailableBalanceMap); err == nil {
					minerAvailableBalance = minerAvailableBalanceMap["result"]

					stateReadStateResult := filrpc.StateReadState(account, cidStr)
					var stateReadStateMap map[string]interface{}
					if err := json.Unmarshal([]byte(stateReadStateResult), &stateReadStateMap); err == nil {
						preCommitDeposits = stateReadStateMap["result"].(map[string]interface{})["State"].(map[string]interface{})["PreCommitDeposits"]
						lockedFunds = stateReadStateMap["result"].(map[string]interface{})["State"].(map[string]interface{})["LockedFunds"]
						initialPledge = stateReadStateMap["result"].(map[string]interface{})["State"].(map[string]interface{})["InitialPledge"]
					}
				}
			}
			var info = types.MinerInfo{}
			info.Id = account
			info.Balance = minerBalance.(string)
			info.BlockHeight = i
			totalBurnFee := "0"
			totalMinerTip := "0"
			totalSend := "0"
			totalSendIn := "0"
			totalSendOut := "0"
			totalPreCommitSectors := "0"
			totalProveCommitSectors := "0"
			totalWithdrawBalance := "0"
			TotalTodayRewards := "0"

			derivedGasOutputs, _ := s.PostgresClient.QueryDerivedGasOutputs(account, i)
			if len(derivedGasOutputs) > 0 {
				for j := 0; j < len(derivedGasOutputs); j++ {
					// 出账才有手续费
					if strings.EqualFold(derivedGasOutputs[j].From, account) {
						totalBurnFee = utils.BigIntAddStr(totalBurnFee, derivedGasOutputs[j].BaseFeeBurn)
						totalBurnFee = utils.BigIntAddStr(totalBurnFee, derivedGasOutputs[j].OverEstimationBurn)
						totalMinerTip = utils.BigIntAddStr(totalMinerTip, derivedGasOutputs[j].MinerTip)
					}
					if strings.EqualFold(derivedGasOutputs[j].From, account) && derivedGasOutputs[j].Method == 0 { // sub
						totalSend = utils.BigIntReduceStr(totalSend, derivedGasOutputs[j].Value)
						totalSendOut = utils.BigIntReduceStr(totalSendOut, derivedGasOutputs[j].Value)
					} else if strings.EqualFold(derivedGasOutputs[j].To, account) && derivedGasOutputs[j].Method == 0 { // add
						totalSend = utils.BigIntAddStr(totalSend, derivedGasOutputs[j].Value)
						totalSendIn = utils.BigIntAddStr(totalSendIn, derivedGasOutputs[j].Value)
					}
					if derivedGasOutputs[j].Method == 6 {
						totalPreCommitSectors = utils.BigIntAddStr(totalPreCommitSectors, derivedGasOutputs[j].Value)
					}
					if derivedGasOutputs[j].Method == 7 {
						totalProveCommitSectors = utils.BigIntAddStr(totalProveCommitSectors, derivedGasOutputs[j].Value)
					}
					// miner withdrawBalance
					if derivedGasOutputs[j].Method == 16 {

						stateReplayResult := filrpc.StateReplay(cidStr, derivedGasOutputs[j].Cid)
						if stateReplayResult != "" {
							var stateReadStateMap map[string]interface{}
							if err := json.Unmarshal([]byte(stateReplayResult), &stateReadStateMap); err == nil {
								withdrawBalance := stateReadStateMap["result"].(map[string]interface{})["ExecutionTrace"].(map[string]interface{})["Subcalls"].([]interface{})[0].(map[string]interface{})["Msg"].(map[string]interface{})["Value"]
								totalWithdrawBalance = utils.BigIntAddStr(totalWithdrawBalance, withdrawBalance.(string))
							}
						}
					}
				}
			}

			info.Fee = totalBurnFee
			info.MinerTip = totalMinerTip
			info.Send = totalSend // sub + add
			info.SendIn = totalSendIn
			info.SendOut = totalSendOut
			info.PreCommitDeposits = preCommitDeposits.(string)
			info.PreCommitSectors = totalPreCommitSectors
			info.ProveCommitSectors = totalProveCommitSectors

			info.WithdrawBalance = totalWithdrawBalance
			info.BlockReward = "0"
			info.FlagBlockIsNull = flagBlockIsNull
			info.PunishFee = "0"
			info.MinerAvailableBalance = minerAvailableBalance.(string)
			info.LockedFunds = lockedFunds.(string)
			info.InitialPledge = initialPledge.(string)

			info.Balance = minerBalance.(string)
			infos = append(infos, info)
			var k = len(infos)
			if k >= 2 {
				subBalance := utils.BigIntReduceStr(infos[k-1].Balance, infos[k-2].Balance)
				totalPreCommitSectors := utils.BigIntAddStr(infos[k-2].PreCommitSectors, infos[k-2].ProveCommitSectors)

				// TODO 出块奖励
				infos[k-2].BlockReward = utils.BigIntReduceStr(subBalance, totalPreCommitSectors)
				blockReward, _ := strconv.ParseInt(infos[k-2].BlockReward, 10, 64)
				if blockReward > 0 {
					infos[k-2].BlockReward = utils.BigIntReduceStr(infos[k-2].BlockReward, infos[k-2].Send)
					infos[k-2].BlockReward = utils.BigIntAddStr(infos[k-2].BlockReward, infos[k-2].WithdrawBalance)
					k3totalPreCommitSectors := utils.BigIntAddStr(infos[k-3].PreCommitSectors, infos[k-3].ProveCommitSectors)
					var p = utils.BigIntReduceStr(infos[k-2].BlockReward, k3totalPreCommitSectors)
					// TODO block is null
					//if strings.EqualFold(p, "0") && !strings.EqualFold(k3totalPreCommitSectors, "0") {
					//	infos[k-2].BlockReward = "0"
					//	infos[k-2].TAG = "block is null"
					//}
					if infos[k-2].FlagBlockIsNull && strings.EqualFold(p, "0") && !strings.EqualFold(k3totalPreCommitSectors, "0") {
						infos[k-3].BlockReward = infos[k-2].BlockReward //将值赋值给上一层
						infos[k-2].BlockReward = "0"
						infos[k-2].TAG = "block is null"
					}
				} else if blockReward < 0 {
					// maybe this block is null or other bug
					infos[k-2].BlockReward = "0"
				}

				// TODO 线性释放金额
				// 求某个高度 线性释放的金额  = （下一高度的可用余额-上一个高度可用余额）+ withdrawBalance -(上一个高度的(pre + pro + InitialPledge + PreCommitDeposits + send) - 下一个高度的（PreCommitDeposits+InitialPledge）)
				// 连续三个高度中间出现空块的情况
				if k >= 3 {
					if !infos[k-1].FlagBlockIsNull && infos[k-2].FlagBlockIsNull && !infos[k-3].FlagBlockIsNull {
						add1 := utils.BigIntAddStr(infos[k-1].PreCommitDeposits, infos[k-1].InitialPledge)
						add1 = utils.BigIntAddStr(add1, infos[k-1].MinerAvailableBalance)
						add3 := utils.BigIntAddStr(infos[k-3].PreCommitDeposits, infos[k-3].InitialPledge)
						add3 = utils.BigIntAddStr(add3, infos[k-3].Send)
						// k3 totalPreCommitSector
						totalPreCommitSector := utils.BigIntAddStr(infos[k-3].PreCommitSectors, infos[k-3].ProveCommitSectors)
						add3 = utils.BigIntAddStr(totalPreCommitSector, add3)
						add3 = utils.BigIntAddStr(infos[k-3].MinerAvailableBalance, add3)
						if !strings.EqualFold(infos[k-3].WithdrawBalance, "0") { // miner withdrawBalance
							add3 = utils.BigIntReduceStr(add3, infos[k-3].WithdrawBalance)
						}
						subBlockRewardAvalible := utils.BigIntReduceStr(add1, add3)
						TotalTodayRewards = utils.BigIntAddStr(subBlockRewardAvalible, TotalTodayRewards)
						// 连续三个高度不出现空块的情况
					} else if !infos[k-1].FlagBlockIsNull && !infos[k-2].FlagBlockIsNull && !infos[k-3].FlagBlockIsNull {
						if k == 3 {
							add2 := utils.BigIntAddStr(infos[k-2].PreCommitDeposits, infos[k-2].InitialPledge)
							add2 = utils.BigIntAddStr(add2, infos[k-2].MinerAvailableBalance)
							add3 := utils.BigIntAddStr(infos[k-3].PreCommitDeposits, infos[k-3].InitialPledge)
							add3 = utils.BigIntAddStr(add3, infos[k-3].Send)
							totalPreCommitSector := utils.BigIntAddStr(infos[k-3].PreCommitSectors, infos[k-3].ProveCommitSectors)
							add3 = utils.BigIntAddStr(totalPreCommitSector, add3)
							add3 = utils.BigIntAddStr(infos[k-3].MinerAvailableBalance, add3)
							if !strings.EqualFold(infos[k-3].WithdrawBalance, "0") { // miner withdrawBalance
								add3 = utils.BigIntReduceStr(add3, infos[k-3].WithdrawBalance)
							}
							subBlockRewardAvalible := utils.BigIntReduceStr(add2, add3)
							TotalTodayRewards = utils.BigIntAddStr(subBlockRewardAvalible, TotalTodayRewards)
						} else {
							add1 := utils.BigIntAddStr(infos[k-1].PreCommitDeposits, infos[k-1].InitialPledge)
							add1 = utils.BigIntAddStr(add1, infos[k-1].MinerAvailableBalance)
							add2 := utils.BigIntAddStr(infos[k-2].PreCommitDeposits, infos[k-2].InitialPledge)
							add2 = utils.BigIntAddStr(add2, infos[k-2].Send)
							add2 = utils.BigIntAddStr(totalPreCommitSectors, add2)
							add2 = utils.BigIntAddStr(infos[k-2].MinerAvailableBalance, add2)
							if !strings.EqualFold(infos[k-2].WithdrawBalance, "0") { // miner withdrawBalance
								add2 = utils.BigIntReduceStr(add2, infos[k-2].WithdrawBalance)
							}
							subBlockRewardAvalible := utils.BigIntReduceStr(add1, add2)
							TotalTodayRewards = utils.BigIntAddStr(subBlockRewardAvalible, TotalTodayRewards)
						}
					}
				}

				// TODO lockFunds
				subLockFunds := "0"
				if k >= 3 {
					// 中间出现空块
					if !infos[k-1].FlagBlockIsNull && infos[k-2].FlagBlockIsNull && !infos[k-3].FlagBlockIsNull {
						subLockFunds = utils.BigIntReduceStr(infos[k-1].LockedFunds, infos[k-3].LockedFunds)
						if !strings.Contains(subLockFunds, "-") {
							totalSubLockFunds = utils.BigIntAddStr(subLockFunds, totalSubLockFunds)
						}
						// 连续不出现空块
					} else if !infos[k-1].FlagBlockIsNull && !infos[k-2].FlagBlockIsNull && !infos[k-3].FlagBlockIsNull {
						if k == 3 {
							subLockFunds = utils.BigIntReduceStr(infos[k-2].LockedFunds, infos[k-3].LockedFunds)
							if !strings.Contains(subLockFunds, "-") {
								totalSubLockFunds = utils.BigIntAddStr(subLockFunds, totalSubLockFunds)
							}
							subLockFunds = utils.BigIntReduceStr(infos[k-1].LockedFunds, infos[k-2].LockedFunds)
							if !strings.Contains(subLockFunds, "-") {
								totalSubLockFunds = utils.BigIntAddStr(subLockFunds, totalSubLockFunds)
							}
						} else {
							subLockFunds = utils.BigIntReduceStr(infos[k-1].LockedFunds, infos[k-2].LockedFunds)
							if !strings.Contains(subLockFunds, "-") {
								totalSubLockFunds = utils.BigIntAddStr(subLockFunds, totalSubLockFunds)
							}
						}

					}

				}
				infos[k-2].SubLockFunds = subLockFunds

				fmt.Printf("--blockNo:" + strconv.FormatInt(i, 10) + "--TotalTodayRewards:" + TotalTodayRewards + "totalSubLockFunds:" + totalSubLockFunds + "\n")

				// TODO 惩罚
				addBalance := utils.BigIntAddStr(totalPreCommitSectors, infos[k-2].BlockReward)
				addBalance = utils.BigIntAddStr(addBalance, infos[k-2].Send)
				if !strings.EqualFold(addBalance, subBalance) {
					if !strings.EqualFold(infos[k-2].TAG, "block is null") {
						infos[k-2].TAG = "惩罚(销毁)"
						infos[k-2].PunishFee = utils.BigIntReduceStr(addBalance, subBalance)
						if !strings.EqualFold(infos[k-2].WithdrawBalance, "0") { // 去掉提现的部分
							fmt.Printf("--blockNo:" + strconv.FormatInt(i, 10) + "---totalWithdrawBalance-------:" + infos[k-2].WithdrawBalance + "\n")
							infos[k-2].PunishFee = utils.BigIntReduceStr(infos[k-2].PunishFee, infos[k-2].WithdrawBalance)
						}
						fmt.Printf("--blockNo:" + strconv.FormatInt(i, 10) + "---burn-PunishFee-------:" + infos[k-2].PunishFee + "\n")
					} else {
						infos[k-3].PunishFee = "0"
						infos[k-3].TAG = ""
					}
					// 或者 subBalance=addBalance 且 lockFunds 不相等
				} else if strings.EqualFold(infos[k-1].Balance, infos[k-2].Balance) && !strings.EqualFold(subLockFunds, "0") {
					fmt.Printf("--blockNo:" + strconv.FormatInt(i, 10) + "---lockFunds不相等 :" + subLockFunds + "\n")
					infos[k-2].TAG = "掉算力惩罚(销毁)"
					infos[k-2].PunishFee = subLockFunds
				}

			}

		}
	}

	f, err := os.Create("./" + account + "filecoin-" + strconv.FormatInt(realStartHeight, 10) + "-" + strconv.FormatInt(realEndHeight, 10) + ".csv")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	f.WriteString("\xEF\xBB\xBF") // 写入UTF-8 BOM
	w := csv.NewWriter(f)
	w.Write([]string{"Id", "Balance", "BlockHeight", "Fee", "MinerTip", "SendIn", "SendOut", "Send", "PreCommitSectors",
		"ProveCommitSectors", "PunishFee", "PreCommitDeposits", "BlockReward", "TAG", "MinerAvailableBalance", "LockedFunds",
		"InitialPledge", "subLockFunds", "WithdrawBalance"})
	for i := 0; i < len(infos); i++ {
		w.Write([]string{infos[i].Id + "\t", infos[i].Balance + "\t", strconv.FormatInt(infos[i].BlockHeight, 10) + "\t", infos[i].Fee + "\t",
			infos[i].MinerTip + "\t", infos[i].SendIn + "\t", infos[i].SendOut + "\t",
			infos[i].Send + "\t", infos[i].PreCommitSectors + "\t", infos[i].ProveCommitSectors + "\t", infos[i].PunishFee + "\t",
			infos[i].PreCommitDeposits + "\t", infos[i].BlockReward + "\t", infos[i].TAG + "\t", infos[i].MinerAvailableBalance + "\t",
			infos[i].LockedFunds + "\t", infos[i].InitialPledge + "\t",
			infos[i].SubLockFunds + "\t", infos[i].WithdrawBalance + "\t"})
	}
	w.Flush()
	res["data"] = infos
	spendtime := time.Since(t)
	fmt.Println("spend time:", spendtime)
	return res

}
