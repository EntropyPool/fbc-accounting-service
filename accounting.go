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
	"net/http"
	"strconv"
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

	log.Infof(log.Fields{}, "start http daemon at %v", s.config.Port)
	httpdaemon.Run(s.config.Port)
	return nil
}

func (s *AccountingServer) GetMinerPledgeRequest(w http.ResponseWriter, req *http.Request) (interface{}, string, int) {

	query := req.URL.Query()
	minerId := string(query["minerId"][0])
	fmt.Println(minerId)
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
	cell.Value = "MinerId"
	cell = row.AddCell()
	cell.Value = "height"
	cell = row.AddCell()
	cell.Value = "PreCommitDeposit"
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
	cell.Value = "InitialPledge"
	cell = row.AddCell()
	cell.Value = "ExpectedStoragePledge"
	cell = row.AddCell()
	cell.Value = "minerAvailableBalance"
	cell = row.AddCell()
	cell.Value = "preCommitDeposits"
	cell = row.AddCell()
	cell.Value = "lockedFunds"
	cell = row.AddCell()
	cell.Value = "initialPledge"
	cell = row.AddCell()
	cell.Value = "balance at height"

	for i := realStartHeight; i < realEndHeight; i++ {
		// table derived_gas_outputs
		derivedGasOutputs, _ := s.PostgresClient.QueryDerivedGasOutputs(to, i)
		// table miner_sector_infos
		minerSectorInfos, _ := s.PostgresClient.QueryMinerSectorInfos(minerId, i)
		// table miner_pre_commit_infos
		if derivedGasOutputs != nil && minerSectorInfos != nil {
			minerPreCommitInfo, _ := s.PostgresClient.QueryMinerPreCommitInfoAndSectorId(minerId, minerSectorInfos.SectorId)
			minerInfo.MinerId = minerId
			minerInfo.PreCommitDeposit = minerPreCommitInfo.PreCommitDeposit
			minerInfo.Value = derivedGasOutputs.Value
			minerInfo.BaseFeeBurn = derivedGasOutputs.BaseFeeBurn
			minerInfo.OverEstimationBurn = derivedGasOutputs.OverEstimationBurn
			minerInfo.MinerPenalty = derivedGasOutputs.MinerPenalty
			minerInfo.MinerTip = derivedGasOutputs.MinerTip
			minerInfo.Refund = derivedGasOutputs.Refund
			minerInfo.GasRefund = derivedGasOutputs.GasRefund
			minerInfo.InitialPledge = minerSectorInfos.InitialPledge
			minerInfo.ExpectedStoragePledge = minerSectorInfos.ExpectedStoragePledge
		} else {
			minerInfo.MinerId = minerId
			minerInfo.PreCommitDeposit = 0
			minerInfo.Value = 0
			minerInfo.BaseFeeBurn = 0
			minerInfo.OverEstimationBurn = 0
			minerInfo.MinerPenalty = 0
			minerInfo.MinerTip = 0
			minerInfo.Refund = 0
			minerInfo.GasRefund = 0
			minerInfo.InitialPledge = 0
			minerInfo.ExpectedStoragePledge = 0
		}
		iStr := strconv.FormatInt(i, 10)
		result := GetMinerInfoByMinerIdAndHeight(minerId, iStr)
		var resultMap map[string]interface{}

		var minerAvailableBalance interface{}
		var preCommitDeposits interface{}
		var lockedFunds interface{}
		var initialPledge interface{}
		var balance interface{}
		if err := json.Unmarshal([]byte(result), &resultMap); err == nil {
			cid := resultMap["result"].(map[string]interface{})["Cids"].([]interface{})[0].(map[string]interface{})["/"]
			fmt.Println(cid)
			cidStr := cid.(string)
			minerAvailableBalanceResult := StateMinerAvailableBalance(minerId, cidStr)
			var minerAvailableBalanceMap map[string]interface{}
			if err := json.Unmarshal([]byte(minerAvailableBalanceResult), &minerAvailableBalanceMap); err == nil {
				minerAvailableBalance = minerAvailableBalanceMap["result"]
				fmt.Println(minerAvailableBalance)

				stateReadStateResult := StateReadState(minerId, cidStr)
				var stateReadStateMap map[string]interface{}
				if err := json.Unmarshal([]byte(stateReadStateResult), &stateReadStateMap); err == nil {
					preCommitDeposits = stateReadStateMap["result"].(map[string]interface{})["State"].(map[string]interface{})["PreCommitDeposits"]
					fmt.Println(preCommitDeposits)
					lockedFunds = stateReadStateMap["result"].(map[string]interface{})["State"].(map[string]interface{})["LockedFunds"]
					fmt.Println(lockedFunds)
					initialPledge = stateReadStateMap["result"].(map[string]interface{})["State"].(map[string]interface{})["InitialPledge"]
					fmt.Println(initialPledge)
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
				row = sheet.AddRow()
				cell = row.AddCell()
				cell.Value = minerInfo.MinerId
				cell = row.AddCell()
				cell.Value = strconv.FormatInt(i, 10)
				cell = row.AddCell()
				cell.Value = strconv.FormatInt(minerInfo.PreCommitDeposit, 10)
				cell = row.AddCell()
				cell.Value = strconv.FormatInt(minerInfo.Value, 10)
				cell = row.AddCell()
				cell.Value = strconv.FormatInt(minerInfo.BaseFeeBurn, 10)
				cell = row.AddCell()
				cell.Value = strconv.FormatInt(minerInfo.OverEstimationBurn, 10)
				cell = row.AddCell()
				cell.Value = strconv.Itoa(minerInfo.MinerPenalty)
				cell = row.AddCell()
				cell.Value = strconv.FormatInt(minerInfo.MinerTip, 10)
				cell = row.AddCell()
				cell.Value = strconv.FormatInt(minerInfo.Refund, 10)
				cell = row.AddCell()
				cell.Value = strconv.FormatInt(minerInfo.GasRefund, 10)
				cell = row.AddCell()
				cell.Value = strconv.FormatInt(minerInfo.InitialPledge, 10)
				cell = row.AddCell()
				cell.Value = strconv.FormatInt(minerInfo.ExpectedStoragePledge, 10)
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
				file.Save("File.xlsx")

			}

		} else {
			fmt.Println(err)
		}
	}

	return minerInfo, "", 0
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

// StateGetActor  minerId or workerId
func StateGetActor(minerId string, cid string) string {
	json := `{ "jsonrpc": "2.0", "method":"Filecoin.StateGetActor", "params": [` + "\"" + minerId + "\"" + `,[{"/":` + "\"" + cid + "\"" + `}]], "id": 1 }`
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
