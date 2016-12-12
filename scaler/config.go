package scaler

import (
	"io/ioutil"

	"github.com/garyburd/redigo/redis"
	"gopkg.in/yaml.v2"
)

type ConfigFile struct {
	RedisServer string       `yaml:"redisServer"`
	QueueName   string       `yaml:"queueName"`
	DockerImage string       `yaml:"dockerImage"`
	ScaleRanges []ScaleRange `yaml:"ranges"`
}

type ScaleRange struct {
	Start int `yaml:"start"` // Range start
	End   int `yaml:"end"`   // Range end
	Num   int `yaml:"num"`   // Container num to run
}

type Config struct {
	Redis       *redis.Pool
	QueueName   string
	DockerImage string
	ScaleRanges []ScaleRange
}

func LoadConfig(file string) (*Config, error) {
	b, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}

	var conf ConfigFile
	err = yaml.Unmarshal(b, &conf)
	if err != nil {
		return nil, err
	}

	pool := newPool(conf.RedisServer, "")

	return &Config{
		Redis:       pool,
		QueueName:   conf.QueueName,
		DockerImage: conf.DockerImage,
		ScaleRanges: conf.ScaleRanges,
	}, nil
}

func newPool(server, password string) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     10,
		MaxActive:   60,
		IdleTimeout: 240 * time.Second,
		Wait:        true, // Wait for the connection pool, no connection pool exhausted error
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", server, redis.DialConnectTimeout(5*time.Second), redis.DialReadTimeout(5*time.Second), redis.DialReadTimeout(5*time.Second))
			if err != nil {
				return nil, err
			}
			if password != "" {
				if _, err := c.Do("AUTH", password); err != nil {
					c.Close()
					return nil, err
				}
			}
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			if time.Since(t) < 5*time.Second { // ping only the older than 5 sec. connections
				return nil
			}
			_, err := c.Do("PING")
			return err
		},
	}
}
