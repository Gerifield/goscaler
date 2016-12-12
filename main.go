package main

import (
	"log"

	"github.com/gerifield/scaler"
)

func main() {

	conf, err := scaler.LoadConfig("config.yaml")
	if err != nil {
		log.Println(err)
		return
	}

	s := NewScaler(conf)
	s.Run()
}
