package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-redis/redis"
	"github.com/gorilla/mux"
)

const version = "0.1.0"

var redisClient *redis.Client

func main() {
	r := mux.NewRouter()

	urlOpts, err := redis.ParseURL(os.Getenv("REDIS_URL"))

	if err != nil {
		log.Println("Error while getting $REDIS_URL: ", err)
	}

	redisClient = redis.NewClient(&redis.Options{
		Addr:     urlOpts.Addr,
		Password: "",
		DB:       0,
	})

	// VIEW
	r.HandleFunc("/", mainViewController).Methods("GET")
	r.HandleFunc("/latest_block", latestBlockViewController).Methods("GET")
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

	blocks, err := getLatestBlocksByCount(5)

	if err != nil {
		tmpl := template.Must(template.ParseFiles("./public/index.tmpl",
			"./public/templates/header.tmpl"))
		tmpl.ExecuteTemplate(w, "main", "")

		log.Println("Error while getting latest blocks: ", err)
		return
	}

	shortcutBlocks := []map[string]string{}
	for _, block := range blocks {
		blockNumber, err := strconv.Atoi(block["number"])

		if err != nil {
			log.Println(err)
			continue
		}

		txs, err := getBlockTransactions(blockNumber)

		if err != nil {
			log.Println(err)
			continue
		}

		shortcutBlockInfo := make(map[string]string)

		shortcutBlockInfo["number"] = block["number"]
		shortcutBlockInfo["txs"] = strconv.Itoa(len(txs))

		shortcutBlocks = append(shortcutBlocks, shortcutBlockInfo)
	}

	txs, err := getLatestTransactionsByCount(5)

	if err != nil {
		log.Println(err)
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	transactions := []map[string]string{}
	for _, tx := range txs {
		txKey, err := redisClient.Keys(fmt.Sprintf(
			"block_tx:*:%v:detail", tx)).Result()

		if err != nil {
			log.Println("Error while getting key of transaction:", tx)
			continue
		}

		txDetail, err := redisClient.HGetAll(txKey[0]).Result()

		if err != nil {
			log.Println("Error while getting transactions detail with hash:", tx)
			continue
		}

		transactions = append(transactions, txDetail)
	}

	response["blocks"] = shortcutBlocks
	response["txs"] = transactions

	tmpl := template.Must(template.ParseFiles("./public/index.tmpl",
		"./public/templates/header.tmpl"))

	tmpl.ExecuteTemplate(w, "main", response)
}

func latestBlockViewController(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("./public/latest_block.tmpl",
		"./public/templates/header.tmpl"))
	latestBlock, err := getLatestBlockData()

	if len(latestBlock) == 0 {
		tmpl.ExecuteTemplate(w, "latest_block", "")
		return
	}

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	tmpl.ExecuteTemplate(w, "latest_block", latestBlock)
}

func blocksViewController(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("./public/blocks.tmpl",
		"./public/templates/header.tmpl"))
	blocks, err := getBlocksData()

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
	transactions, err := getTransactionsData()

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
	blocks, err := getBlocksData()

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
	transactions, err := getTransactionsData()

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

func getLatestBlocksByCount(count int) ([]map[string]string, error) {
	latestBlock, err := getLatestBlockNumber()

	if err != nil {
		return []map[string]string{}, err
	}

	blocks := []map[string]string{}

	for i := latestBlock; i > latestBlock-count; i-- {
		block, err := redisClient.HGetAll(
			fmt.Sprintf("block:%v:detail", i)).Result()

		if err != nil {
			log.Println("Error while getting block #", i)
			continue
		}

		blocks = append(blocks, block)
	}

	return blocks, nil
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

func getLatestTransactionsByCount(count int) ([]string, error) {
	txs, err := redisClient.ZRevRangeByScore("transactions:order",
		redis.ZRangeBy{
			Min:    "-inf",
			Max:    "+inf",
			Offset: int64(0),
			Count:  int64(count),
		}).Result()

	if err != nil {
		return []string{}, fmt.Errorf("Error while getting transactions: %v", err)
	}

	return txs, nil
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

func getLatestBlockData() (map[string]interface{}, error) {
	latestBlockNumber, err := getLatestBlockNumber()
	responseData := make(map[string]interface{})

	if err != nil {
		return map[string]interface{}{},
			fmt.Errorf("Error while getting latest block: %v", err)
	}

	latestBlock, err := getLatestBlocksByCount(1)

	if err != nil {
		return map[string]interface{}{},
			fmt.Errorf("Error while getting latest block: %v", err)
	}

	txs, err := getBlockTransactions(latestBlockNumber)

	if err != nil {
		log.Println("Error while getting transactions for block:", latestBlock)
		return map[string]interface{}{},
			fmt.Errorf("Error while getting transactions for block: %v. Error: %v",
				latestBlock, err)
	}

	transactions := []map[string]string{}
	for _, tx := range txs {
		txDetail, err := redisClient.HGetAll(
			fmt.Sprintf("block_tx:%v:%v:detail", latestBlockNumber, tx)).Result()

		if err != nil {
			log.Println("Error while getting transactions detail with hash:", tx)
			continue
		}

		transactions = append(transactions, txDetail)
	}

	responseData["block"] = latestBlock[0]
	responseData["txs"] = transactions

	return responseData, nil
}

func getBlocksData() ([]map[string]string, error) {
	blocks, err := getLatestBlocksByCount(100)

	if err != nil {
		return []map[string]string{}, err
	}

	shortcutBlocks := []map[string]string{}

	for _, block := range blocks {
		blockNumber, err := strconv.Atoi(block["number"])

		if err != nil {
			log.Println(err)
			continue
		}

		txs, err := getBlockTransactions(blockNumber)

		if err != nil {
			log.Println(err)
			continue
		}

		shortcutBlockInfo := make(map[string]string)

		shortcutBlockInfo["number"] = block["number"]
		shortcutBlockInfo["txs"] = strconv.Itoa(len(txs))

		shortcutBlocks = append(shortcutBlocks, shortcutBlockInfo)
	}

	return shortcutBlocks, nil
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

func getTransactionsData() ([]map[string]string, error) {
	txs, err := redisClient.ZRevRangeByScore("transactions:order",
		redis.ZRangeBy{
			Min:    "-inf",
			Max:    "+inf",
			Offset: int64(0),
			Count:  int64(100),
		}).Result()

	if err != nil {
		return []map[string]string{}, fmt.Errorf("Error while getting transactions:order %v",
			err)
	}

	transactions := []map[string]string{}
	for _, tx := range txs {
		txKey, err := redisClient.Keys(fmt.Sprintf(
			"block_tx:*:%v:detail", tx)).Result()

		if err != nil {
			log.Println("Error while getting key of transaction:", tx)
			continue
		}

		txDetail, err := redisClient.HGetAll(txKey[0]).Result()

		log.Println(txDetail)
		if err != nil {
			log.Println("Error while getting transactions detail with hash:", tx)
			continue
		}

		transactions = append(transactions, txDetail)
	}

	return transactions, nil
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

func getAccountData(vars map[string]string) (map[string]interface{}, error) {
	accountTxs, err := redisClient.ZRange(
		fmt.Sprintf("account:%v:tx_list", vars["address"]), int64(0), int64(-1)).
		Result()

	if err != nil {
		return map[string]interface{}{}, fmt.Errorf(
			"Error while getting transactions list of account: %v. Error %v",
			vars["address"], err)
	}

	balance, err := redisClient.HGetAll(
		fmt.Sprintf("account:%v:detail", vars["address"])).Result()

	if err != nil {
		return map[string]interface{}{}, fmt.Errorf(
			"Error while getting balance of address %v. Error %v",
			balance["address"], err)
	}

	responseData := make(map[string]interface{})

	responseData["address"] = vars["address"]
	responseData["balance"] = balance["balance"]
	responseData["txs"] = accountTxs

	return responseData, nil
}
