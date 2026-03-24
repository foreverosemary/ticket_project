-- KEYS: stock, Key set
-- ARGV: userId, need
-- 检查用户是否已经抢过票
local isMemeber = redis.call('sismember', KEYS[2], ARGV[1])
if isMemeber == 1 then
    return -1 -- 表示重复下单
end

-- 获取当前库存与用户所需量
local stock = tonumber(redis.call("get", KEYS[1]))
local need = ARGV[2]

-- 判断库存是否充足
if stock  == nil or stock < need then
    return 0 -- 表示库存不足
end

-- 执行扣减逻辑
redis.call("decrby", KEYS[1], need)
redis.call("sadd", KEYS[2], ARGV[1])

return 1 -- 表示抢票成功
