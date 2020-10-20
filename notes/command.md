#### 字符串
> 字符串可以3种类型的值，`字节串`, `整数`, `浮点数`  
> 可以设定任意的数值，对存储着整数和浮点数的key进行自增或者自减操作. 整数的取值范围与系统的long integer范围相同  

```shell script
# 处理数字
incr num # 自增1
incrby num amount # 增加amount
decr num # 自减1
decrby num amount # 减少amount

# 处理字节串
append key value # 想key的值添加vlaue  a:ss append a bb --> ssbb
getrange key start end # 获取一个字符串的部分闭区间
setrange key start end # 给某个区间设置值
setbit key offset value # 给二进制的某个位置设置值, offset为偏移量
bitcount key start end # 统计范围区间内1个数

setbit a-key 2 1; setbit a-key 7 1 --> ! # 偏移量是从高位开始00000000 10000100
```


#### 列表
> redis允许列表从两端插入数据

```shell script
rpush key val ...# 列表的右侧插入若干
lpush key val ...# 列表左侧插入若干
rpop key # 右侧删除，并返回
lpop key # 左侧删除并返回
lindex key offset # 查看index的数据
lrange key start end # 遍历列表
ltrim key start end # 从左侧对列表进行修剪

blpop key ... timeout #  从第一个非空的列表弹出左边的元素，或者timeout
brpop key ... timeout
rpoplpush source-key dest-key  # 从sourcekey右侧去除数据，从destkey左侧添加
brpoplpush source-key dest-key timeout # 复制阻塞，或者时间超时
```

#### 集合
> 以无序的方式存储数据，用户可以快速的添加，移除，检查元素是否存在

```shell script
sadd key item... # 集合添加元素
srem key item... # 集合移除元素
sismember key item # 检查元素是否存在集合中
scard key # 返回集合元素的数量
smembers kye # 返回集合列表
srandmember key count # 随机返回一(多个)个元素
spop key # 在集合随机的移除元素
smove source-kye dest-key item # 如果集合source-key包含item，那么删除，切在dest-key添加item

# 结合操作
sdiff key ... # 返回第一个集合不存在与其他集合的元素
sdiffstore dest key ... # 得到差异并保存
sinter key ... # 返回同时存在所有集合的元素
sinterstore dest key ... # 将交集的数据保存在dest
sunion key ... # 并集
sunion dest key ... # 并集并保存
```

#### 散列
> 可以将更多的键值对存储到key里。

```shell script
hget/hset # 单个的字段获取，设置
hmget key-name field ... # 从散列中取出一个或者多个键的值
hmset key-name field value ... # 给散列设置一个或者多个值
hdel key-name field ... # 从散列删除多个字段
hlen key-name # 返回键值对的数量

hexists key-name field # 检查字段是否存在
hkeys key-name # 返回所有的key
hvals key-name # 返回所有的val
hgetall key-name # 返回所有的key和val
hincrby key-name field increment # 将字段的只增加increment。
hincrbyfloat key-name field increment # 增加浮点数
```

#### 有序集合
> 有序集合存储这分数与成员之际的映射，提供分值处理命令.

```shell script
zadd key-name score member #添加元素
zrem key-name member ... # 移除多个member
zcard key-name # 成员数量
zincrby key-name increment member # 成员分值增加
zcount key-name min max # 在min，max直接成员的数量
zrank key-name member # 返回member的名次
zscore key-name member # 返回成员的分值
zrange key-name # 遍历有序集合

zrevrank key-name member # 返回从大小的顺序，member的位置
zrevrange key-name # 反向遍历
zrangebyscore key-name min max # 返回范围内的对象的数据
zrevrangebyscore key-name max min # 反向遍历取范围内的数据
zremrangebyrank key-name start stop # 删除排名位于start，stop直接的数据
zremrangebyscore key-name min max # 删除分数位于min，max之间的数据
zinterstore dest z key-count key... aggregate(聚合条件sum, min, max), 默认聚合是sum
zunionstore dest z key-count key ... aggergate
```

#### 发布订阅
> 发布者向channel发送消息，每当有新的消息到channel，那么会通知所有的订阅者。

```shell script
subscribe chan1 ... # 订阅一个或者多个chan
unsbuscribe chan ... # 取消订阅, 执行的时候没有加任何参数，退订所有的chan
publish chan message # 向通道发布通知
psubscribe pattern ... # 订阅符合匹配的所有频道
punsbuscribe pattern ... # 退订给定模式的通道，不加参数，退订所有

127.0.0.1:6379> publish test1 helloworld
(integer) 1
127.0.0.1:6379> publish test1 youcanspeak
(integer) 1
127.0.0.1:6379>

# 下面先订阅，上面在发布
root@a900f61edacb:/data# redis-cli
127.0.0.1:6379> subscribe test1
Reading messages... (press Ctrl-C to quit)
1) "subscribe"
2) "test1"
3) (integer) 1
1) "message"  
2) "test1"
3) "helloworld"

```

#### 排序
```shell script
sort key [BY pattern] [LIMIT offset count] [GET pattern [GET pattern ...]] [ASC|DESC] [ALPHA] [STORE destination]
```

#### 事务
> 有五个操作让用户在不被打断的情况下执行多个操作`watch, multi, exec, unwatch, discard`
> redis的基本事务需要用到`multi, exec`, 客户端收到multi命令时，后续的命令都放到一个队列
> 里面，直到客户端发送exec命令，


#### 设置键的过期
```shell script
persist key-name # 去掉过期时间
ttl key-name # 距离过期还有多长时间
expire key-name seconds # 设置几秒后过期
expireat key-name timestamp # 设置过期时间，unix时间戳
pttl key-name # 距离过期还有多少毫秒
pexpire key-name millionseconds # 设置毫秒过期
pexpireat key-name timestamp-millionseconds # 设置时间戳毫秒过去，unix时间戳
```


#### 完成代码
- 订阅与取消订阅
- 事务操作
