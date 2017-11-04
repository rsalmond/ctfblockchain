package main

import "fmt"
import "os"
import "runtime"
import "strings"
import "bytes"
import "strconv"
import "time"
import "io/ioutil"
import "math/rand"
import "crypto/sha256"
import "net/http"
import "encoding/json"
import "github.com/pkg/errors"
import (log "github.com/sirupsen/logrus")

var blockserver string = "https://rob.salmond.ca/mine"
//var blockserver string = "http://localhost:5000/mine"

type Status struct {
	Username string `json:"username"`
	ClientId string `json:"client_id"`
	HashRate int `json:"hashrate"`
}

type Config struct {
	Username string `json:"username"`
	ClientId string `json:"client_id"`
	MaxWorkers int `json:"max_workers"`
}

type Block struct {
	Identifier string `json:"identifier"`
	Nonce int `json:"nonce"`
	Data string `json:"data"`
	Previous_hash string `json:"previous_hash"`
	Difficulty int `json:"difficulty"`
}

func (b *Block) hash() []byte {
	h := sha256.New()
	h.Write([]byte(b.Identifier))
	h.Write([]byte(strconv.Itoa(b.Nonce)))
	h.Write([]byte(b.Data))
	h.Write([]byte(b.Previous_hash))
	return h.Sum(nil)
}

func (b *Block) set_blockid() {
	// set a consistent block identifier by hashing what we have to start with
	b.Identifier = strings.ToUpper(fmt.Sprintf("%x", b.hash())[0:32])
}

func getChain() (blocks []Block, err error) {
	// fetch and decode blockchain state json
	log.Info("Getting blockchain from server.")
	endpoint := fmt.Sprintf("%s/chain", blockserver)
	resp, err := http.Get(endpoint)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to connect to blockserver: " + blockserver)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to retrieve blocks from blockserver: " + string(body))
	}

	if err := json.Unmarshal(body, &blocks); err != nil {
		return nil, errors.Wrap(err, "Failed to parse block JSON: " + string(body))
	}

	return blocks, nil
}

func postChain(blocks_json []byte) error {
	//TODO: some better checking on this response
	log.Info("Posting update to blockserver.")
	endpoint := fmt.Sprintf("%s/chain", blockserver)
	resp, err := http.Post(endpoint, "application/json", bytes.NewBuffer(blocks_json))
	if err != nil {
		return errors.Wrap(err, "Failed to post updated blockchain to blockserver.")
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "Blockserver did not validate chain update." + string(body))
	}

	log.Info(fmt.Sprintf("Blockserver responded: %s", string(body)))
	return nil
}

func difficultyTarget(target int) string {
	// difficulty is based on the number of 0's at the beginning of the hash
	// so build a buffer of 0's to compare to the hash
	var target_buffer bytes.Buffer
	for i := 0; i < target; i++ {
		target_buffer.WriteString("0")
	}
	return target_buffer.String()
}

func hashWorker(result *chan Block, end_hashrate *chan int, current_block *Block, worker_id int) {
	hashed := make([]byte,0)
	hash_count := 0
	hashes_per_sec := 0
	report_interval := 30

	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	difficulty_target := difficultyTarget(current_block.Difficulty)

	start_time := time.Now().Unix()
	// hash block with random nonce values until we hit a hash that has enough 0's 
	// at the beginning to satisfy the difficulty requirement
	for {
		current_block.Nonce = random.Int()
		hashed = current_block.hash()
		hashed_string := fmt.Sprintf("%x", hashed)
		if hashed_string[0:current_block.Difficulty] == difficulty_target {
			*result <- *current_block
		}
		hash_count++
		if start_time + int64(report_interval) < time.Now().Unix() {
			hashes_per_sec = hash_count / report_interval
			log.Info(fmt.Sprintf("Worker: %d hashes per second: %d", worker_id, hashes_per_sec))
			*end_hashrate <- hashes_per_sec
			return
		}
	}
}

func mineChain(c *[]Block, hashrate_report *chan int, quit *chan bool, max_workers int) ([]Block, error) {
	// mine a block on the chain until it is solved then return the chain for processing back to the server
	var currentBlock Block
	var currentBlock_num int
	var previousBlock Block
	for i, block := range *c {
		if block.Nonce == 0 {
			previousBlock = (*c)[i-1]
			currentBlock_num = i
			currentBlock = block
			break
		}
	}
	// stop mining if all blocks are solved
	if currentBlock.Difficulty == 0 {
		return nil, errors.New("No blocks found to solve.")
		close(*quit)
	}

	log.Info(fmt.Sprintf("Working on block number %d at difficulty rating %d", currentBlock_num, currentBlock.Difficulty))

	currentBlock.set_blockid()
	currentBlock.Previous_hash = fmt.Sprintf("%x", previousBlock.hash())

	//func hashWorker(result *chan Block, hashrate_report *chan int, current_block *Block) {
	result := make(chan Block)
	end_hashrate := make(chan int)
	ended_count := 0
	total_hashrate := 0
	for {
		// launch an async hashworker to try random nonces
		for i := 0; i < max_workers; i++ {
			log.Debug(fmt.Sprintf("Launching worker id: %d", i))
			go hashWorker(&result, &end_hashrate, &currentBlock, i)
		}

		for j := 0; j < max_workers; j++ {
			select {
			// return if we mine a block so it can be sent to the blockserver and we'll fetch the next block
			case retval := <-result:
				(*c)[currentBlock_num] = retval
				return *c, nil
			// if all the hashworkers ended their report interval, continue and spawn new ones
			case rate := <-end_hashrate:
				ended_count++
				total_hashrate += rate
				if ended_count == max_workers {
					log.Info(fmt.Sprintf("Total hashes per second: %d", total_hashrate))
					*hashrate_report <- total_hashrate
					ended_count = 0
					total_hashrate = 0
					return nil, nil
				}
			}
		}
	}
	log.Info(fmt.Sprintf("Block: %+v\n", currentBlock))
	return nil, errors.New("Unable to solve this block!")
}

func getConfig(username string) (*Config, error) {
	// either produce a new config file or read an existing one

	config := new(Config)

	config_filename := "miner_config.json"

	config_default_username := "your username here"
	if username != "" {
		config_default_username = username
	}

	if _, err := os.Stat(config_filename); os.IsNotExist(err) {
		// generate a new config file
		config.Username = config_default_username
		config.MaxWorkers = runtime.NumCPU()
		random := rand.New(rand.NewSource(time.Now().UnixNano()))
		config.ClientId = strings.ToUpper(fmt.Sprintf("%x", random.Int()))
		config_data, _ := json.Marshal(config)

		file, file_err := os.Create(config_filename)
		if file_err != nil {
			return nil, errors.Wrap(file_err, "Unable to create config file: " + config_filename)
		}
		defer file.Close()

		_, write_err := file.Write(config_data)
		if write_err != nil {
			return nil, errors.Wrap(file_err, "Unable to write to config file: " + config_filename)
		}
		return nil, errors.New("No config file found, a new one has been created. Please configure your username and restart the miner.")
	} else {
		config_data, file_err := ioutil.ReadFile(config_filename)
		if err != nil {
			return nil, errors.Wrap(file_err, "Unable to read config file: " + config_filename)
		}

		if json_err := json.Unmarshal(config_data, &config); json_err != nil {
			return nil, errors.Wrap(json_err, "Unable to parse config json.")
		}
		if config.Username == config_default_username {
			return nil, errors.New("Please edit config file and add a username before restarting miner.")
		}
		return config, nil
	}
	return nil, nil
}

func postStatus(status_json []byte) error {
	log.Info("Posting status to blockserver.")
	endpoint := fmt.Sprintf("%s/status", blockserver)
	resp, err := http.Post(endpoint, "application/json", bytes.NewBuffer(status_json))
	if err != nil {
		return errors.Wrap(err, "Failed to post status to blockserver.")
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "Blockserver did not accept status update." + string(body))
	}

	log.Info(fmt.Sprintf("Blockserver responded: %s", string(body)))
	return nil
}

func toil(hashrate_report *chan int, quit *chan bool, max_workers int) {
	// slave away in the mines
	for {
		// fetch blocks
		blockchain, err := getChain()
		if err != nil {
			log.Warn(fmt.Sprintf("Chain fetcher exited with message: %s", err))
			log.Info("Sleeping until retry.")
			time.Sleep(10 * time.Second)
			continue
		}

		// mine a block
		newchain, err := mineChain(&blockchain, hashrate_report, quit, max_workers)
		if err != nil {
			log.Info(fmt.Sprintf("Miner exited with message: %s", err))
			log.Info("Exiting.")
			return
		}

		if newchain != nil {
			// json encode mined block
			json_chain, err := json.Marshal(newchain);
			if  err != nil {
				panic(err)
			}

			// post mined block to blockserver
			if err := postChain(json_chain); err != nil {
				panic(err)
			}
		}
	}
}

func reportStatus(username string, client_id string, hashrate_report *chan int) {
	// send a status report every time the miner writes current hashrate to the report channel
	for {
		status := Status{}
		status.Username = username
		status.ClientId = client_id
		status.HashRate = <-*hashrate_report
		status_data, _ := json.Marshal(status)

		if err := postStatus(status_data); err != nil {
			log.Warn(fmt.Sprintf("Error posting status to blockserver: %s", err))
			continue
		}
	}
}

func printUsage() {
	fmt.Println("Usage:")
	fmt.Printf("\t%s <username>", os.Args[1])
	fmt.Printf("\tGenerate a new config file using the specified username.")
	fmt.Printf("\t%s", os.Args[1])
	fmt.Printf("\tStart mining using existing config file or generate a default config if none is present.")
}

func needHelp(param string) (n bool) {
	switch param {
	case "-h":
		return true
	case "--help":
		return true
	case "help":
		return true
	default:
		return false
	}
}

func main() {
	var config *Config
	var err error

	// fiddle about with args
	if len(os.Args) == 2 {
		// display help and exit
		if needHelp(os.Args[1]) {
			printUsage()
			return
		// if arg doesnt look like help accept it as username for config
		} else {
			config, err = getConfig(os.Args[1])
		}
	// just go ahead and load config then
	} else {
		config, err = getConfig("")
	}

	if err != nil {
		fmt.Printf("%s\n", err)
		fmt.Println("Exiting.")
		return
	}

	log.Info(fmt.Sprintf("Detected %d available CPU cores.", runtime.NumCPU()))
	log.Info(fmt.Sprintf("Launching with configuration=> username: %s, client_id: %s, max_workers: %d", config.Username, config.ClientId, config.MaxWorkers))
	hashrate_report := make(chan int)
	quit := make(chan bool)
	go toil(&hashrate_report, &quit, config.MaxWorkers)
	go reportStatus(config.Username, config.ClientId, &hashrate_report)
	select {}
}
