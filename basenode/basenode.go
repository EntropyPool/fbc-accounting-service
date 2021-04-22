package basenode

import (
	"fmt"
	log "github.com/EntropyPool/entropy-logger"
	"golang.org/x/xerrors"
	"io/ioutil"
	"net"
	"net/http"
	"os/exec"
	"time"
)

func getPublicAddr(url string) (string, error) {
	localAddr := ""

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	client := &http.Client{
		Transport: &http.Transport{
			Dial: func(network, addr string) (net.Conn, error) {
				localAddr, err := net.ResolveTCPAddr(network, fmt.Sprintf("%v:0", localAddr))
				if err != nil {
					return nil, err
				}
				remoteAddr, err := net.ResolveTCPAddr(network, addr)
				if err != nil {
					return nil, err
				}
				conn, err := net.DialTCP(network, localAddr, remoteAddr)
				if err != nil {
					return nil, err
				}
				deadline := time.Now().Add(35 * time.Second)
				conn.SetDeadline(deadline)
				return conn, nil
			},
		},
	}

	req.Header.Set("User-Agent",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_8_3) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/27.0.1453.93 Safari/537.36")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	addr := net.ParseIP(string(body))
	if addr == nil {
		return "", xerrors.Errorf("invalid ip address")
	}

	return string(body), nil
}

func GetAddress() (string, string, error) {
	localAddr := ""

	addr, err := exec.Command(
		"dig", "+short", "myip.opendns.com",
		"@resolver1.opendns.com", "-b", localAddr,
	).Output()
	if err == nil {
		//n.hasPublicAddr = true
		return localAddr, string(addr), nil
	}

	log.Errorf(log.Fields{}, "cannot get public address with dig: %v", err)

	publicAddr, err := getPublicAddr("http://inet-ip.info/ip")
	if err == nil {
		//n.hasPublicAddr = true
		return localAddr, publicAddr, err
	}

	log.Errorf(log.Fields{}, "cannot get public address: %v", err)

	publicAddr, err = getPublicAddr("http://ipinfo.io/ip")
	if err == nil {
		//hasPublicAddr = true
		return localAddr, publicAddr, err
	}

	log.Errorf(log.Fields{}, "cannot get public address: %v", err)

	return localAddr, "", err
}
