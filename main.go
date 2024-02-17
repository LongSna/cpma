package main

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"sync"
	"sync/atomic"

	"github.com/BurntSushi/toml"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"

	log "github.com/sirupsen/logrus"
)

type Config struct {
	RPC_URL           string
	ContractAddresses []string
	FromBlock         int64
	ToBlock           int64
	Step              uint64
}

type atomicCounter struct {
	number int64
}

func newAtomicCounter() *atomicCounter {
	return &atomicCounter{0}
}

func search_contract_data(client *ethclient.Client, fromBlock *big.Int, toBlock *big.Int, contractAddr common.Address) ([]string, error) {
	query := ethereum.FilterQuery{
		FromBlock: fromBlock,
		ToBlock:   toBlock,
		Addresses: []common.Address{
			contractAddr,
		},
	}

	logs, err := client.FilterLogs(context.Background(), query)
	if err != nil {
		log.Error(fmt.Sprintf("Failed to get blocks data from %s to %s:%s", fromBlock.String(), toBlock.String(), err))
	}

	logTransferSig := []byte("Transfer(address,address,uint256)")
	logERC404TransferSig := []byte("ERC20Transfer(address,address,uint256)")
	logTransferSigHash := crypto.Keccak256Hash(logTransferSig)
	logERC404TransferSigHash := crypto.Keccak256Hash(logERC404TransferSig)

	var ref1 []string
	for _, vLog := range logs {
		switch vLog.Topics[0].Hex() {
		case logTransferSigHash.Hex():
			ref1 = append(ref1, vLog.Topics[2].String())
		case logERC404TransferSigHash.Hex():
			ref1 = append(ref1, vLog.Topics[2].String())
		}
	}

	if len(ref1) != 0 {
		return ref1, nil
	}

	return nil, fmt.Errorf("noMsg")
}

func put_data_into_chan(wg *sync.WaitGroup, ch chan string, atomicCounter *atomicCounter, client *ethclient.Client, fromBlock *big.Int, toBlock *big.Int, contractAddr common.Address) {
	data, err := search_contract_data(client, fromBlock, toBlock, contractAddr)
	defer wg.Done()

	if err != nil {
		atomic.AddInt64(&atomicCounter.number, 1)
		return
	}

	for _, pdata := range data {
		ch <- pdata
	}
	atomic.AddInt64(&atomicCounter.number, 1)
}

func removeDuplicateElement(languages []string) []string {
	result := make([]string, 0, len(languages))
	temp := map[string]struct{}{}
	for _, item := range languages {
		if _, ok := temp[item]; !ok {
			temp[item] = struct{}{}
			result = append(result, item)
		}
	}
	return result
}

func collect_address_data(client *ethclient.Client, step uint64, fromBlock *big.Int, toBlock *big.Int, contractAddr common.Address) []string {

	counter := uint64(0)
	var wg sync.WaitGroup
	ch := make(chan string)
	var task_counter uint64
	task_counter = 0
	atomicCounter := newAtomicCounter()

	for {
		if counter+step+1 > (toBlock.Uint64() - fromBlock.Uint64()) {
			break
		}
		go put_data_into_chan(&wg, ch, atomicCounter, client, big.NewInt(toBlock.Int64()+int64(counter)), big.NewInt(toBlock.Int64()+int64(counter)+int64(step)), contractAddr)
		wg.Add(1)

		counter += step + 1
		task_counter += 1
		log.Info(fmt.Sprintf("START collect Task 「%d」for 「%s」", task_counter, contractAddr))
	}
	go put_data_into_chan(&wg, ch, atomicCounter, client, big.NewInt(fromBlock.Int64()+int64(counter)), toBlock, contractAddr)
	wg.Add(1)
	task_counter += 1
	log.Info(fmt.Sprintf("START collect Task 「%d」for 「%s」", task_counter, contractAddr))

	go func() {
		wg.Wait()
		close(ch)
	}()

	var value []string
	var cache_counter int64

	for datas := range ch {

		atomic_num := atomic.LoadInt64(&atomicCounter.number)
		if atomic_num != cache_counter {
			cache_counter = atomic_num
			log.Info(fmt.Sprintf("Collect Task process:「%d」/「%d」", cache_counter, task_counter))
		}
		if datas == "" {
			continue
		}
		value = append(value, datas)
	}

	return removeDuplicateElement(value)
}
func collect_mul_address_data(client *ethclient.Client, step uint64, fromBlock *big.Int, toBlock *big.Int, contractAddrs []common.Address) [][]string {
	var datas [][]string
	for _, contractAddr := range contractAddrs {
		d := collect_address_data(client, step, fromBlock, toBlock, contractAddr)
		datas = append(datas, d)
	}
	return datas
}

func compare_mul_address_data(client *ethclient.Client, step uint64, fromBlock *big.Int, toBlock *big.Int, contractAddrs []common.Address) []string {
	ref := collect_mul_address_data(client, step, fromBlock, toBlock, contractAddrs)
	for i := 0; i < len(ref)-1; i++ {
		for j := i + 1; j < len(ref); j++ {
			if len(ref[i]) > len(ref[j]) {
				a := ref[j]
				ref[j] = ref[i]
				ref[i] = a
			}
		}
	}

	for {
		log.Info("Start Compare Task")
		if len(ref) <= 1 {
			break
		}
		fmt.Println(ref[0])
		log.Info(fmt.Sprintf("Compare address process: left groups「%d」-minimum's group left length「%d」", len(ref), len(ref[0])))
		temp := map[string]struct{}{}
		for _, item := range ref[1] {
			temp[item] = struct{}{}
		}
		var compared_data []string
		for _, item := range ref[0] {
			if _, ok := temp[item]; ok {
				compared_data = append(compared_data, item)
			}
		}
		ref[1] = compared_data
		ref = ref[1:]
	}
	return ref[0]
}

func main() {
	var config Config
	if _, err := toml.DecodeFile("./config.toml", &config); err != nil {
		log.Panic(err)
	}
	log.SetOutput(os.Stdout)
	log.SetFormatter(&log.TextFormatter{
		ForceQuote:      true,
		TimestampFormat: "2006-01-02 15:04:05",
		FullTimestamp:   true,
	})
	client, _ := ethclient.Dial(config.RPC_URL)
	length := len(config.ContractAddresses)
	contractAddrs := make([]common.Address, length)

	for i := 0; i < length; i++ {
		if !common.IsHexAddress(config.ContractAddresses[i]) {
			log.Panic(fmt.Sprintf("%s is not Hex Address", config.ContractAddresses[i]))
		}
		contractAddrs[i] = common.HexToAddress(config.ContractAddresses[i])

	}

	result := compare_mul_address_data(client, config.Step, big.NewInt(config.FromBlock), big.NewInt(config.ToBlock), contractAddrs)

	log.Info("————————————————Results————————————————")
	for _, addr := range result {
		fmt.Printf("0x%s\n", fmt.Sprint(addr)[26:])
	}
}
