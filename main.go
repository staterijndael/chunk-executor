package main

import (
	"bytes"
	config2 "chunk-executor/config"
	"encoding/json"
	"fmt"
	"github.com/BurntSushi/toml"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

type ServerInstance struct {
	Name         string
	ChunkCh      chan *Chunk
	FreeChunks   int
	FreeChunksMx sync.Mutex
	Client       *ssh.Client
	CurrentTask  int
}

func (s *ServerInstance) ExecuteCmd(command string) (string, error) {
	var stdoutBuf bytes.Buffer

	session, err := s.Client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	session.Stdout = &stdoutBuf
	err = session.Run(command)
	if err != nil {
		return "", err
	}

	return stdoutBuf.String(), nil
}

func (s *ServerInstance) GetLoadAverage() (string, error) {
	return s.ExecuteCmd("uptime")
}

type Chunk struct {
	ChunkID int `json:"chunk_id"`
	X       int
	Y       int
	Z       string
}

type ChunksBase struct {
	Chunks []*Chunk
}

func main() {
	var config config2.Config
	if _, err := toml.DecodeFile("./config/config.toml", &config); err != nil {
		fmt.Println(err)
		return
	}

	files, err := ioutil.ReadDir("./logs")
	if err != nil {
		log.Fatal(err)
	}

	var maxTaskID int
	for _, file := range files {
		taskID, err := strconv.Atoi(file.Name()[:len(file.Name())-4])
		if err != nil {
			log.Fatal(err)
		}

		if taskID > maxTaskID {
			maxTaskID = taskID
		}
	}

	logFile, err := os.Create("./logs/" + strconv.Itoa(maxTaskID+1) + ".txt")
	if err != nil {
		log.Panic(err.Error())
	}

	var workDoneCounter int32

	servers := make([]*ServerInstance, 0, len(config.Servers))

	var wg sync.WaitGroup
	var serverMutex sync.Mutex

	wg.Add(len(config.Servers))
	for serverName, serverCfg := range config.Servers {
		go func(servName string, servCfg *config2.Server) {
			defer wg.Done()

			conn, err := ssh.Dial("tcp", servCfg.IP+":22", &ssh.ClientConfig{
				User: servCfg.User,
				Auth: []ssh.AuthMethod{
					ssh.Password(servCfg.Password),
				},
				HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			})
			if err != nil {
				logFile.Write([]byte(err.Error()))
				log.Panic("server " + servName + "panic: " + err.Error())
			}

			serverInstance := &ServerInstance{
				Name:         servName,
				ChunkCh:      make(chan *Chunk),
				Client:       conn,
				FreeChunks:   servCfg.MaxChunks,
				FreeChunksMx: sync.Mutex{},
				CurrentTask:  0,
			}

			//// getting all tasks directories
			//output, err := serverInstance.ExecuteCmd("find . -maxdepth 1 -type d -regex \".*[0-9]\"")
			//if err != nil {
			//	log.Panic("server " + servName + "panic: " + err.Error())
			//}
			//taskDirs := strings.Split(output, "\n")
			//var maxTaskID int
			//for _, taskDir := range taskDirs {
			//	if len(taskDir) != 0 {
			//		currentTaskID, err := strconv.Atoi(taskDir[2:])
			//		if err != nil {
			//			if err != nil {
			//				log.Panic("server " + servName + "panic: " + err.Error())
			//			}
			//		}
			//		if currentTaskID > maxTaskID {
			//			maxTaskID = currentTaskID
			//		}
			//	}
			//}

			_, err = serverInstance.ExecuteCmd("mkdir -p " + strconv.Itoa(maxTaskID+1) + " && cd " + strconv.Itoa(maxTaskID+1))
			if err != nil {
				logFile.Write([]byte(err.Error()))
				log.Panic("server " + servName + "panic: " + err.Error())
			}

			serverInstance.CurrentTask = maxTaskID + 1

			go serverInstance.startHandlingChunks(logFile, &workDoneCounter)

			serverMutex.Lock()
			servers = append(servers, serverInstance)
			serverMutex.Unlock()
		}(serverName, serverCfg)
	}

	wg.Wait()

	file, err := os.Open("./chunks.json")
	if err != nil {
		log.Panic(err.Error())
	}

	byteValue, err := ioutil.ReadAll(file)
	if err != nil {
		log.Panic(err.Error())
	}

	var chunksBase ChunksBase

	err = json.Unmarshal(byteValue, &chunksBase)
	if err != nil {
		log.Panic(err.Error())
	}

	for _, chunk := range chunksBase.Chunks {
		for _, serverInstance := range servers {
			serverInstance.FreeChunksMx.Lock()
			if serverInstance.FreeChunks > 0 {
				serverInstance.ChunkCh <- chunk
				serverInstance.FreeChunks--
			}
			serverInstance.FreeChunksMx.Unlock()
		}
	}

	for int(workDoneCounter) != len(chunksBase.Chunks) {
		continue
	}

	fmt.Println("work done!")
}

func (s *ServerInstance) startHandlingChunks(logFile *os.File, workDoneCounter *int32) {
	for i := 0; i < s.FreeChunks; i++ {
		go func() {
			for {
				chunk := <-s.ChunkCh

				command := fmt.Sprintf("«-x %v -y %v -z %v -outputdir ./{%v}/{%v}/ > ./{%v}/{%v}.log", chunk.X, chunk.Y, chunk.Z, s.CurrentTask, chunk.ChunkID, s.CurrentTask, chunk.ChunkID)

				output, err := s.ExecuteCmd(command)
				if err != nil {
					LA, LaErr := s.GetLoadAverage()
					if LaErr != nil {
						LA = LaErr.Error()
					}

					_, err = logFile.Write([]byte(generateLogOutput(chunk.ChunkID, time.Now(), err.Error(), LA)))
					if err != nil {
						log.Panic(err.Error())
					}

					atomic.AddInt32(workDoneCounter, 1)
					continue
				}
				LA, err := s.GetLoadAverage()
				if err != nil {
					LA = err.Error()
				}

				_, err = logFile.Write([]byte(generateLogOutput(chunk.ChunkID, time.Now(), output, LA)))
				if err != nil {
					log.Panic(err.Error())
				}

				s.FreeChunksMx.Lock()
				s.FreeChunks++
				s.FreeChunksMx.Unlock()
				atomic.AddInt32(workDoneCounter, 1)
			}
		}()
	}
}

func generateLogOutput(ChunkID int, Time time.Time, Message string, LoadAverage string) string {
	return fmt.Sprintf("ChunkID: %v, Time: %v, Message: %v, LA: %v", ChunkID, Time.String(), Message, LoadAverage)
}
