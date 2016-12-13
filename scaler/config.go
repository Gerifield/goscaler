package scaler

import (
	"io/ioutil"
	"time"

	"github.com/garyburd/redigo/redis"
	"gopkg.in/yaml.v2"
)

type ConfigFile struct {
	RedisServer  string            `yaml:"redisServer"`
	QueueName    string            `yaml:"queueName"`
	SleepTimeout string            `yaml:"sleepTimeout"`
	DockerImage  DockerImageConfig `yaml:"dockerImage"`
	ScaleRanges  []ScaleRange      `yaml:"ranges"`
}

type ScaleRange struct {
	Start int `yaml:"start"` // Range start
	End   int `yaml:"end"`   // Range end
	Num   int `yaml:"num"`   // Container num to run
}

type Config struct {
	Redis               *redis.Pool
	QueueName           string
	DockerImage         DockerImageConfig
	PrevDockerImageName string
	SleepTimeout        time.Duration
	ScaleRanges         []ScaleRange
}

type DockerImageConfig struct {
	Name          string   `yaml:"name"`
	Command       string   `yaml:"command"`
	NetworkMode   string   `yaml:"networkMode"`
	Volumes       []string `yaml:"volumes"`
	RestartPolicy string   `yaml:"restartPolicy"`
	Repo          string   `yaml:"repo"`
	RunningName   string   `yaml:"runningName"`
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
	dur, err := time.ParseDuration(conf.SleepTimeout)
	if err != nil {
		return nil, err
	}

	return &Config{
		Redis:        pool,
		QueueName:    conf.QueueName,
		DockerImage:  conf.DockerImage,
		ScaleRanges:  conf.ScaleRanges,
		SleepTimeout: dur,
	}, nil
}

func newPool(server, password string) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     10,
		MaxActive:   60,
		IdleTimeout: 240 * time.Second,
		Wait:        true,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", server, redis.DialConnectTimeout(10*time.Second), redis.DialReadTimeout(10*time.Second), redis.DialWriteTimeout(10*time.Second))
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
