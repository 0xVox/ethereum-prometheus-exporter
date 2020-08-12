package collector

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/prometheus/client_golang/prometheus"
)

type EthHeightDiff struct {
	rpc  *rpc.Client
	desc *prometheus.Desc
}

type RPC2 struct {
	Jsonrpc string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Result  string `json:"result"`
}

func NewEthHeightDiff(rpc *rpc.Client) *EthHeightDiff {
	return &EthHeightDiff{
		rpc: rpc,
		desc: prometheus.NewDesc(
			"eth_block_height_diff",
			"The difference between current node's block height and external source (Etherscan)",
			nil,
			nil,
		),
	}
}

func (collector *EthHeightDiff) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.desc
}

func (collector *EthHeightDiff) Collect(ch chan<- prometheus.Metric) {
	var our_height hexutil.Uint64
	if err := collector.rpc.Call(&our_height, "eth_blockNumber"); err != nil {
		ch <- prometheus.NewInvalidMetric(collector.desc, err)
		return
	}

	// etherscan RPC2.0 over http (hex)
	var etherscan_height int64
	respo, err := http.Get("https://api.etherscan.io/api?module=proxy&action=eth_blockNumber&apikey=" + os.Getenv("ETHERSCAN_TOKEN"))
	if err != nil || respo.StatusCode != 200 {
		fmt.Println("Error getting etherscan result", err)
	} else {
		defer respo.Body.Close()
		cont, err := ioutil.ReadAll(respo.Body)
		if err != nil {
			fmt.Println("Reading etherscan response body failed")
		}
		stri := string(cont)
		ress := strings.NewReader(stri)
		var ee RPC2
		err = json.NewDecoder(ress).Decode(&ee)
		if err != nil {
			fmt.Println("Failed to decode etherscan json response")
		}
		etherscan_height, err = strconv.ParseInt(ee.Result, 0, 64)
		if err != nil {
			fmt.Println("Failed to parse etherscan hex into integer")
		}
	}

	value := float64(etherscan_height) - float64(our_height)

	ch <- prometheus.MustNewConstMetric(collector.desc, prometheus.GaugeValue, value)
}
