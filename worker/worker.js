console.log('REDIS_URL', process.env.REDIS_URL);
console.log('JSON_RPC_API_URL', process.env.JSON_RPC_API_URL);
const HOME = process.env.HOME;

const {promisify} = require('util');

const net = require('net');
const Web3 = require('web3');
const redis = require('redis');
const assert = require('assert');
const winston = require('winston');
const bluebird = require('bluebird');

const redisClient = redis.createClient(process.env.REDIS_URL || "http://localhost:6379");
const web3 = new Web3(process.env.JSON_RPC_API_URL || "http://localhost:8545", net);

bluebird.promisifyAll(redis.Multi.prototype);

const zrangebyscoreAsync = promisify(redisClient.zrangebyscore).bind(redisClient);
// const zaddAsync = promisify(redisClient.zadd).bind(redisClient);

Promise.prototype.finally = function (cb) {
    const res = () => this;
    const fin = () => Promise.resolve(cb()).then(res);
    return this.then(fin, fin);
};

function sleep(ms = 0) {
    return new Promise(r => setTimeout(r, ms));
}

const logger = winston.createLogger({
    level: 'info',
    transports: [
        new winston.transports.File({filename: './logs/error.log', level: 'error'}),
        new winston.transports.File({filename: './logs/combined.log'})
    ]
});

if (process.env.NODE_ENV !== 'production') {
    logger.add(new winston.transports.Console({
        format: winston.format.simple()
    }));
}

async function checkConnections() {
    try {
        await web3.eth.getProtocolVersion();
    } catch (e) {
        logger.log({level: 'error', message: `Failed to connect parity node: ${e}`});
        throw new Error(`Failed to connect parity node: ${e}`);
    }

    try {
        const pingAsync = promisify(redisClient.ping).bind(redisClient);
        await pingAsync();
    } catch (e) {
        logger.log({level: 'error', message: `Failed to connect redis: ${e}`});
    }
}

// (async () => {
//     await checkConnections();
//
//     for (let i = 0; i < 100000; i++) {
//         zaddAsync('queue:blocks', 0, i);
//         if (2000 % 400) {
//             await sleep(3000);
//         }
//     }
// })();

(() => {
    redisClient.on('ready', () => {
      checkConnections()
          .then(parse)
          .catch(e => {
              logger.log({level: 'error', message: `Failed to connect parity node`});
          });
    });

    function parse() {
        zrangebyscoreAsync('queue:blocks', 0, 0)
            .then((res) => {
                let promises = [];

                let len = (res.length > 10000) ? 10000 : res.length;

                logger.log({level: 'info', message: `Blocks to parse: ${len} of ${res.length}`});

                for (let i = 0; i < len; i++) {
                    promises.push(parseBlock(res[i]));
                }

                return Promise.all(promises);
            })
            .catch(e => {
                logger.log({level: 'error', message: `Error while parsing blocks: ${e}`});
            })
            .then(() => {
                setTimeout(parse, 500);
            });
    }
})();

async function parseBlock(blockId) {
    assert.notEqual(blockId, null);

    let block;

    try {
        block = await web3.eth.getBlock(blockId);
    } catch (e) {
        logger.log({level: 'error', message: `Failed to fetch block #${blockId} information from chain: ${e}`});
        return;
    }

    if (block === null) {
      logger.log({level: 'error', message: `Null block by number ${blockId}. Skipping...`});
      return;
    }

    // multi request should go through block => txs => accounts parsing and commit changes at the end
    let multi = redisClient.multi();

    if (block && block.transactions) {
      for (let i = 0; i < block.transactions.length; i++) {
        multi.zadd('transactions:order', block.timestamp, block.transactions[i]);
      }
    }

    let blockToStore = [
        "number", block.number,
        "validator", block.author || block.miner,
        "gasUsed", block.gasUsed,
        "size", block.size,
        "timestamp", block.timestamp,
        "hash", block.hash,
    ];

    multi.hset(`block:${block.number}:detail`, ...blockToStore);

    let txHashes = block.transactions;

    if (txHashes.length !== 0) {
        logger.log({level: 'debug', message: `Block #${block.number} is empty`});

        logger.log({level: 'info', message: `Parsing block #${blockId}`});

        for (let i = 0; i < txHashes.length; i++) {
            let txHash = txHashes[i];

            await parseTransaction(multi, txHash);
            associateTxWithBlock(multi, txHash, i, blockId);
        }
    }

    multi.zadd(`queue:blocks`, 1, blockId);

    await multi.execAsync();
}

async function parseTransaction(multi, txHash) {
    assert.notEqual(multi, null);
    assert.notEqual(txHash, null);

    logger.log({level: 'info', message: `Parsing tx #${txHash}`});

    let tx = await web3.eth.getTransaction(txHash);
    let value;

    if (typeof parseInt(tx.value) === "number") {
        value = web3.utils.fromWei(tx.value);
        value = cutNumber(+value);
    } else {
        value = tx.value;
    }

    let txToStore = [
        "from", tx.from,
        "to", tx.to || "",
        "value", value,
        "gas", tx.gas,
        "gasPrice", tx.gasPrice,
        "hash", tx.hash,
        "input", tx.input,
        "nonce", tx.nonce,
        "blockNumber", tx.blockNumber,
    ];

    multi.hset(`block_tx:${tx.blockNumber}:${txHash}:detail`, ...txToStore);
    multi.zadd(`block_tx:${tx.blockNumber}:${txHash}:list`, 0, tx.from);

    // tx is a contract deployment
    if (tx.to === null) {
        // TODO: handle contract deployment
    } else {
        multi.zadd(`block_tx:${tx.blockNumber}:${txHash}:list`, 1, tx.to);
        associateAccountWithTx(multi, tx.to, tx.blockNumber, tx.hash);
        addAccountOrder(multi, tx.to, tx.blockNumber);
    }

    associateAccountWithTx(multi, tx.from, tx.blockNumber, tx.hash);
    addAccountOrder(multi, tx.from, tx.blockNumber);
}

function associateTxWithBlock(multi, txHash, order, blockId) {
    assert.notEqual(txHash, null);
    assert.notEqual(order, null);
    assert.notEqual(blockId, null);

    multi.zadd(`block:${blockId}:tx_list`, order, txHash);
}

function associateAccountWithTx(multi, accountAddress, blockNumber, txHash) {
    assert.notEqual(accountAddress, null);
    assert.notEqual(blockNumber, null);
    assert.notEqual(txHash, null);

    multi.zadd(`account:${accountAddress}:tx_list`, blockNumber, txHash);
}

function addAccountOrder(multi, accountAddress, blockNumber) {
    assert.notEqual(accountAddress, null);
    assert.notEqual(blockNumber, null);

    multi.zadd(`account:order`, blockNumber, accountAddress);
}

function cutNumber(number) {
    if (number.toString().split(".").length === 2) {
        if (number.toString().split(".")[1].length > 9) {
            return number.toFixed(9);
        }
    }
    return number;
}
