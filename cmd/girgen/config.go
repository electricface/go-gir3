package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
)

type config struct {
	Black     []string `json:"black"`
	CIncludes []string `json:"cIncludes"`
}

func loadConfig(filename string, cfg *config) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return json.Unmarshal(data, cfg)
}
