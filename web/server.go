package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-redis/redis"
	"github.com/gorilla/mux"
)

var redisClient *redis.Client

func main() {
	log.Println("info:", getVersion())

	r := mux.NewRouter()

	redisClient = redis.NewClient(&redis.Options{
		// REDIS_URL should be similar to 127.0.0.1:6379
		Addr:     os.Getenv("REDIS_URL"),
		Password: "",
		DB:       0,
	})

	r.HandleFunc("/api/latest_block", latestBlockController).Methods("GET")
	r.HandleFunc("/api/blocks/{blockNumber}", blockController).Methods("GET")
	r.HandleFunc("/api/transactions", transactionsController).Methods("GET")
	r.HandleFunc("/api/transactions/{hash}", transactionController).Methods("GET")
	r.HandleFunc("/api/accounts", accountsController).Methods("GET")
	r.HandleFunc("/api/accounts/{address}/transactions",
		accountTranscationsController).Methods("GET")

	srv := &http.Server{
		Handler:      r,
		Addr:         "0.0.0.0:8080",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  20 * time.Second,
	}

	log.Fatal(srv.ListenAndServe())
}

func latestBlockController(w http.ResponseWriter, r *http.Request) {
	blocks, err := redisClient.Keys("block:*:detail").Result()

	if err != nil {
		log.Println("Error while getting latest block: ", err)
	}

	latestBlock, err := redisClient.HGetAll(
		fmt.Sprintf("block:%v:detail", len(blocks)-1)).Result()

	if err != nil {
		log.Println("Error while getting latest block: ", err)
	}

	jsonString, err := json.Marshal(latestBlock)

	if err != nil {
		log.Println("Error while marshalling latest block: ", err)
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
	}

	jsonString, err := json.Marshal(block)

	if err != nil {
		log.Println("Error while marshalling block #", vars["blockNumber"], err)
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, string(jsonString))
}

func transactionsController(w http.ResponseWriter, r *http.Request) {
	keys, err := redisClient.Keys("block:*:tx_list").Result()

	if err != nil {
		log.Println("Error while getting transactions list: ", err)
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
	}

	txDetail, err := redisClient.HGetAll(txKey[0]).Result()

	if err != nil {
		log.Println("Error while getting transaction detail with hash: ",
			vars["hash"], err)
	}

	jsonString, err := json.Marshal(txDetail)

	if err != nil {
		log.Println("Error while marshalling transaction detail: ", err)
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, string(jsonString))
}

// TODO: duplicates of accounts. Fix later
func accountsController(w http.ResponseWriter, r *http.Request) {
	keys, err := redisClient.Keys("block_tx:*:*:list").Result()

	if err != nil {
		log.Println("Error while getting accounts list: ", err)
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
	}

	jsonString, err := json.Marshal(account)

	if err != nil {
		log.Println("Error while marshalling account detail: ", err)
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, string(jsonString))
}

func accountTranscationsController(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	tx, err := redisClient.ZRange(
		fmt.Sprintf("account:%v:tx_list", vars["address"]), int64(0), int64(-1)).
		Result()

	if err != nil {
		log.Println("Error while getting account transactions: ", vars["address"],
			err)
	}

	jsonString, err := json.Marshal(tx)

	if err != nil {
		log.Println("Error while marshalling account transactions: ", err)
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, string(jsonString))
}

func getVersion() string {
	return fmt.Sprintf("API version: v%v", version)
}
