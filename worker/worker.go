package main

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
	"github.com/go-redis/redis"
	web3 "github.com/goodstemy/w3"
	"github.com/goodstemy/w3/complex/types"
	"github.com/goodstemy/w3/providers"
	"regexp"
	//abi "github.com/ethereum/go-ethereum/accounts/abi"
)

const version = "0.1.0"

var (
	redisClient   *redis.Client
	w3            *web3.Web3
	blocks        chan string
	redisURL      = os.Getenv("REDIS_URL")
	jsonRPCAPIURL = os.Getenv("JSON_RPC_API_URL")
)

func main() {
	numcpu := runtime.NumCPU()
	runtime.GOMAXPROCS(numcpu)
	blocks = make(chan string)

	urlOpts, err := redis.ParseURL(redisURL)

	if err != nil {
		log.Println("Error while getting $REDIS_URL:", err)
	}

	redisClient = redis.NewClient(&redis.Options{
		Addr:         urlOpts.Addr,
		Password:     "",
		DB:           0,
		ReadTimeout:  time.Second * 10,
		WriteTimeout: time.Second,
	})

	w3 = web3.NewWeb3(providers.NewHTTPProvider(
		strings.TrimPrefix(jsonRPCAPIURL, "http://"), 10, false))

	checkConnections()

	for i := 0; i < 10; i++ {
		go worker(blocks)
	}

	for {
		parse()
	}
}

func worker(blocks <-chan string) {
	for b := range blocks {
		parseBlock(b)
	}
}

func checkConnections() {
	_, err := redisClient.Ping().Result()

	if err != nil {
		panic(err)
	}

	_, err = w3.Eth.GetProtocolVersion()

	if err != nil {
		panic(err)
	}
}

func parse() {
	l, err := redisClient.ZCount("queue:blocks", "-inf", "+inf").Result()

	if err != nil {
		log.Printf("Error while getting zcount of queue:blocks. Error: %v\n", err)
		return
	}

	queue, err := redisClient.ZRangeByScore("queue:blocks",
		redis.ZRangeBy{
			Min:    "0",
			Max:    "0",
			Offset: int64(0),
			Count:  int64(l),
		}).Result()

	log.Printf("Zrange get %v blocks\n", len(queue))

	if err != nil {
		log.Printf("Error while getting queue:blocks. %v\n", err)
		return
	}

	var lastBlock int

	if len(queue) > 1000 {
		lastBlock = 1000
	} else {
		lastBlock = len(queue)
	}

	for _, block := range queue[:lastBlock] {
		blocks <- block
	}
}

func parseBlock(blockRaw string) {
	blockNumber, err := strconv.Atoi(blockRaw)

	if err != nil {
		log.Printf("Error while conversion %v to int. Continue...\n", blockRaw)
		return
	}

	block, err := w3.Eth.GetBlockByNumber(
		types.ComplexIntParameter(blockNumber), true)

	if err != nil {
		log.Printf("Error while getting block by number %v. Error: %v\n", blockNumber,
			err)
		return
	}

	redisClient.Pipelined(func(pipe redis.Pipeliner) error {
		for _, tx := range block.Transactions {
			_, err := pipe.ZAdd("transactions:order", redis.Z{
				Score:  float64(block.Timestamp.ToInt64()),
				Member: tx.Hash,
			}).Result()

			if err != nil {
				log.Printf("Error while adding tx:%v to transactions:order. %v\n", tx, err)
				return err
			}
		}

		addr := block.Author

		if block.Author == "" {
			addr = block.Miner
		}

		blockToStore := map[string]interface{}{
			"number":    strconv.FormatInt(block.Number.ToInt64(), 10),
			"validator": addr,
			"gasUsed":   strconv.FormatInt(block.GasUsed.ToInt64(), 10),
			"size":      strconv.FormatInt(block.Size.ToInt64(), 10),
			"timestamp": strconv.FormatInt(block.Timestamp.ToInt64(), 10),
			"hash":      block.Hash,
		}

		_, err = pipe.HMSet(
			fmt.Sprintf("block:%v:detail",
				strconv.FormatInt(block.Number.ToInt64(), 10)), blockToStore).Result()

		if err != nil {
			log.Printf("Error while HMSet block %v. Error: %v\n", blockToStore, err)
			return err
		}

		for _, tx := range block.Transactions {
			err := parseTransaction(pipe, tx.Hash)

			if err != nil {
				return err
			}
		}

		_, err = pipe.ZAdd("queue:blocks", redis.Z{
			Score:  float64(1),
			Member: strconv.FormatInt(block.Number.ToInt64(), 10),
		}).Result()

		if err != nil {
			log.Printf("Error while set score to 1 in queue:blocks on block %v\n",
				strconv.FormatInt(block.Number.ToInt64(), 10))
			return err
		}

		return nil
	})
}

func parseTransaction(pipe redis.Pipeliner, txHash string) error {
	tx, err := w3.Eth.GetTransactionByHash(txHash)

	if err != nil {
		log.Printf("Error while getting tx with hash %v. Error %v\n", txHash, err)
		return err
	}

	txToStore := map[string]interface{}{
		"from":        tx.From,
		"to":          tx.To,
		"value":       strconv.FormatInt(tx.Value.ToInt64(), 10),
		"gas":         strconv.FormatInt(tx.Gas.ToInt64(), 10),
		"gasPrice":    strconv.FormatInt(tx.GasPrice.ToInt64(), 10),
		"hash":        tx.Hash,
		"input":       tx.Input,
		"nonce":       tx.Nonce,
		"blockNumber": tx.BlockNumber,
	}

	_, err = pipe.HMSet(
		fmt.Sprintf("block_tx:%v:%v:detail", tx.BlockNumber, tx.Hash), txToStore).
		Result()

	if err != nil {
		log.Printf("Error while HMSet tx %v. Error: %v\n", txToStore, err)
		return err
	}

	_, err = pipe.ZAdd(fmt.Sprintf("block_tx:%v:%v:list", tx.BlockNumber, tx.Hash),
		redis.Z{
			Score:  0,
			Member: tx.From,
		}).Result()

	if err != nil {
		log.Printf("Error while ZAdd block_tx:list with hash %v. Error: %v\n",
			tx.Hash, err)
		return err
	}

	if tx.To == "" {
		//TODO: handle contract deployment
	} else {

		//TODO: find method in database by method prefix in transaction input
		if strings.HasPrefix(tx.Input, "0xa9059cbb") {
			//0xa9059cbb means transfer tokens method
			runes := []rune(tx.Input)
			//method := string(runes[0:10])
			re := regexp.MustCompile("^0+")
			to_address := "0x" + re.ReplaceAllLiteralString(string(runes[11:74]), "")
			value, _ := strconv.ParseUint("0x" + string(runes[75:138]), 0, 64)

			log.Printf("Token transaction in count %v to address %v\nContract: %v\nHash: %v\n",
				value, to_address, tx.To, tx.Hash)
			/*
			Output example:
			Token transaction in count 1000 to address 0xe47494379c1d48ee73454c251a6395fdd4f9eb43
			Contract: 0xadfe00d92e5a16e773891f59780e6e54f40b532e
			Hash: 0xbe5d15e63e151142a525812cc02a08f8e6489af1b7720c54378d75220fe0422b
			 */
		} else if tx.Input != "" {
			//WRONG WAY: there is no way to get abiJsonString from w3.Eth.GetCode
			//log.Printf("Unexpected input in transaction:\n%v\nTx hash: %v\nContract: %v\n",
			//	tx.Input, tx.Hash, tx.To)
			//contract_code, err := w3.Eth.GetCode(tx.To)
			//TODO: get abiJsonString from database by contract address
			//myAbi, err := abi.JSON(strings.NewReader(abiJsonString))
			//if err != nil {
			//	log.Fatal(err)
			//}
			//var ifc map[string]interface{}
			//err = myAbi.Unpack(&ifc, "someMethod", tx.Input)
			//if err != nil {
			//	log.Fatal(err)
			//}
		}
		_, err = pipe.ZAdd(fmt.Sprintf("block_tx:%v:%v:list", tx.BlockNumber, tx.Hash),
			redis.Z{
				Score:  1,
				Member: tx.To,
			}).Result()

		if err != nil {
			log.Printf("Error while ZAdd block_tx:list with hash %v. Error: %v\n",
				tx.Hash, err)
			return err
		}

		err = associateAccountWithTx(pipe, tx.To, tx.Hash, tx.BlockNumber)

		if err != nil {
			return err
		}

		err = addAccountOrder(pipe, tx.To, tx.BlockNumber)

		if err != nil {
			return err
		}
	}

	err = associateAccountWithTx(pipe, tx.From, tx.Hash, tx.BlockNumber)

	if err != nil {
		return err
	}

	err = addAccountOrder(pipe, tx.From, tx.BlockNumber)

	if err != nil {
		return err
	}

	return nil
}

func associateAccountWithTx(pipe redis.Pipeliner, accountAddress, txHash string,
	blockNumber int64) error {
	_, err := pipe.ZAdd(fmt.Sprintf("account:%v:tx_list", accountAddress),
		redis.Z{
			Score:  float64(blockNumber),
			Member: txHash,
		}).Result()

	if err != nil {
		log.Printf("Error while ZAdd block_tx:account with hash %v. Error: %v\n",
			txHash, err)
		return err
	}

	return nil
}

func addAccountOrder(pipe redis.Pipeliner, accountAddress string,
	blockNumber int64) error {
	_, err := pipe.ZAdd("account:order",
		redis.Z{
			Score:  float64(blockNumber),
			Member: accountAddress,
		}).Result()

	if err != nil {
		log.Printf("Error while ZAdd account:order with address %v. Error: %v\n",
			accountAddress, err)
		return err
	}

	return nil
}
