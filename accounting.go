package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	log "github.com/EntropyPool/entropy-logger"
	fbcpostgres "github.com/EntropyPool/fbc-accounting-service/postgres"
	types "github.com/EntropyPool/fbc-accounting-service/types"
	httpdaemon "github.com/NpoolRD/http-daemon"
	"io/ioutil"
	"net/http"
	"strconv"
	_ "strings"
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

	query := req.URL.Query()
	minerId := string(query["minerId"][0])
	beginHeight := string(query["beginHeight"][0])
	begin, _ := strconv.ParseInt(beginHeight, 10, 32)
	endHeight := string(query["endHeight"][0])
	end, _ := strconv.ParseInt(endHeight, 10, 32)
	startTime := string(query["startTime"][0])

	endTime := string(query["endTime"][0])
	fmt.Println(minerId + "," + beginHeight + "," + endHeight + "," + startTime + "," + endTime)
	// table derived_gas_outputs
	to := minerId
	derivedGasOutputs, _ := s.PostgresClient.QueryDerivedGasOutputs(to, begin, end)

	// table miner_sector_infos
	minerSectorInfos, _ := s.PostgresClient.QueryMinerSectorInfos(minerId, begin, end)

	// table miner_pre_commit_infos
	minerPreCommitInfo, _ := s.PostgresClient.QueryMinerPreCommitInfoAndSectorId(minerId, minerSectorInfos.SectorId)

	minerInfo := types.MinerInfos{}
	minerInfo.MinerId = minerPreCommitInfo.MinerId
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

	result := GetMinerInfoByMinerIdAndHeight(minerId, beginHeight)
	var vrfProof interface{}
	var resultMap map[string]interface{}
	if err := json.Unmarshal([]byte(result), &resultMap); err == nil {
		vrfProof = resultMap["result"].(map[string]interface{})["Blocks"].([]interface{})[0].(map[string]interface{})["Ticket"].(map[string]interface{})["VRFProof"]
	} else {
		fmt.Println(err)
	}
	fmt.Println(vrfProof)
	// TODO lotus rpc getMinerInfo  by height  getReadState byheight

	return minerInfo, "", 0
}

//  find filecoin chain tipset
func GetMinerInfoByMinerIdAndHeight(minerId string, height string) string {
	httpUrl := "http://106.74.7.3:34569"
	json := `{ "jsonrpc": "2.0", "method":"Filecoin.ChainGetTipSetByHeight", "params": [` + height + `,[]], "id": 1 }`
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
