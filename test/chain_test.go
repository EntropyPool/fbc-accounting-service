package test

import (
	"encoding/json"
	"fmt"
	accounting "github.com/EntropyPool/fbc-accounting-service/accounting"
	"github.com/EntropyPool/fbc-accounting-service/rpc"
	"github.com/EntropyPool/fbc-accounting-service/utils"
	"strconv"
	"testing"
)

func TestFaulty(t *testing.T) {
	beginHeight := int64(625381)
	oneDayBlock := int64(10880) //一天出块数量
	minerId := "f0134006"
	// 往回推一天
	for i := beginHeight; i >= beginHeight-oneDayBlock; i-- {

		iStr := strconv.FormatInt(i, 10)
		result := rpc.GetMinerInfoByMinerIdAndHeight(minerId, iStr)
		var resultMap map[string]interface{}
		if err := json.Unmarshal([]byte(result), &resultMap); err == nil {
			cid := resultMap["result"].(map[string]interface{})["Cids"].([]interface{})[0].(map[string]interface{})["/"]
			cidStr := cid.(string)
			stateMinerSectorResult := rpc.StateMinerSectorCount(minerId, cidStr)
			var stateMinerSectorMap map[string]interface{}
			if err := json.Unmarshal([]byte(stateMinerSectorResult), &stateMinerSectorMap); err == nil {
				Faulty := stateMinerSectorMap["result"].(map[string]interface{})["Faulty"]
				ff := Faulty.(float64)
				if ff != 0 {
					fmt.Println("-----------blockNo:", i, "Faulty", Faulty)
				} else {
					fmt.Println("blockNo:", i, "Faulty", Faulty)
				}
			}

		}

	}

}

func TestPreCommitSector(t *testing.T) {
	server := accounting.NewAccountingServer("../fbc-accounting-service.conf")
	minerPreCommitInfo, _ := server.PostgresClient.QueryCalculaDerivedGasOutputs("f0134006", 624646)
	fmt.Println("0;", minerPreCommitInfo.TotalSendIn)
	endHeight := int64(624381)
	oneDayBlock := int64(2880) //一天出块数量
	subBlock := int64(900)     //一天+900内proveCommit
	beginHeight := endHeight - oneDayBlock - subBlock
	minerId := "f0134006"
	var totalPreCommit = "0"

	for i := beginHeight; i < endHeight; i++ {
		minerPreCommitInfos, _ := server.PostgresClient.QueryMinerPreCommitInfoAndBlockNo(minerId, i)
		if len(minerPreCommitInfos) > 0 {
			w := len(minerPreCommitInfos)
			for j := 0; j < w; j++ {
				sectorId := minerPreCommitInfos[j].SectorId
				// miner_sector_events
				event := "COMMIT_CAPACITY_ADDED"
				minerSectorEvents, _ := server.PostgresClient.QueryMinerSectorEventsAndEvent(minerId, sectorId, event)
				if minerSectorEvents != nil {
					fmt.Println("blockNO:", i, "sector_id:", sectorId, "minerId:", minerId)
				} else {
					fmt.Println("------blockNO------:", i, "sector_id:", sectorId, "minerId:", minerId)
					totalPreCommit = utils.BigIntAddStr(totalPreCommit, minerPreCommitInfos[j].PreCommitDeposit)
				}
			}

		}
	}
	fmt.Println("totalPreCommitDeposit", totalPreCommit)
}
