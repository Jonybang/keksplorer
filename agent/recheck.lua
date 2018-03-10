local latest = tonumber(KEYS[1])
local missing = 0
local step = 1000

local from = 0
local to = from + step

while from < latest do
    local changed = false    
    local args = {"ZADD", "queue:blocks"}

    for i = from, to do
        if i == latest then
            break
        end

        local res = redis.call("EXISTS", "block:" .. i .. ":detail")
        if res == 0 then
            table.insert(args, 0)
            table.insert(args, i)
            changed = true
            missing = missing + 1
        end
    end

    if changed then
        redis.call(unpack(args))
    end

    from = to
    to = to + step
end
