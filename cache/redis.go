package cache

import (
	"flag"
	"time"

	"github.com/garyburd/redigo/redis"
)

var (
	redisPool     *redis.Pool
	redisServer   = flag.String("redisServer", ":6379", "the redis's server")
	redisPassword = flag.String("redisPassword", "", "the redis's password")
	redisDB       = flag.String("redisDB", "0", "the redis's db")
)

func GetRedisConn() redis.Conn {
	conn := redisPool.Get()
	return conn
}
func InitRedis() error {

	redisPool = &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", *redisServer)
			if err != nil {
				return nil, err
			}

			if *redisPassword != "" {
				if _, err := c.Do("AUTH", redisPassword); err != nil {
					c.Close()
					return nil, err
				}
			}

			if *redisDB != "0" {
				if _, err := c.Do("select", redisDB); err != nil {
					c.Close()
					return nil, err
				}

			}

			return c, nil

		},
		// 这个空闲链接测试连接的过程 不到一分钟 直接返回给pool,一分钟之后测试连接，成功返回给pool
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			if time.Since(t) < time.Minute {
				return nil
			}

			_, err := c.Do("PING")
			return err
		},
	}

	//fmt.Printf("%+v", redisPool)
	return nil
}
