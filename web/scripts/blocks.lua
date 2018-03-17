local count = tonumber(KEYS[1])
local k = 1
local blocks = {}

local latest_block = tonumber(redis.call("GET", "latest_block"))
local i = latest_block

while(i > latest_block - count) do
	local block = redis.call("HGETALL", "block:" .. i .. ":detail")
	local blockAsObj = {}

	local nextkey
	for i, v in ipairs(block) do
		if i % 2 == 1 then
			nextkey = v
		else
			blockAsObj[nextkey] = v
		end
	end

	local txKeys = redis.call("KEYS", "block_tx:" .. i .. ":*:detail")

	if table.getn(txKeys) ~= 0 then
		blockAsObj["txs"] = table.getn(txKeys)
	end

	if table.getn(block) ~= 0 then
		blocks[k] = blockAsObj
		k = k + 1
	end
	i = i - 1
end

return cjson.encode(blocks)