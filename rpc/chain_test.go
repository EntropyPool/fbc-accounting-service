package rpc

import (
	"encoding/json"
	"fmt"
	"strconv"
	"testing"
)

func TestFaulty(t *testing.T) {
	beginHeight := int64(625381)
	oneDayBlock := int64(5880) //一天出块数量
	minerId := "f0134006"
	// 往回推一天
	for i := beginHeight; i >= beginHeight-oneDayBlock; i-- {

		iStr := strconv.FormatInt(i, 10)
		result := GetMinerInfoByMinerIdAndHeight(minerId, iStr)
		var resultMap map[string]interface{}
		if err := json.Unmarshal([]byte(result), &resultMap); err == nil {
			cid := resultMap["result"].(map[string]interface{})["Cids"].([]interface{})[0].(map[string]interface{})["/"]
			cidStr := cid.(string)
			stateMinerSectorResult := StateMinerSectorCount(minerId, cidStr)
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
