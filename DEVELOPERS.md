## Redis keys

* **Hash** `block:150000:detail` Details of block
* **Sorted set** `block:150000:tx_list` List of tx

* **Hash** `block_tx:150000:0x000000000:detail` Details of tx
* **Sorted set** `block_tx:150000:0x000000000:list` List of accounts

* **Hash** `account:0x000:detail` # Details of account
* **Sorted set** `account:0x000:tx_list` # List of tx

* **Sorted set** `queue:blocks`