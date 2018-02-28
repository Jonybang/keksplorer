const Web3 = require('web3');
const redis = require('redis');

const WS_RPC_URL = "ws://127.0.0.1:8546" || process.env.WS_RPC_URL;
const REDIS_URL = "redis://127.0.0.1:6379" || process.env.REDIS_URL;
const RECONNECTION_MULTIPLIER = 1000;

const redisClient = redis.createClient(REDIS_URL);

let web3 = new Web3(new Web3.providers.WebsocketProvider(WS_RPC_URL));
let reconnectionTimeout = 5;
let subscription = null;
let reconnection = null;

(() => {
  redisClient.on('error', (err) => {
    throw err;
  });

  let length = 0;

  getBlocksCountFromRedis().then((l) => {
    length = l;
  }).catch((err) => {
    console.log(`Error while getting blocks count from redis: ${err}`);
  });

  if (length === 0) {
    getAllBlocks();
  } else {
    getNewBlocks(length);
    subscribeToHeaders();
  }
})();

async function getWSProvider() {
  return new Promise((resolve, reject) => {
    resolve(new Web3.providers.WebsocketProvider(WS_RPC_URL));
  });
}

async function getWeb3(WSProvider) {
  return new Promise((resolve, reject) => {
    resolve(new Web3(WSProvider));
  });
}

async function getBlocksCountFromRedis() {
  return new Promise((resolve, reject) => {
    redisClient.zcount('queue:blocks', '-inf', '+inf', (err, length) => {
      if (err) reject(err);

      resolve(length);
    });
  });
}


async function addDataToRedis(args) {
  return new Promise((resolve, reject) => {
    redisClient.zadd(args, (err, res) => {
      if (err) reject(err);

      resolve(res);
    });
  });
}

async function getAllBlocks () {
  console.log("Start grabbing");

  let latestBlockNumber = 0;
  let args = ['queue:blocks'];

  for (let i = 0; ; i++) {
    let block = await web3.eth.getBlock(i);

    if (block === null) {
      console.log(`Stop grabbing. Latest block number is ${i}`);

      latestBlockNumber = i;

      break;
    }

    args.push(0, block.number);
  }

  try {
    await addDataToRedis(args);
  } catch(err) {
    console.log(`Error while adding block to redis: ${err}`)
  }

  getNewBlocks(latestBlockNumber);
  subscribeToHeaders();
}

async function getNewBlocks (latestBlockNumber) {
  console.log("Start grabbing new blocks");

  let args = ['queue:blocks'];

  for (let i = latestBlockNumber; ; i++) {
    let block = await web3.eth.getBlock(i);

    if (block === null) {
      console.log(`Stop grabbing. Latest block number is ${i}`);
      break;
    }

    args.push(0, block.number.toString());
  }

  if (args.length === 1) {
    return;
  }

  try {
    await addDataToRedis(args);
  } catch(err) {
    console.log(`Error while adding block to redis: ${err}`)
  }
}

async function subscribeToHeaders() {
  subscription = web3.eth.subscribe('newBlockHeaders', (err, res) => {
    if (err && err.reason) {
      console.log(`Error while subscribe: ${err.reason}`);

      try {
        reconnect();
      } catch (err) {
        console.log(`Error while trying to reconnect: ${err}`);
      }
    }
  }).on('data', async (block) => {
    let length = await getBlocksCountFromRedis();

    if (length === 0) {
      console.log(`Ops! Set 'queue:blocks' are empty. Adding all blocks...`);
      return getAllBlocks();
    }

    let args = ['queue:blocks', 0, block.number];

    try {
      addDataToRedis(args);
      console.log(`Added block #${block.number}`);
    } catch (err) {
      console.log(`Error while adding block to redis: ${err}`);
    }
  });
}

async function reconnect() {
  subscription = null;

  reconnection = setInterval(async () => {
    try {
      let WSProvider = await getWSProvider();
      web3 = await getWeb3(WSProvider);

      let latestBlockNumber = await getBlocksCountFromRedis();

      await getNewBlocks(latestBlockNumber);

      subscribeToHeaders();
      stopReconnection();
    } catch (err) {
      console.log(`Trying to reconnect...\n${err}`);
    }
  }, reconnectionTimeout * RECONNECTION_MULTIPLIER);
}

async function stopReconnection() {
  clearInterval(reconnection);

  console.log("Connection retrieved. Subscribe to blocks.")
}
