package main

type Block struct {
	Hash      string `json:"hash"`
	Validator string `json:"validator"`
	GasUsed   string `json:"gasUsed"`
	Number    string `json:"number"`
	Size      string `json:"size"`
	Timestamp string `json:"timestamp"`

	TransactionsLength int `json:"txs, omitempty"`
}

type Transaction struct {
	From        string `json:"from"`
	To          string `json:"to"`
	Hash        string `json:"hash"`
	Input       string `json:"input"`
	Value       string `json:"value"`
	Gas         string `json:"gas"`
	GasPrice    string `json:"gasPrice"`
	Nonce       string `json:"nonce"`
	BlockNumber string `json:"blockNumber"`
	Timestamp   string `json:"timestamp"`
}

type Account struct {
	Address string `json:"address"`
	Balance string `json:"balance, omitempty"`

	Txs []string `json:"txs, omitempty"`
}
