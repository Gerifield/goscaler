package scaler

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/garyburd/redigo/redis"

	"github.com/docker/docker/api/types"
	//"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
)

type Scaler struct {
	configFile string

	configLock *sync.Mutex
	config     *Config

	stopChan chan struct{}

	dockerClient *client.Client
}

func NewScaler(c string) *Scaler {
	dc, _ := client.NewEnvClient()

	return &Scaler{
		configFile:   c,
		configLock:   &sync.Mutex{},
		stopChan:     make(chan struct{}),
		dockerClient: dc,
	}
}

func (s *Scaler) LoadConfig() error {
	c, err := LoadConfig(s.configFile)
	if err != nil {
		return err
	}

	s.configLock.Lock()
	prevImg := s.config.DockerImage.Name // Save the prev. image name
	s.config = c
	s.config.PrevDockerImageName = prevImg // save it back
	s.configLock.Unlock()
	return nil
}

func (s *Scaler) getConfig() *Config {
	s.configLock.Lock()
	defer s.configLock.Unlock()
	return s.config
}

func (s *Scaler) Run() error {
	var err error
	for {
		select {
		case <-s.stopChan:
			return nil
		case <-time.After(s.getConfig().SleepTimeout):
			err = s.doAction()
			if err != nil {
				return err
			}
		}
	}
}

func (s *Scaler) Stop() {
	close(s.stopChan)
}

func (s *Scaler) doAction() error {

	cs, err := s.getActualRunningImages()
	if err != nil {
		return err
	}

	runningNum := len(cs)
	diff := s.getRequiredCountNum() - runningNum

	if diff == 0 {
		return s.cleanupOld()
	}

	if diff < 0 { // Stop diff containers
		s.stopContainers(diff * -1)
	}

	if diff > 0 { // Start diff containers
		s.startContainers(runningNum, diff)
	}

	return s.cleanupOld()
}

func (s *Scaler) cleanupOld() error {
	return nil
}

// TODO: Add error handling and retry
func (s *Scaler) stopContainers(num int) {
	cs, _ := s.getActualRunningImages()

	for _, c := range cs {
		if num > 0 {
			s.stopContainer(c.ID)
			num--
		} else {
			return
		}
	}
}

func (s *Scaler) stopContainer(id string) error {
	return s.dockerClient.ContainerRemove(context.Background(), id, types.ContainerRemoveOptions{Force: true})
}

// TODO: Add error handling and retry
func (s *Scaler) startContainers(runningNum, num int) {
	for i := 0; i < num; i++ {
		s.startContainer(runningNum + i)
	}
}

func (s *Scaler) startContainer(index int) error {
	//s.dockerClient.ContainerCreate(
	//	context.Background(),
	//	container.Config{},
	//	container.HostConfig{},
	//	network.NetworkingConfig{},
	//	fmt.Sprintf("%s-%d", s.config.DockerImage.RunningName, index),
	//)
	return nil
}

func (s *Scaler) getRequiredCountNum() int {
	queueSize := s.getQueueSize()

	for _, sr := range s.config.ScaleRanges {
		if queueSize >= sr.Start && queueSize < sr.End { // In range!
			return sr.Num
		}
	}
}

func (s *Scaler) getQueueSize() int {
	conn := s.config.Redis.Get()
	l, err := redis.Int(conn.Do("LLEN", s.config.QueueName))
	conn.Close()
	if err != nil { // Log error?
		return 0
	}
	return l
}

func (s *Scaler) getActualRunningImages() ([]types.Container, error) {
	//filter := filters.NewArgs()
	//filter.Add("image", getBaseImageName(s.config.DockerImage)) // Filter only given images -> Nop :/
	cs, err := s.dockerClient.ContainerList(context.Background(), types.ContainerListOptions{
	//Filters: filter,
	})
	if err != nil {
		return nil, err
	}

	// Filter the other images
	var ourCs []types.Container
	for _, c := range cs {
		if c.Image == s.config.DockerImage.Name {
			ourCs = append(ourCs, c)
		}
	}
	return ourCs, nil
}

func getBaseImageName(imageName string) string {
	if strings.Contains(imageName, ":") {
		imageName = imageName[:strings.LastIndex(imageName, ":")]
	}
	return imageName
}
