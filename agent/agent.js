const Web3 = require('web3');
const redis = require('redis');
const winston = require('winston');
const bluebird = require('bluebird');

const WS_API_URL = process.env.WS_API_URL || "ws://127.0.0.1:8546";
const REDIS_URL = process.env.REDIS_URL || "redis://127.0.0.1:6379";

const redisClient = redis.createClient(REDIS_URL);

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

(() => {
  redisClient.on('error', (err) => {
    throw err;
  });

  getBlocks();
  subscribeToNewBlocks();
})();

async function getBlocks() {
  logger.log({level: 'info', message: "Start grabbing blocks."});

  let latestBlock = await redisClient.getAsync('latest_block');

  if (!latestBlock) {
    logger.log({level: 'info', message: "Latest block not found. Starting from 0."});

    latestBlock = 0;
  }

  let args = ['queue:blocks'];
  let latestBlockInChain = await web3.eth.getBlock('latest');

  for (let i = latestBlock; i <= latestBlockInChain.number; i++) {
    args.push(0, i);
  }

  if (args.length === 1) {
    return;
  }

  try {
    redisClient.zadd(args);
  } catch (err) {
    logger.log({level: 'error', message: `Error while adding block tu queue: ${err}`})
  }
}

async function subscribeToNewBlocks() {
  subscription = web3.eth.subscribe('newBlockHeaders', (err, res) => {
    if (err && err.reason) {
      logger.log({level: 'error', message: `Error while subscribe: ${err.reason}.`});
    }
  }).on('data', async (block) => {
    let args = ['queue:blocks', 0, block.number];

    try {
      await redisClient.set('latest_block', block.number);
      await redisClient.zadd(args);
      logger.log({level: 'info', message: `Added block #${block.number}.`});
    } catch (err) {
      logger.log({level: 'error', message: `Error while adding block to redis: ${err}.`});
    }
  });
}
