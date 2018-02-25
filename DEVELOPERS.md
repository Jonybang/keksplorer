## Redis keys

* **Hash** `block:150000:detail` Details of block
* **Sorted set** `block:150000:tx_list` List of tx

HSET block_tx:150000:0x000000000:detail # Details of tx
ZSET block_tx:150000:0x000000000:list # List of accounts

HSET account:0x000:detail # Details of account
ZSET account:0x000:tx_list # List of tx

ZSET queue:blocks