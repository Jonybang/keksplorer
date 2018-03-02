package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-redis/redis"
	"github.com/gorilla/mux"
)

var redisClient *redis.Client

func main() {
	r := mux.NewRouter()

	redisClient = redis.NewClient(&redis.Options{
		Addr:     "127.0.0.1:6379",
		Password: "",
		DB:       0,
	})

	r.HandleFunc("/api/blocks", blocksController).Methods("GET")
	r.HandleFunc("/api/block/{blockNumber}", blockController).Methods("GET")
	r.HandleFunc("/api/transactions/{blockNumber}", transactionsController).Methods("GET")

	srv := &http.Server{
		Handler:      r,
		Addr:         "0.0.0.0:8080",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  20 * time.Second,
	}

	log.Fatal(srv.ListenAndServe())
}

func blocksController(w http.ResponseWriter, r *http.Request) {
	blockNumber := 0
	blocks := []map[string]string{}

	for ; ; blockNumber++ {
		block, err := redisClient.HGetAll(
			fmt.Sprintf("block:%v:detail", blockNumber)).Result()

		if err != nil {
			log.Println("Error while grabbing blocks: ", err)
			break
		}

		if len(block) == 0 {
			break
		}

		blocks = append(blocks, block)
	}

	jsonString, err := json.Marshal(blocks)

	if err != nil {
		log.Println("Error while marshalling blocks: ", err)
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
	vars := mux.Vars(r)

	tx, err := redisClient.ZRange(
		fmt.Sprintf("block:%v:tx_list", vars["blockNumber"]), int64(0), int64(-1)).
		Result()

	if err != nil {
		log.Println("Error while getting transaction list on block #",
			vars["blockNumber"], err)
	}

	log.Println(fmt.Sprintf("Get transactions list: %v", tx))

	jsonString, err := json.Marshal(tx)

	if err != nil {
		log.Println("Error while marshalling transactions: ", err)
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, string(jsonString))
}
