console.log('REDIS_URL', process.env.REDIS_URL);
console.log('JSON_RPC_API_URL', process.env.JSON_RPC_API_URL);

const {promisify} = require('util');

const net = require('net');
const Web3 = require('web3');
const redis = require('redis');

const redisClient = redis.createClient(process.env.REDIS_URL);
const web3 = new Web3(process.env.JSON_RPC_API_URL, net);

const hsetAsync= promisify(redisClient.hset).bind(redisClient);
const zaddAsync = promisify(redisClient.zadd).bind(redisClient);
const zrangeAsync = promisify(redisClient.zrange).bind(redisClient);
const zrangebyscoreAsync = promisify(redisClient.zrangebyscore).bind(redisClient);

const redisConnection = "";
Promise.prototype.finally = function(cb) {
    const res = () => this;
    const fin = () => Promise.resolve(cb()).then(res);
    return this.then(fin, fin);
};

function sleep(ms = 0) {return new Promise(r => setTimeout(r, ms));}

const logger = winston.createLogger({
    level: 'info',
    format: winston.format.json(),
    transports: [
        new winston.transports.File({ filename: 'error.log', level: 'error' }),
        new winston.transports.File({ filename: 'combined.log' })
    ]
});

if (process.env.NODE_ENV !== 'production') {
    logger.add(new winston.transports.Console({
        format: winston.format.simple()
    }));
}

// NOTICE: if already parsed the new parsing should not affect on redis values
(async () => {
    for (let i = 0; i < 20000; i++) {
        zaddAsync('queue:blocks', 0, i);
        if (2000 % 400) {
            await sleep(3000);
        }
    }
})();

( () => {
    function parse() {
        zrangebyscoreAsync('queue:blocks', 0, 0)
            .then((res) => {
                logger.log({level: 'info', message: `Blocks to parse: ${res.length}`});
                for (let i = 0; i < res.length; i++) {
                    parseBlock(i);
                }
            })
            .catch((e) => {
                logger.log({level: 'error', message: `Failed to fetch new blocks from db queue: ${e}`});
            })
            .finally(() => {
                setTimeout(parse, 500);
            });
    }

    parse();
})();

async function parseBlock(blockId) {
    let block;

    try {
        block = await web3.eth.getBlock(blockId);
    } catch (e) {
        logger.log({level: 'error', message: `Failed to fetch block #${blockId} information from chain: ${e}`});
        return;
    }

    let multi = redisClient.multi();

    let blockToStore = [
        "number", block.number,
        "validator", block.author || block.miner,
        "gasUsed", block.gasUsed,
        "size", block.size,
        "timestamp", block.timestamp,
        "hash", block.hash,
    ];

    try {
        await hsetAsync(`block:${block.number}:detail`, ...blockToStore);
    } catch (e) {
        logger.log({level: 'error', message: `Failed to save block #${blockId} information to db: ${e}`});
        return;
    }

    let txHashes = block.transactions;

    console.log(`Parsed Block #${block.number}`);

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
    zaddAsync(`block_tx:${tx.blockNumber}:${txHash}:list`, 0, tx.from);
    zaddAsync(`block_tx:${tx.blockNumber}:${txHash}:list`, 1, tx.to);

    associateAccountWithTx(tx.from, tx.blockNumber, tx.hash);
    associateAccountWithTx(tx.to, tx.blockNumber, tx.hash);
}

async function associateTxWithBlock(txHash, order, blockId) {
    zaddAsync(`block:${blockId}:tx_list`, order, txHash);
}

async function associateAccountWithTx(accountAddress, nonce, txHash) {
    zaddAsync(`account:${accountAddress}:tx_list`, nonce, txHash);
}
