package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gerifield/goscaler/scaler"
)

func main() {
	sigs := make(chan os.Signal, 1)

	signal.Notify(sigs, syscall.SIGUSR2, syscall.SIGINT, syscall.SIGKILL)
	s := scaler.NewScaler("config.yaml")
	err := s.LoadConfig()
	if err != nil {
		log.Println(err)
		return
	}

	// Config reload listener
	go func(s *scaler.Scaler) {
		for {
			sig := <-sigs
			if sig == syscall.SIGUSR2 {
				log.Println("Reload config")
				err := s.LoadConfig()
				if err != nil {
					log.Println(err)
					s.Stop()
				}
			} else {
				log.Println("Stop...")
				s.Stop()
			}
		}
	}(s)

	log.Println("Started")
	err = s.Run()
	if err != nil {
		log.Println(err)
	}
}
