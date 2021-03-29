package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	log "github.com/EntropyPool/entropy-logger"
	fbcpostgres "github.com/EntropyPool/fbc-accounting-service/postgres"
	types "github.com/EntropyPool/fbc-accounting-service/types"
	httpdaemon "github.com/NpoolRD/http-daemon"
	"github.com/tealeg/xlsx"
	"io/ioutil"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	_ "strings"
	"sync/atomic"
	_ "time"
	"unsafe"
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

func (s *AccountingServer) GetMinerPledgeRequest(w http.ResponseWriter, req *http.Request) (interface{}, string, int) {

	query := req.URL.Query()
	minerId := string(query["minerId"][0])
	minerPreCommitInfo, _ := s.PostgresClient.QueryMinerPreCommitInfo(minerId)
	return minerPreCommitInfo, "", 0
}

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
	startHeightInt64 := atomic.AddInt64(&startTimestamp, -1598277600)
	realStartHeight := startHeightInt64 / 30
	fmt.Printf("realStartHeight%s", realStartHeight)

	endTime := string(query["endTime"][0])
	endTimestamp, _ := strconv.ParseInt(endTime, 10, 32)
	endHeightInt64 := atomic.AddInt64(&endTimestamp, -1598277600)
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
		result := GetMinerInfoByMinerIdAndHeight(minerId, iStr)
		var resultMap map[string]interface{}
		var minerAvailableBalance interface{}
		var preCommitDeposits interface{}
		var lockedFunds interface{}
		var initialPledge interface{}
		var balance interface{}
		var minerBalance interface{}
		if err := json.Unmarshal([]byte(result), &resultMap); err == nil {
			cid := resultMap["result"].(map[string]interface{})["Cids"].([]interface{})[0].(map[string]interface{})["/"]
			cidStr := cid.(string)
			minerAvailableBalanceResult := StateMinerAvailableBalance(minerId, cidStr)
			var minerAvailableBalanceMap map[string]interface{}
			if err := json.Unmarshal([]byte(minerAvailableBalanceResult), &minerAvailableBalanceMap); err == nil {
				minerAvailableBalance = minerAvailableBalanceMap["result"]

				stateReadStateResult := StateReadState(minerId, cidStr)
				var stateReadStateMap map[string]interface{}
				if err := json.Unmarshal([]byte(stateReadStateResult), &stateReadStateMap); err == nil {
					preCommitDeposits = stateReadStateMap["result"].(map[string]interface{})["State"].(map[string]interface{})["PreCommitDeposits"]
					lockedFunds = stateReadStateMap["result"].(map[string]interface{})["State"].(map[string]interface{})["LockedFunds"]
					initialPledge = stateReadStateMap["result"].(map[string]interface{})["State"].(map[string]interface{})["InitialPledge"]
				}
				var stateGetActorMinerMap map[string]interface{}
				stateGetActorMinerResult := StateGetActor(minerId, cidStr)
				if err := json.Unmarshal([]byte(stateGetActorMinerResult), &stateGetActorMinerMap); err == nil {
					minerBalance = stateGetActorMinerMap["result"].(map[string]interface{})["Balance"]
				}
				var stateMinerInfoMap map[string]interface{}
				stateMinerInfoResult := StateMinerInfo(minerId, cidStr)
				if err := json.Unmarshal([]byte(stateMinerInfoResult), &stateMinerInfoMap); err == nil {
					workerId := stateMinerInfoMap["result"].(map[string]interface{})["Worker"]
					workerIdStr := workerId.(string)
					var stateGetActorMap map[string]interface{}
					stateGetActorResult := StateGetActor(workerIdStr, cidStr)
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
					totalCostFee = BigIntAddStr(totalCostFee, minerInfo.BaseFeeBurn)
					totalCostFee = BigIntAddStr(totalCostFee, minerInfo.OverEstimationBurn)
					totalCostFee = BigIntAddStr(totalCostFee, minerInfo.MinerTip)

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
						minerPreCommitInfo, _ := s.PostgresClient.QueryMinerPreCommitInfoAndSectorId(minerId, minerSectorInfos[j].SectorId)
						if minerPreCommitInfo != nil {
							minerInfo.PreCommitDeposit = minerPreCommitInfo.PreCommitDeposit
						} else {
							minerInfo.PreCommitDeposit = "0"
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
	// 初始高度0 出块时间 1598277600 后面每一个加30s
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

	var accountType = ""
	if strings.HasPrefix(account, "f1") {
		accountType = "normal"
	} else if strings.HasPrefix(account, "f3") {
		accountType = "worker" // worker address
	} else if strings.HasPrefix(account, "f0") { // if begin f0 maybe miner id  or f1 f2 f3' address id
		// check account
		result := ChainHead()
		var resultMap map[string]interface{}
		if err := json.Unmarshal([]byte(result), &resultMap); err == nil {
			cid := resultMap["result"].(map[string]interface{})["Cids"].([]interface{})[0].(map[string]interface{})["/"]
			cidStr := cid.(string)
			resultLookupID := StateLookupID(account, cidStr)
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

		infos := s.findAccountInfoByAccountAndBlockNo(account, realStartHeight, realEndHeight)
		return infos, "", 0

	} else if strings.EqualFold(accountType, "worker") {
		infos := s.findWorkerInfoByAccountAndBlockNo(account, realStartHeight, realEndHeight)
		return infos, "", 0

	} else if strings.EqualFold(accountType, "miner") {
		infos := s.findMinerInfoByAccountAndBlockNo(account, realStartHeight, realEndHeight)
		return infos, "", 0
	}
	return nil, "not found anything", 0
}

// account
func (s *AccountingServer) findAccountInfoByAccountAndBlockNo(account string, realStartHeight int64, realEndHeight int64) []types.AccountInfo {
	var infos []types.AccountInfo
	var accountBalance interface{}
	for i := realStartHeight; i < realEndHeight; i++ {
		// 统计fee minertip totalvalue
		iStr := strconv.FormatInt(i, 10)
		var resultMap map[string]interface{}
		result := GetMinerInfoByMinerIdAndHeight(account, iStr)
		if err := json.Unmarshal([]byte(result), &resultMap); err == nil {
			cid := resultMap["result"].(map[string]interface{})["Cids"].([]interface{})[0].(map[string]interface{})["/"]
			cidStr := cid.(string)
			var stateGetActorMinerMap map[string]interface{}
			stateGetActorMinerResult := StateGetActor(account, cidStr)
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
						totalBurnFee = BigIntAddStr(totalBurnFee, derivedGasOutputs[i].BaseFeeBurn)
						totalBurnFee = BigIntAddStr(totalBurnFee, derivedGasOutputs[i].OverEstimationBurn)
						totalMinerTip = BigIntAddStr(totalMinerTip, derivedGasOutputs[i].MinerTip)
					}
					if strings.EqualFold(derivedGasOutputs[i].From, account) && derivedGasOutputs[j].Method == 0 { // sub
						totalSend = BigIntReduceStr(totalSend, derivedGasOutputs[i].Value)
						totalSendOut = BigIntReduceStr(totalSendOut, derivedGasOutputs[i].Value)
					} else if strings.EqualFold(derivedGasOutputs[i].To, account) && derivedGasOutputs[j].Method == 0 { // add
						totalSend = BigIntAddStr(totalSend, derivedGasOutputs[i].Value)
						totalSendIn = BigIntAddStr(totalSendIn, derivedGasOutputs[i].Value)
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
	return infos
}

// worker
func (s *AccountingServer) findWorkerInfoByAccountAndBlockNo(account string, realStartHeight int64, realEndHeight int64) []types.WorkerInfo {

	var infos []types.WorkerInfo
	var workerBalance interface{}
	for i := realStartHeight; i < realEndHeight; i++ {
		// 统计fee minertip totalvalue

		iStr := strconv.FormatInt(i, 10)
		var resultMap map[string]interface{}
		result := GetMinerInfoByMinerIdAndHeight(account, iStr)
		if err := json.Unmarshal([]byte(result), &resultMap); err == nil {
			cid := resultMap["result"].(map[string]interface{})["Cids"].([]interface{})[0].(map[string]interface{})["/"]
			cidStr := cid.(string)
			var stateGetActorMinerMap map[string]interface{}
			stateGetActorMinerResult := StateGetActor(account, cidStr)
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
						totalBurnFee = BigIntAddStr(totalBurnFee, derivedGasOutputs[j].BaseFeeBurn)
						totalBurnFee = BigIntAddStr(totalBurnFee, derivedGasOutputs[j].OverEstimationBurn)
						totalMinerTip = BigIntAddStr(totalMinerTip, derivedGasOutputs[j].MinerTip)
					}
					if strings.EqualFold(derivedGasOutputs[j].From, account) && derivedGasOutputs[j].Method == 0 { // sub
						totalSend = BigIntReduceStr(totalSend, derivedGasOutputs[j].Value)
						totalSendOut = BigIntReduceStr(totalSendOut, derivedGasOutputs[j].Value)
					} else if strings.EqualFold(derivedGasOutputs[j].To, account) && derivedGasOutputs[j].Method == 0 { // add
						totalSend = BigIntAddStr(totalSend, derivedGasOutputs[j].Value)
						totalSendIn = BigIntAddStr(totalSendIn, derivedGasOutputs[j].Value)
					}
					if derivedGasOutputs[j].Method == 6 {
						totalPreCommitSectors = BigIntAddStr(totalPreCommitSectors, derivedGasOutputs[j].Value)
					}
					if derivedGasOutputs[j].Method == 7 {
						totalProveCommitSectors = BigIntAddStr(totalProveCommitSectors, derivedGasOutputs[j].Value)
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
	return infos
}

// miner
func (s *AccountingServer) findMinerInfoByAccountAndBlockNo(account string, realStartHeight int64, realEndHeight int64) []types.MinerInfo {
	var infos []types.MinerInfo
	var minerAvailableBalance interface{}
	var preCommitDeposits interface{}
	var lockedFunds interface{}
	var initialPledge interface{}
	var minerBalance interface{}
	for i := realStartHeight; i < realEndHeight; i++ {
		// 统计fee minertip totalvalue
		// table derived_gas_outputs
		iStr := strconv.FormatInt(i, 10)
		var resultMap map[string]interface{}
		result := GetMinerInfoByMinerIdAndHeight(account, iStr)
		if err := json.Unmarshal([]byte(result), &resultMap); err == nil {
			cid := resultMap["result"].(map[string]interface{})["Cids"].([]interface{})[0].(map[string]interface{})["/"]
			cidStr := cid.(string)
			var stateGetActorMinerMap map[string]interface{}
			stateGetActorMinerResult := StateGetActor(account, cidStr)
			if err := json.Unmarshal([]byte(stateGetActorMinerResult), &stateGetActorMinerMap); err == nil {
				minerBalance = stateGetActorMinerMap["result"].(map[string]interface{})["Balance"]
				minerAvailableBalanceResult := StateMinerAvailableBalance(account, cidStr)
				var minerAvailableBalanceMap map[string]interface{}
				if err := json.Unmarshal([]byte(minerAvailableBalanceResult), &minerAvailableBalanceMap); err == nil {
					minerAvailableBalance = minerAvailableBalanceMap["result"]

					stateReadStateResult := StateReadState(account, cidStr)
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
			derivedGasOutputs, _ := s.PostgresClient.QueryDerivedGasOutputs(account, i)
			if len(derivedGasOutputs) > 0 {
				for j := 0; j < len(derivedGasOutputs); j++ {
					// 出账才有手续费
					if strings.EqualFold(derivedGasOutputs[j].From, account) {
						totalBurnFee = BigIntAddStr(totalBurnFee, derivedGasOutputs[j].BaseFeeBurn)
						totalBurnFee = BigIntAddStr(totalBurnFee, derivedGasOutputs[j].OverEstimationBurn)
						totalMinerTip = BigIntAddStr(totalMinerTip, derivedGasOutputs[j].MinerTip)
					}
					if strings.EqualFold(derivedGasOutputs[j].From, account) && derivedGasOutputs[j].Method == 0 { // sub
						totalSend = BigIntReduceStr(totalSend, derivedGasOutputs[j].Value)
						totalSendOut = BigIntReduceStr(totalSendOut, derivedGasOutputs[j].Value)
					} else if strings.EqualFold(derivedGasOutputs[j].To, account) && derivedGasOutputs[j].Method == 0 { // add
						totalSend = BigIntAddStr(totalSend, derivedGasOutputs[j].Value)
						totalSendIn = BigIntAddStr(totalSendIn, derivedGasOutputs[j].Value)
					}
					if derivedGasOutputs[j].Method == 6 {
						totalPreCommitSectors = BigIntAddStr(totalPreCommitSectors, derivedGasOutputs[j].Value)
					}
					if derivedGasOutputs[j].Method == 7 {
						totalProveCommitSectors = BigIntAddStr(totalProveCommitSectors, derivedGasOutputs[j].Value)
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
			info.BlockReward = "0"
			info.MinerAvailableBalance = minerAvailableBalance.(string)
			info.LockedFunds = lockedFunds.(string)
			info.InitialPledge = initialPledge.(string)
			info.BlockRewardToAvailableBalance = "0"
			info.BlockRewardToLockedFunds = "0"
			info.Balance = minerBalance.(string)
			infos = append(infos, info)
			var k = len(infos)
			if k >= 2 {
				subBalance := BigIntReduceStr(infos[k-1].Balance, infos[k-2].Balance)
				totalPreCommitSectors := BigIntAddStr(infos[k-2].PreCommitSectors, infos[k-2].ProveCommitSectors)
				infos[k-2].BlockReward = BigIntReduceStr(subBalance, totalPreCommitSectors)
				blockReward, _ := strconv.ParseInt(infos[k-2].BlockReward, 10, 64)
				if blockReward > 0 {
					infos[k-2].BlockReward = BigIntReduceStr(infos[k-2].BlockReward, infos[k-2].Send)
				}
				// TODO 惩罚
			}

		}
	}
	return infos

}

var httpUrl = "http://106.74.7.3:34569"

//  find filecoin chain tipset
func GetMinerInfoByMinerIdAndHeight(minerId string, height string) string {
	json := `{ "jsonrpc": "2.0", "method":"Filecoin.ChainGetTipSetByHeight", "params": [` + height + `,[]], "id": 1 }`
	reader := bytes.NewReader([]byte(json))
	return funcHttp(httpUrl, reader)
}

//  StateMinerAvailableBalance
func StateMinerAvailableBalance(minerId string, cid string) string {
	json := `{ "jsonrpc": "2.0", "method":"Filecoin.StateMinerAvailableBalance", "params": [` + "\"" + minerId + "\"" + `,[{"/":` + "\"" + cid + "\"" + `}]], "id": 1 }`
	reader := bytes.NewReader([]byte(json))
	return funcHttp(httpUrl, reader)
}

//  StateReadState
func StateReadState(minerId string, cid string) string {
	json := `{ "jsonrpc": "2.0", "method":"Filecoin.StateReadState", "params": [` + "\"" + minerId + "\"" + `,[{"/":` + "\"" + cid + "\"" + `}]], "id": 1 }`
	reader := bytes.NewReader([]byte(json))
	return funcHttp(httpUrl, reader)
}

// StateMinerInfo return worker
func StateMinerInfo(minerId string, cid string) string {
	json := `{ "jsonrpc": "2.0", "method":"Filecoin.StateMinerInfo", "params": [` + "\"" + minerId + "\"" + `,[{"/":` + "\"" + cid + "\"" + `}]], "id": 1 }`
	reader := bytes.NewReader([]byte(json))
	return funcHttp(httpUrl, reader)
}

// StateGetActor  minerId or workerId  or normal id  to find balance at current blockNo
func StateGetActor(minerId string, cid string) string {
	json := `{ "jsonrpc": "2.0", "method":"Filecoin.StateGetActor", "params": [` + "\"" + minerId + "\"" + `,[{"/":` + "\"" + cid + "\"" + `}]], "id": 1 }`
	reader := bytes.NewReader([]byte(json))
	return funcHttp(httpUrl, reader)
}

// StateLookupID  根据高度反查ID
func StateLookupID(account string, cid string) string {
	json := `{ "jsonrpc": "2.0", "method":"Filecoin.StateLookupID", "params": [` + "\"" + account + "\"" + `,[{"/":` + "\"" + cid + "\"" + `}]], "id": 1 }`
	reader := bytes.NewReader([]byte(json))
	return funcHttp(httpUrl, reader)
}

// ChainHead  获取最新高度
func ChainHead() string {
	json := `{ "jsonrpc": "2.0", "method":"Filecoin.ChainHead", "params": [], "id": 1 }`
	reader := bytes.NewReader([]byte(json))
	return funcHttp(httpUrl, reader)
}

func funcHttp(httpUrl string, reader *bytes.Reader) string {
	request, err := http.NewRequest("POST", httpUrl, reader)
	if err != nil {
		fmt.Println(err.Error())
	}
	request.Header.Set("Content-Type", "application/json;charset=UTF-8")
	client := http.Client{}
	resp, err := client.Do(request)
	if err != nil {
		fmt.Println(err.Error())
	}
	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err.Error())
	}
	//byte数组直接转成string
	str := (*string)(unsafe.Pointer(&respBytes))
	return *str
}

// over int64 calculate add
func BigIntAdd(numstr string, num int64) string {
	n, _ := new(big.Int).SetString(numstr, 10)
	m := new(big.Int)
	m.SetInt64(num)
	m.Add(n, m)
	return m.String()
}

// over int64 calculate sub
func BigIntReduce(numstr string, num int64) string {
	n, _ := new(big.Int).SetString(numstr, 10)
	m := new(big.Int)
	m.SetInt64(-num)
	m.Add(n, m)
	return m.String()
}

func BigIntAddStr(numstr string, num string) string {
	n, _ := new(big.Int).SetString(numstr, 10)
	m := new(big.Int)
	m.SetString(num, 10)
	m.Add(n, m)
	return m.String()
}

// over int64 calculate sub
func BigIntReduceStr(numstr string, num string) string {
	n, _ := new(big.Int).SetString(numstr, 10)
	m := new(big.Int)
	m.SetString(num, 10)
	m.Sub(n, m)
	return m.String()
}
