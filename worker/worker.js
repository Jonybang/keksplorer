console.log('REDIS_URL', process.env.REDIS_URL);
console.log('JSON_RPC_API_URL', process.env.JSON_RPC_API_URL);

const {promisify} = require('util');


const Web3 = require('web3');
const redis = require('redis');

const redisClient = redis.createClient(process.env.REDIS_URL);
const web3 = new Web3(process.env.JSON_RPC_API_URL);

const getAsync = promisify(redisClient.get).bind(redisClient);
const hsetAsync= promisify(redisClient.hset).bind(redisClient);
const zaddAsyng = promisify(redisClient.zadd).bind(redisClient);

const redisConnection = "";

(async () => {
    // TODO: fetch and loop; sleep if no results available

    await parseBlock(1975);
})();


// HSET block:150000:detail # Details of block
// ZSET block:150000:tx_list # List of tx
//
// HSET block_tx:150000:0x000000000:detail # Details of tx
// ZSET block_tx:150000:0x000000000:list # List of accounts
//
// HSET account:0x000:detail # Details of account
// ZSET account:0x000:tx_list # List of tx
//
// ZSET queue:blocks

async function parseBlock(blockId) {
    let block = await web3.eth.getBlock(blockId);
    // let multi = redisClient.multi();

    let blockToStore = [
        "number", block.number,
        "validator", block.author || block.miner,
        "gasUsed", block.gasUsed,
        "size", block.size,
        "timestamp", block.timestamp,
        "hash", block.hash,
    ];

    hsetAsync(`block:${block.number}:detail`, ...blockToStore);

    console.log('block to store', blockToStore);
    let txHashes = block.transactions;

    if (txHashes.length === 0) {
        return;
    }

    for (let i = 0; i < txHashes.length; i++) {
        let txHash = txHashes[i];

        parseTransaction(txHash);
        associateTxWithBlock(txHash, i, blockId);
    }
}

async function parseTransaction(txHash) {
    let tx = await web3.eth.getTransaction(txHash);

    let txToStore = [
        "from", tx.from,
        "gas", tx.gas,
        "gasPrice", tx.gasPrice,
        "hash", tx.hash,
        "input", tx.input,
        "nonce", tx.nonce,
    ];

    hsetAsync(`block_tx:${tx.blockNumber}:${txHash}:detail`, ...txToStore);
    zaddAsyng(`block_tx:${tx.blockNumber}:${txHash}:list`, 0, tx.from);
    zaddAsyng(`block_tx:${tx.blockNumber}:${txHash}:list`, 1, tx.to);

    console.log('tx to store', txToStore);

    associateAccountWithTx(tx.from, tx.blockNumber, tx.hash);
    associateAccountWithTx(tx.to, tx.blockNumber, tx.hash);
}

async function associateTxWithBlock(txHash, order, blockId) {
    zaddAsyng(`block:${blockId}:tx_list`, order, txHash);
}

async function associateAccountWithTx(accountAddress, nonce, txHash) {
    zaddAsyng(`account:${accountAddress}:tx_list`, nonce, txHash);
}
