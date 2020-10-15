#### 数据库与缓存服务器的相关特性

| 名称 | 类型 | 数据存储选项 | 查询类型 | 附加功能 |
| :----: | :----: | :----: | :----: | :----: |
| redis | 使用内存的非关系型数据库 | string,list,set,zset,hash | 每种数据类型都有自己的一套操作 | 发布订阅， 主从复制, 数据持久化 |
| memcached | 使用内存的键值缓存 | 只有string类型 | 创建，删除，更新，读取等其他的几个命令 | 提升性能，多线程 |
| mysql | 关系型数据库 | 每个数据库有多张表，每张表多个字段，多表的视图 | select, update, delete, drop, 存储过程 | ACID特性(需要innodb存储引擎), 多种集群模式 |
| mongodb | 使用磁盘存储的非关系型文档存储 | bson文档 | 有一套自己的查询语法 | 分片/副本模式, map-reduce操作 |

#### 使用redis的理由
- memcached支持的数据类型是string，可以通过append向后追加元素，但是删除的时候采取的是黑名单的模式，避免对这些数据的操作。
而redis可以直接添加删除元素

#### 启动redis
```shell script
docker pull redis
docker run -itd -p6379:6379 --name redis redis
```

#### 操作基本类型

- string
```shell script
127.0.0.1:6379> set hello world
OK
127.0.0.1:6379> get hello
"world"
127.0.0.1:6379> del hello
(integer) 1
127.0.0.1:6379> get hello
(nil)
```

- list
```shell script
127.0.0.1:6379> lpush list 1 2 3
(integer) 3
127.0.0.1:6379> rpush list 4 5 6
(integer) 6
127.0.0.1:6379> lrange list 0 -1
1) "3"
2) "2"
3) "1"
4) "4"
5) "5"
6) "6"
127.0.0.1:6379> lindex list 0
"3"
127.0.0.1:6379> lpop list
"3"
127.0.0.1:6379> lrange list 0 -1
1) "2"
2) "1"
3) "4"
4) "5"
5) "6"
```

- set
```shell script
127.0.0.1:6379> sadd set 1 2 3
(integer) 3
127.0.0.1:6379> smembers set
1) "1"
2) "2"
3) "3"
127.0.0.1:6379> sismember set 3
(integer) 1
127.0.0.1:6379> srem set 3
(integer) 1
127.0.0.1:6379> sismember set 3
(integer) 0
127.0.0.1:6379> smembers set
1) "1"
2) "2"
```

- hash
```shell script
127.0.0.1:6379> hset hash sk1 v1 sk2 v2
(integer) 2
127.0.0.1:6379> hgetall hash
1) "sk1"
2) "v1"
3) "sk2"
4) "v2"
127.0.0.1:6379> hset hash sk3 v3
(integer) 1
127.0.0.1:6379> hgetall hash
1) "sk1"
2) "v1"
3) "sk2"
4) "v2"
5) "sk3"
6) "v3"
127.0.0.1:6379> hdel hash sk2
(integer) 1
127.0.0.1:6379> hgetall hash
1) "sk1"
2) "v1"
3) "sk3"
4) "v3"
```

- zset
```shell script
127.0.0.1:6379> zadd zset 3 member1 2 member2 8 member3 0 member4
(integer) 4
127.0.0.1:6379> zrange zset 0 -1 withscores
1) "member4"
2) "0"
3) "member2"
4) "2"
5) "member1"
6) "3"
7) "member3"
8) "8"
127.0.0.1:6379> zrem zset member1
(integer) 1
127.0.0.1:6379> zrange zset 0 -1 withscores
1) "member4"
2) "0"
3) "member2"
4) "2"
5) "member3"
6) "8"
```


