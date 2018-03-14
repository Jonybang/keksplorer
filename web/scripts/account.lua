local accountAddress = KEYS[1]
local account = {}

local txList = redis.call("ZRANGE", "account:" .. accountAddress .. ":tx_list", "0", "-1")

if table.getn(txList) ~= 0 then
	account["txs"] = txList
end

return cjson.encode(account)
