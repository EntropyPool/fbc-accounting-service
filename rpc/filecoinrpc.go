package rpc

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"unsafe"
)

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

// ChainHead  miner power
func StateMinerPower(account string, cid string, flag bool) string {
	json := ""
	if flag {
		json = `{ "jsonrpc": "2.0", "method":"Filecoin.StateMinerPower", "params": [` + "\"" + account + "\"" + `,[{"/":` + "\"" + cid + "\"" + `}]], "id": 1 }`
	} else {
		json = `{ "jsonrpc": "2.0", "method":"Filecoin.StateMinerPower", "params": [` + "\"" + account + "\"" + `,[]], "id": 1 }`
	}
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

//  StateMinerSectorCount 查看矿工在当前高度的Faulty是否掉算力
func StateMinerSectorCount(minerId string, cid string) string {
	json := `{ "jsonrpc": "2.0", "method":"Filecoin.StateMinerSectorCount", "params": [` + "\"" + minerId + "\"" + `,[{"/":` + "\"" + cid + "\"" + `}]], "id": 1 }`
	reader := bytes.NewReader([]byte(json))
	return funcHttp(httpUrl, reader)
}

//  StateSectorPreCommitInfo 查看矿工的某个扇区在当前高度的preCommitDeposit
func StateSectorPreCommitInfo(minerId string, sectorId string, cid string) string {
	json := `{ "jsonrpc": "2.0", "method":"Filecoin.StateSectorPreCommitInfo", "params": [` + "\"" + minerId + "\"," + sectorId + `,[{"/":` + "\"" + cid + "\"" + `}]], "id": 1 }`
	reader := bytes.NewReader([]byte(json))
	return funcHttp(httpUrl, reader)
}
