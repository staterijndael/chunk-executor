package main

import (
	config2 "chunk-executor/config"
	"chunk-executor/models"
	"encoding/json"
	"github.com/BurntSushi/toml"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func main() {
	var config config2.Config
	if _, err := toml.DecodeFile("./config/config.toml", &config); err != nil {
		log.Panic(err.Error())
	}

	file, err := os.Open("chunk/chunks.json")
	if err != nil {
		log.Panic(err.Error())
	}

	byteValue, err := ioutil.ReadAll(file)
	if err != nil {
		log.Panic(err.Error())
	}

	var chunksBase models.ChunksBase

	err = json.Unmarshal(byteValue, &chunksBase)
	if err != nil {
		log.Panic(err.Error())
	}

	for _, chunk := range chunksBase.Chunks {
		params := make(map[string]string)
		err = json.Unmarshal(chunk.Params, &params)
		if err != nil {
			log.Panic(err.Error())
		}

		params["server"] = config.Dns.Hosts[rand.Intn(len(config.Dns.Hosts))]

		chunk.Params, err = json.Marshal(params)
		if err != nil {
			log.Panic(err.Error())
		}
	}

	outFile, _ := json.MarshalIndent(chunksBase, "", " ")

	_ = ioutil.WriteFile("chunk/chunks.json", outFile, 0644)
}
