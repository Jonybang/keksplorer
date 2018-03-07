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

(async () => {
  redisClient.on('error', (err) => {
    throw err;
  });

  getBlocks();
  subscribeToNewBlocks();
})();

async function getBlocks() {
  logger.log({level: 'info', message: "Start grabbing blocks."});

  let latestRedisBlock = await redisClient.getAsync('latest_block');

  if (!latestRedisBlock) {
    logger.log({level: 'info', message: "Latest block not found. Starting from 0."});

    latestRedisBlock = 0;
  }

  let latestChainBlock = await web3.eth.getBlock('latest');

  logger.log({level: 'info', message: `Latest redis block: ${latestRedisBlock}, latest chain block: ${latestChainBlock.number}`});

  const STEP = 100000;
  let from = latestRedisBlock;
  let to = ((latestRedisBlock + STEP) > latestChainBlock.number) ? latestChainBlock.number : latestRedisBlock + STEP;

  while (from < latestChainBlock.number) {
    logger.log({level: 'info', message: `from ${from}, to ${to}, latestChainBlock ${latestChainBlock.number}`});
    await putBlockNumbersToQueue(from, to);
    from = to;
    to = ((to + STEP) > latestChainBlock.number) ? latestChainBlock.number : to + STEP;
  }
}

async function putBlockNumbersToQueue(from, to) {
  let args = ['queue:blocks'];

  for (let i = from; i <= to; i++) {
    args.push(0, i);
  }

  if (args.length === 1) {
    return;
  }

  try {
    logger.log({level: 'info', message: `Blocks to add: ${args[0]}...${args[args.length - 1]}`});

    await redisClient.zaddAsync(args);
  } catch (err) {
    logger.log({level: 'error', message: `Error while adding block tu queue: ${err}`});
  }
}

async function subscribeToNewBlocks() {
  web3.eth.subscribe('newBlockHeaders', (err, res) => {
    if (err) {
      logger.log({level: 'error', message: `Error while subscribe: ${err}.`});
    }
  }).on('data', async (block) => {
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
