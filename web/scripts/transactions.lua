local count = tonumber(KEYS[1])
local transactions = {}

local txs = redis.call("ZREVRANGEBYSCORE", "transactions:order", "+inf", "-inf", "LIMIT", "0", count)

for k,v in pairs(txs) do
	local txKey = redis.call("KEYS", "block_tx:*:" .. v .. ":detail")
	local txDetail = redis.call("HGETALL", unpack(txKey))

	local txDetails = {}

	local nextkey
	for i, v in ipairs(txDetail) do
		if i % 2 == 1 then
			nextkey = v
		else
			txDetails[nextkey] = v
		end
	end

	transactions[k] = txDetails
end

return cjson.encode(transactions)
