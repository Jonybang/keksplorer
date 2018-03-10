const fs = require('fs');
const path = require('path');
const Web3 = require('web3');
const redis = require('redis');
const winston = require('winston');
const bluebird = require('bluebird');

const WS_API_URL = process.env.WS_API_URL || "ws://127.0.0.1:8546";
const REDIS_URL = process.env.REDIS_URL || "redis://127.0.0.1:6379";

const redisClient = redis.createClient(REDIS_URL);
const recheckScript = fs.readFileSync(path.join(__dirname, 'recheck.lua'));

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

bluebird.promisifyAll(redis.RedisClient.prototype);
bluebird.promisifyAll(redis.Multi.prototype);

let web3 = new Web3(new Web3.providers.WebsocketProvider(WS_API_URL));

redisClient
  .on('error', (err) => {
    throw err;
  })
  .on('ready', subscribeToNewBlocks);

async function subscribeToNewBlocks() {
  let recheckLaunched = false;
  web3.eth.subscribe('newBlockHeaders', (err, res) => {
    if (err) {
      logger.log({level: 'error', message: `Error while subscribe: ${err}.`});
    }
  }).on('data', async (block) => {
    if (!recheckLaunched) {
      recheckLaunched = true;

      redisClient.eval(recheckScript, 1, block.number);
    }

    let args = ['queue:blocks', 0, block.number];

    try {
      await redisClient.setAsync('latest_block', block.number);
      await redisClient.zaddAsync(args);
      logger.log({level: 'info', message: `Added block #${block.number}.`});
    } catch (err) {
      logger.log({level: 'error', message: `Error while adding block to redis: ${err}.`});
    }
  });
}
