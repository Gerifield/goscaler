package scaler

import (
	"sync"
	"time"
)

type Scaler struct {
	configFile string

	configLock *sync.Mutex
	config     *Config

	stopChan chan struct{}
}

func NewScaler(c string) *Scaler {
	return &Scaler{
		configFile: c,
		configLock: &sync.Mutex{},
		stopChan:   make(chan struct{}),
	}
}

func (s *Scaler) LoadConfig() error {
	c, err := LoadConfig(s.configFile)
	if err != nil {
		return err
	}

	s.configLock.Lock()
	prevImg := s.config.DockerImage // Save the prev. image
	s.config = c
	s.config.PrevDockerImage = prevImg // put it back
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
	return nil
}
