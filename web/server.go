package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis"
	"github.com/gorilla/mux"
)

const version = "0.1.0"

var redisClient *redis.Client

var transactionsScript []byte
var blocksScript []byte
var accountScript []byte

func main() {
	err := loadScripts()

	if err != nil {
		log.Println(err)
		return
	}

	r := mux.NewRouter()

	urlOpts, err := redis.ParseURL(os.Getenv("REDIS_URL"))

	if err != nil {
		log.Println("Error while getting $REDIS_URL: ", err)
	}

	redisClient = redis.NewClient(&redis.Options{
		Addr:        urlOpts.Addr,
		Password:    "",
		DB:          0,
		ReadTimeout: time.Second * 10,
	})

	// VIEW
	r.HandleFunc("/", mainViewController).Methods("GET")
	r.HandleFunc("/blocks", blocksViewController).Methods("GET")
	r.HandleFunc("/blocks/{blockNumber}", blockViewController).Methods("GET")
	r.HandleFunc("/transactions", transactionsViewController).Methods("GET")
	r.HandleFunc("/transactions/{hash}", transactionViewController).Methods("GET")
	r.HandleFunc("/accounts", accountsViewController).Methods("GET")
	r.HandleFunc("/accounts/{address}", accountViewController).Methods("GET")

	// API
	r.HandleFunc("/api/latest_block", latestBlockController).Methods("GET")
	r.HandleFunc("/api/blocks", blocksController).Methods("GET")
	r.HandleFunc("/api/blocks/{blockNumber}", blockController).Methods("GET")
	r.HandleFunc("/api/transactions", transactionsController).Methods("GET")
	r.HandleFunc("/api/transactions/{hash}", transactionController).Methods("GET")
	r.HandleFunc("/api/accounts", accountsController).Methods("GET")
	r.HandleFunc("/api/accounts/{address}",
		accountController).Methods("GET")

	// STATIC
	r.PathPrefix("/assets").
		Handler(http.StripPrefix("/assets", http.FileServer(
			http.Dir("./public/assets/"))))

	srv := &http.Server{
		Handler:      r,
		Addr:         "0.0.0.0:8080",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  20 * time.Second,
	}

	log.Fatal(srv.ListenAndServe())
}

// VIEW

func mainViewController(w http.ResponseWriter, r *http.Request) {
	response := make(map[string]interface{})
	tmpl := template.Must(template.ParseFiles("./public/index.tmpl",
		"./public/templates/header.tmpl"))

	blocks, err := getBlocksData(5)

	if len(blocks) == 0 && err == nil {
		countBlocks, err := redisClient.Keys("block:*:detail").Result()
		if err != nil {
			tmpl.ExecuteTemplate(w, "main", "")

			log.Println("Error while getting count of blocks: ", err)
			return
		}

		latestBlock, err := redisClient.Get("latest_block").Result()
		if err != nil {
			tmpl.ExecuteTemplate(w, "main", "")

			log.Println("Error while getting latest block: ", err)
			return
		}

		latestBlockNumber, err := strconv.Atoi(latestBlock)

		if err != nil {
			tmpl.ExecuteTemplate(w, "main", "")

			log.Println("Error while converting: ", err)
			return
		}

		progress := make(map[string]interface{})

		percentLoaded := float32(len(countBlocks)) /
			float32(latestBlockNumber) * float32(100)

		progress["percentLoaded"] = int(percentLoaded)

		tmpl.ExecuteTemplate(w, "main", progress)
		return
	}

	if err != nil {
		tmpl.ExecuteTemplate(w, "main", "")

		log.Println("Error while getting latest blocks: ", err)
		return
	}

	txs, err := getTransactionsData(5)

	if err != nil {
		log.Println(err)
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	response["blocks"] = blocks
	response["txs"] = txs

	tmpl.ExecuteTemplate(w, "main", response)
}

func blocksViewController(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("./public/blocks.tmpl",
		"./public/templates/header.tmpl"))
	blocks, err := getBlocksData(50)

	log.Println(blocks)
	if len(blocks) == 0 {
		tmpl.ExecuteTemplate(w, "blocks", "")
		return
	}

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	tmpl.ExecuteTemplate(w, "blocks", blocks)
}

func blockViewController(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("./public/block.tmpl",
		"./public/templates/header.tmpl"))
	vars := mux.Vars(r)

	block, err := getBlockData(vars)

	if err != nil {
		tmpl.ExecuteTemplate(w, "block", "")
		return
	}

	tmpl.ExecuteTemplate(w, "block", block)
}

func transactionsViewController(w http.ResponseWriter, r *http.Request) {
	transactions, err := getTransactionsData(50)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	tmpl := template.Must(template.ParseFiles("./public/transactions.tmpl",
		"./public/templates/header.tmpl"))

	tmpl.ExecuteTemplate(w, "transactions", transactions)
}

func transactionViewController(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	txDetail, err := getTransactionData(vars)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	tmpl := template.Must(template.ParseFiles("./public/transaction.tmpl",
		"./public/templates/header.tmpl"))

	tmpl.ExecuteTemplate(w, "transaction", txDetail)
}

func accountsViewController(w http.ResponseWriter, r *http.Request) {
	accounts, err := getAccountsData()

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	tmpl := template.Must(template.ParseFiles("./public/accounts.tmpl",
		"./public/templates/header.tmpl"))

	tmpl.ExecuteTemplate(w, "accounts", accounts)
}

func accountViewController(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	account, err := getAccountData(vars)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	tmpl := template.Must(template.ParseFiles("./public/account.tmpl",
		"./public/templates/header.tmpl"))

	tmpl.ExecuteTemplate(w, "account", account)
}

// API

func latestBlockController(w http.ResponseWriter, r *http.Request) {
	latestBlock, err := getLatestBlockData()

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	jsonString, err := json.Marshal(latestBlock)

	if err != nil {
		log.Println("Error while marshalling latest block: ", err)
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, string(jsonString))
}

func blocksController(w http.ResponseWriter, r *http.Request) {
	blocks, err := getBlocksData(50)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	jsonString, err := json.Marshal(blocks)

	if err != nil {
		log.Println("Error while marshalling latest block: ", err)
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, string(jsonString))
}

func blockController(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	block, err := getBlockData(vars)

	if err != nil {
		log.Println("Error while getting block data #", vars["blockNumber"], err)
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	jsonString, err := json.Marshal(block)

	if err != nil {
		log.Println("Error while marshalling block #", vars["blockNumber"], err)
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, string(jsonString))
}

func transactionsController(w http.ResponseWriter, r *http.Request) {
	transactions, err := getTransactionsData(50)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	jsonString, err := json.Marshal(transactions)

	if err != nil {
		log.Println("Error while marshalling transactions: ", err)
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, string(jsonString))
}

func transactionController(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	txDetail, err := getTransactionData(vars)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	jsonString, err := json.Marshal(txDetail)

	if err != nil {
		log.Println("Error while marshalling transaction detail: ", err)
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, string(jsonString))
}

func accountsController(w http.ResponseWriter, r *http.Request) {
	accounts, err := getAccountsData()

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	jsonString, err := json.Marshal(accounts)

	if err != nil {
		log.Println("Error while marshalling accounts list: ", err)
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, string(jsonString))
}

func accountController(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	account, err := getAccountData(vars)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	jsonString, err := json.Marshal(account)

	if err != nil {
		log.Println("Error while marshalling account detail: ", err)
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, string(jsonString))
}

// HELPERS

func respondWithError(w http.ResponseWriter, httpStatus int, err error) {
	w.WriteHeader(httpStatus)
	fmt.Fprintf(w, err.Error())
}

func getLatestBlockNumber() (int, error) {
	latestBlock, err := redisClient.Get("latest_block").Result()

	if err != nil {
		return 0, fmt.Errorf("Error while getting latest block: %v", err)
	}

	latestBlockNumber, err := strconv.Atoi(latestBlock)

	if err != nil {
		return 0, fmt.Errorf("Error while getting latest block: %v", err)
	}

	return latestBlockNumber, nil
}

func getBlockTransactions(blockNumber int) ([]string, error) {
	txs, err := redisClient.ZRange(
		fmt.Sprintf("block:%v:tx_list", blockNumber), int64(0), int64(-1)).
		Result()

	if err != nil {
		return []string{}, fmt.Errorf(
			"Error while getting transactions for block: %v, %v", blockNumber, err)
	}

	return txs, nil
}

func getBlockByTransaction(tx string) (map[string]string, error) {
	keys, err := redisClient.Keys(fmt.Sprintf("block_tx:*:%v:detail", tx)).Result()

	if err != nil {
		return map[string]string{}, fmt.Errorf(
			"Error while getting keys by hash %v. Error: %v", tx, err)
	}

	if len(keys) == 0 {
		return map[string]string{}, fmt.Errorf(
			"Can't find any keys with txid %v", tx)
	}

	blockNumber := strings.Split(keys[0], ":")[1]

	block, err := redisClient.HGetAll(fmt.Sprintf(
		"block:%v:detail", blockNumber)).Result()

	if err != nil {
		return map[string]string{}, fmt.Errorf(
			"Error while getting block by key %v. Error: %v", keys[0], err)
	}

	return block, nil
}

func getBlockDetailWithTxsByNumber(blockNumber int) (
	map[string]interface{}, error) {
	responseData := make(map[string]interface{})

	block, err := redisClient.HGetAll(
		fmt.Sprintf("block:%v:detail", blockNumber)).Result()

	if err != nil {
		log.Println("Error while getting block #", blockNumber, err)
		return map[string]interface{}{}, err
	}

	txs, err := redisClient.ZRange(
		fmt.Sprintf("block:%v:tx_list", blockNumber), int64(0), int64(-1)).
		Result()

	if err != nil {
		return map[string]interface{}{}, err
	}

	transactions := []map[string]string{}
	for _, tx := range txs {
		txDetail, err := redisClient.HGetAll(
			fmt.Sprintf("block_tx:%v:%v:detail", blockNumber, tx)).Result()

		if err != nil {
			log.Println("Error while getting transactions detail with hash:", tx)
			continue
		}

		transactions = append(transactions, txDetail)
	}

	responseData["block"] = block
	responseData["txs"] = transactions

	return responseData, nil
}

func getLatestBlockData() (Block, error) {
	blocks, err := getBlocksData(1)

	if err != nil {
		return Block{},
			fmt.Errorf("Error while getting latest block: %v", err)
	}

	if len(blocks) != 0 {
		return blocks[0], nil
	}

	return Block{}, nil
}

func getBlocksData(count int) ([]Block, error) {
	cmd := redisClient.Eval(string(blocksScript), []string{strconv.Itoa(count)})
	blocks := []Block{}

	rawBlocks, err := cmd.Result()
	log.Println(rawBlocks, err)
	if err != nil {
		return blocks, fmt.Errorf(
			"Error while getting transactions from script: %v", err)
	}

	if b, ok := rawBlocks.(string); ok {
		if b == "{}" {
			return blocks, nil
		}

		err = json.Unmarshal([]byte(b), &blocks)

		if err != nil {
			return blocks, fmt.Errorf("Error while unmarshal blocks: %v", err)
		}
	}

	return blocks, nil
}

func getBlockData(vars map[string]string) (map[string]interface{}, error) {
	blockNumber, err := strconv.Atoi(vars["blockNumber"])

	if err != nil {
		return map[string]interface{}{}, err
	}

	responseData, err := getBlockDetailWithTxsByNumber(blockNumber)

	if err != nil {
		return map[string]interface{}{}, err
	}

	return responseData, nil
}

func getTransactionsData(count int) (interface{}, error) {
	cmd := redisClient.Eval(string(transactionsScript), []string{
		strconv.Itoa(count)})

	transactions, err := cmd.Result()

	if err != nil {
		return []map[string]string{}, fmt.Errorf(
			"Error while getting transactions from script: %v", err)
	}

	txs := []Transaction{}

	if t, ok := transactions.(string); ok {
		if t == "{}" {
			return txs, nil
		}

		err = json.Unmarshal([]byte(t), &txs)

		if err != nil {
			log.Println(err)
		}
	}

	for i, tx := range txs {
		block, err := getBlockByTransaction(tx.Hash)

		if err != nil {
			log.Println("Error while getting block by txid. ", err)
		}

		txs[i].Timestamp = block["timestamp"]
	}

	return txs, nil
}

func getTransactionData(vars map[string]string) (map[string]string, error) {
	txKey, err := redisClient.Keys(
		fmt.Sprintf("block_tx:*:%v:detail", vars["hash"])).Result()

	if err != nil {
		return map[string]string{}, fmt.Errorf(
			"Error while getting transaction detail with hash: %v. Error %v",
			vars["hash"], err)
	}

	if len(txKey) == 0 {
		err := errors.New("Wrong hash, can't find any transaction with hash: " +
			vars["hash"])

		return map[string]string{}, err
	}

	txDetail, err := redisClient.HGetAll(txKey[0]).Result()

	if err != nil {
		log.Println("Error while getting transaction detail with hash: ",
			vars["hash"], err)
		return map[string]string{}, fmt.Errorf(
			"Error while getting transaction detail with hash: %v. Error %v",
			vars["hash"], err)
	}

	block, err := getBlockByTransaction(vars["hash"])

	if err != nil {
		return map[string]string{}, err
	}

	txDetail["timestamp"] = block["timestamp"]

	return txDetail, err
}

func getAccountsData() ([]string, error) {
	accounts, err := redisClient.ZRange("account:order", int64(0), int64(-1)).
		Result()

	if err != nil {
		return []string{}, fmt.Errorf("Error while getting accoutns: %v", err)
	}

	return accounts, nil
}

func getAccountData(vars map[string]string) (Account, error) {
	cmd := redisClient.Eval(string(accountScript), []string{vars["address"]})
	account := Account{
		Address: vars["address"],
	}

	raw, err := cmd.Result()
	if err != nil {
		return account, fmt.Errorf(
			"Error while getting transactions from script: %v", err)
	}

	if r, ok := raw.(string); ok {
		err = json.Unmarshal([]byte(r), &account)

		if err != nil {
			return account, fmt.Errorf("Error while unmarshal blocks: %v", err)
		}
	}

	fmt.Printf("%+v\n", account)
	return account, nil
}

func loadScripts() error {
	var err error
	transactionsScript, err = ioutil.ReadFile("./scripts/transactions.lua")

	if err != nil {
		return fmt.Errorf("Error while loading transactions.lua: %v", err)
	}

	blocksScript, err = ioutil.ReadFile("./scripts/blocks.lua")

	if err != nil {
		return fmt.Errorf("Error while loading blocks.lua: %v", err)
	}

	accountScript, err = ioutil.ReadFile("./scripts/account.lua")

	if err != nil {
		return fmt.Errorf("Error while loading account.lua: %v", err)
	}

	return nil
}
