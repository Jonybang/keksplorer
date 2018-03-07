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
	r.HandleFunc("/api/blocks/{blockNumber}", blockController).Methods("GET")
	r.HandleFunc("/api/transactions", transactionsController).Methods("GET")
	r.HandleFunc("/api/transactions/{hash}", transactionController).Methods("GET")
	r.HandleFunc("/api/accounts", accountsController).Methods("GET")
	r.HandleFunc("/api/accounts/{address}/transactions",
		accountTransactionsController).Methods("GET")

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
	responseData := make(map[string]interface{})
	latestBlock, err := redisClient.Get("latest_block").Result()

	if err != nil {
		tmpl := template.Must(template.ParseFiles("./public/blocks.html"))
		tmpl.Execute(w, "")

		log.Println("Error while getting latest block: ", err)
		return
	}

	latestBlockNumber, err := strconv.Atoi(latestBlock)

	if err != nil {
		log.Println("Error while conversions latest block to int")
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	blocks := []map[string]string{}
	for i := latestBlockNumber; i >= latestBlockNumber-4; i-- {
		block, err := redisClient.HGetAll(
			fmt.Sprintf("block:%v:detail", i)).Result()

		if err != nil {
			log.Println("Error while getting block #", i)
			continue
		}

		txs, err := redisClient.ZRange(
			fmt.Sprintf("block:%v:tx_list", block["number"]), int64(0), int64(-1)).
			Result()

		if err != nil {
			log.Println("Error while getting tx count for block:", block)
			continue
		}

		shortcutBlockInfo := make(map[string]string)

		shortcutBlockInfo["number"] = block["number"]
		shortcutBlockInfo["txs"] = strconv.Itoa(len(txs))

		blocks = append(blocks, shortcutBlockInfo)
	}

	responseData["blocks"] = blocks

	txs, err := redisClient.ZRange("transactions:order", int64(0), int64(-1)).
		Result()

	if err != nil {
		log.Println("Error while getting transactions list:", err)
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	for i := len(txs)/2 - 1; i >= 0; i-- {
		opp := len(txs) - 1 - i
		txs[i], txs[opp] = txs[opp], txs[i]
	}

	responseData["txs"] = txs[:5]

	tmpl := template.Must(template.ParseFiles("./public/index.html"))

	tmpl.Execute(w, responseData)
}

func latestBlockViewController(w http.ResponseWriter, r *http.Request) {
	latestBlockNumber, err := redisClient.Get("latest_block").Result()
	responseData := make(map[string]interface{})

	if err != nil {
		tmpl := template.Must(template.ParseFiles("./public/latest_block.html"))
		tmpl.Execute(w, responseData)

		log.Println("Error while getting latest block: ", err)
		return
	}

	latestBlock, err := redisClient.HGetAll(
		fmt.Sprintf("block:%v:detail", latestBlockNumber)).Result()

	if err != nil {
		log.Println("Error while getting latest block: ", err)
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	txs, err := redisClient.ZRange(
		fmt.Sprintf("block:%v:tx_list", latestBlock["number"]),
		int64(0), int64(-1)).Result()

	if err != nil {
		log.Println("Error while getting transactions for block:", latestBlock)
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	transactions := []map[string]string{}
	for _, tx := range txs {
		txDetail, err := redisClient.HGetAll(
			fmt.Sprintf("block_tx:%v:%v:detail", latestBlock["number"], tx)).Result()

		if err != nil {
			log.Println("Error while getting transactions detail with hash:", tx)
			continue
		}

		transactions = append(transactions, txDetail)
	}

	responseData["block"] = latestBlock
	responseData["txs"] = transactions

	tmpl := template.Must(template.ParseFiles("./public/latest_block.html"))

	tmpl.Execute(w, responseData)
}

func blocksViewController(w http.ResponseWriter, r *http.Request) {
	latestBlock, err := redisClient.Get("latest_block").Result()

	if err != nil {
		tmpl := template.Must(template.ParseFiles("./public/blocks.html"))
		tmpl.Execute(w, "")

		log.Println("Error while getting latest block: ", err)
		return
	}

	latestBlockNumber, err := strconv.Atoi(latestBlock)

	if err != nil {
		log.Println("Error while conversions latest block to int")
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	blocks := []map[string]string{}
	for i := latestBlockNumber; i >= latestBlockNumber-100; i-- {
		block, err := redisClient.HGetAll(
			fmt.Sprintf("block:%v:detail", i)).Result()

		if err != nil {
			log.Println("Error while getting block #", i)
			continue
		}

		txs, err := redisClient.ZRange(
			fmt.Sprintf("block:%v:tx_list", block["number"]), int64(0), int64(-1)).
			Result()

		if err != nil {
			log.Println("Error while getting tx count for block:", block)
			continue
		}

		shortcutBlockInfo := make(map[string]string)

		shortcutBlockInfo["number"] = block["number"]
		shortcutBlockInfo["txs"] = strconv.Itoa(len(txs))

		blocks = append(blocks, shortcutBlockInfo)
	}

	tmpl := template.Must(template.ParseFiles("./public/blocks.html"))

	tmpl.Execute(w, blocks)
}

func blockViewController(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	responseData := make(map[string]interface{})

	block, err := redisClient.HGetAll(
		fmt.Sprintf("block:%v:detail", vars["blockNumber"])).Result()

	if err != nil {
		log.Println("Error while getting block #", vars["blockNumber"], err)
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	txs, err := redisClient.ZRange(
		fmt.Sprintf("block:%v:tx_list", block["number"]), int64(0), int64(-1)).
		Result()

	if err != nil {
		log.Println("Error while getting transactions for block:", block)
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	transactions := []map[string]string{}
	for _, tx := range txs {
		txDetail, err := redisClient.HGetAll(
			fmt.Sprintf("block_tx:%v:%v:detail", vars["blockNumber"], tx)).Result()

		if err != nil {
			log.Println("Error while getting transactions detail with hash:", tx)
			continue
		}

		transactions = append(transactions, txDetail)
	}

	responseData["block"] = block
	responseData["txs"] = transactions

	tmpl := template.Must(template.ParseFiles("./public/block.html"))

	tmpl.Execute(w, responseData)
}

func transactionsViewController(w http.ResponseWriter, r *http.Request) {
	keys, err := redisClient.Keys("block:*:tx_list").Result()

	if err != nil {
		log.Println("Error while getting transactions list: ", err)
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	transactions := []string{}
	for _, key := range keys {
		tx, err := redisClient.ZRange(key, int64(0), int64(-1)).Result()

		if err != nil {
			log.Println("Error while getting transaction list: ", err)
			continue
		}

		transactions = append(transactions, tx...)
	}

	tmpl := template.Must(template.ParseFiles("./public/transactions.html"))

	tmpl.Execute(w, transactions)
}

func transactionViewController(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	txKey, err := redisClient.Keys(
		fmt.Sprintf("block_tx:*:%v:detail", vars["hash"])).Result()

	if err != nil {
		log.Println("Error while getting transaction detail with hash: ",
			vars["hash"], err)
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	if len(txKey) == 0 {
		err := errors.New("Wrong hash, can't find any transaction with hash: " +
			vars["hash"])

		log.Println(err)

		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	txDetail, err := redisClient.HGetAll(txKey[0]).Result()

	if err != nil {
		log.Println("Error while getting transaction detail with hash: ",
			vars["hash"], err)
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	tmpl := template.Must(template.ParseFiles("./public/transaction.html"))

	tmpl.Execute(w, txDetail)
}

func accountsViewController(w http.ResponseWriter, r *http.Request) {
	accounts, err := redisClient.ZRange("account:order", int64(0), int64(-1)).
		Result()

	if err != nil {
		log.Println("Error while getting account list:", err)
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	tmpl := template.Must(template.ParseFiles("./public/accounts.html"))

	tmpl.Execute(w, accounts)
}

func accountViewController(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	accountTxs, err := redisClient.ZRange(
		fmt.Sprintf("account:%v:tx_list", vars["address"]), int64(0), int64(-1)).
		Result()

	if err != nil {
		log.Println("Error while getting transactions list of account:",
			vars["address"], err)

		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	responseData := make(map[string]interface{})

	responseData["address"] = vars["address"]
	responseData["txs"] = accountTxs

	tmpl := template.Must(template.ParseFiles("./public/account.html"))

	tmpl.Execute(w, responseData)
}

// API

func latestBlockController(w http.ResponseWriter, r *http.Request) {
	blocks, err := redisClient.Keys("block:*:detail").Result()

	if err != nil {
		log.Println("Error while getting latest block: ", err)
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	latestBlock, err := redisClient.HGetAll(
		fmt.Sprintf("block:%v:detail", len(blocks)-1)).Result()

	if err != nil {
		log.Println("Error while getting latest block: ", err)
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

func blockController(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	block, err := redisClient.HGetAll(
		fmt.Sprintf("block:%v:detail", vars["blockNumber"])).Result()

	log.Println(fmt.Sprintf("Get block: %v", block))

	if err != nil {
		log.Println("Error while getting block #", vars["blockNumber"], err)
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
	keys, err := redisClient.Keys("block:*:tx_list").Result()

	if err != nil {
		log.Println("Error while getting transactions list: ", err)
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	transactions := []string{}
	for _, key := range keys {
		tx, err := redisClient.ZRange(key, int64(0), int64(-1)).Result()

		if err != nil {
			log.Println("Error while getting transaction list: ", err)
			continue
		}

		transactions = append(transactions, tx...)
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

	txKey, err := redisClient.Keys(
		fmt.Sprintf("block_tx:*:%v:detail", vars["hash"])).Result()

	if err != nil {
		log.Println("Error while getting transaction detail with hash: ",
			vars["hash"], err)
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	if len(txKey) == 0 {
		err := errors.New("Wrong hash, can't find any transaction with hash: " +
			vars["hash"])

		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	txDetail, err := redisClient.HGetAll(txKey[0]).Result()

	if err != nil {
		log.Println("Error while getting transaction detail with hash: ",
			vars["hash"], err)
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

// TODO: duplicates of accounts. Fix later
func accountsController(w http.ResponseWriter, r *http.Request) {
	keys, err := redisClient.Keys("block_tx:*:*:list").Result()

	if err != nil {
		log.Println("Error while getting accounts list: ", err)
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	accounts := []string{}
	for _, key := range keys {
		account, err := redisClient.ZRange(key, int64(0), int64(-1)).Result()
		log.Println(account)

		if err != nil {
			log.Println("Error while getting account: ", key, err)
		}

		accounts = append(accounts, account...)
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

// TODO: account details is empty for now
func accountController(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	account, err := redisClient.HGetAll(
		fmt.Sprintf("account:%v:detail", vars["address"])).Result()

	if err != nil {
		log.Println("Error while getting account by hash: ", vars["address"], err)
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

func accountTransactionsController(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	tx, err := redisClient.ZRange(
		fmt.Sprintf("account:%v:tx_list", vars["address"]), int64(0), int64(-1)).
		Result()

	if err != nil {
		log.Println("Error while getting account transactions: ", vars["address"],
			err)
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	jsonString, err := json.Marshal(tx)

	if err != nil {
		log.Println("Error while marshalling account transactions: ", err)
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

func getLatestBlock() {

}
