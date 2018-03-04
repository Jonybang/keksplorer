package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-redis/redis"
	"github.com/gorilla/mux"
)

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
	r.HandleFunc("/blocks/{blockNumber}", blockViewController).Methods("GET")
	r.HandleFunc("/transactions", transactionsViewController).Methods("GET")
	r.HandleFunc("/transactions/{hash}", transactionViewController).Methods("GET")

	// API
	r.HandleFunc("/api/latest_block", latestBlockController).Methods("GET")
	r.HandleFunc("/api/blocks/{blockNumber}", blockController).Methods("GET")
	r.HandleFunc("/api/transactions", transactionsController).Methods("GET")
	r.HandleFunc("/api/transactions/{hash}", transactionController).Methods("GET")
	r.HandleFunc("/api/accounts", accountsController).Methods("GET")
	r.HandleFunc("/api/accounts/{address}/transactions",
		accountTransactionsController).Methods("GET")

	staticDir := getWorkDir() + "/web/public/assets/"

	// STATIC
	r.PathPrefix("/assets").
		Handler(http.StripPrefix("/assets", http.FileServer(http.Dir(staticDir))))

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
	dir := getWorkDir()

	tmpl := template.Must(template.ParseFiles(dir + "/web/public/index.html"))

	tmpl.Execute(w, "")
}

func latestBlockViewController(w http.ResponseWriter, r *http.Request) {
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

	dir := getWorkDir()

	tmpl := template.Must(template.ParseFiles(dir + "/web/public/latest_block.html"))

	tmpl.Execute(w, latestBlock)
}

func blockViewController(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	block, err := redisClient.HGetAll(
		fmt.Sprintf("block:%v:detail", vars["blockNumber"])).Result()

	log.Println(fmt.Sprintf("Get block: %v", block))

	if err != nil {
		log.Println("Error while getting block #", vars["blockNumber"], err)
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	dir := getWorkDir()

	tmpl := template.Must(template.ParseFiles(dir + "/web/public/block.html"))

	tmpl.Execute(w, block)
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

	dir := getWorkDir()

	tmpl := template.Must(template.ParseFiles(
		dir + "/web/public/transactions.html"))

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

	dir := getWorkDir()

	tmpl := template.Must(template.ParseFiles(
		dir + "/web/public/transaction.html"))

	tmpl.Execute(w, txDetail)
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

func getWorkDir() string {
	dir, err := os.Getwd()

	if err != nil {
		log.Println("Error while trying to get work directory:", err)
	}

	return dir
}
