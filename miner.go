package main

import "fmt"
import "os"
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

type Status struct {
	Username string `json:"username"`
	ClientId string `json:"client_id"`
	HashRate int `json:"hashrate"`
}

type Config struct {
	Username string `json:"username"`
	ClientId string `json:"client_id"`
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

func mineChain(c []Block, hashrate_report *chan int, quit *chan bool) ([]Block, error) {
	// mine a block on the chain until it is solved then return the chain for processing back to the server
	var currentBlock Block
	var currentBlock_num int
	var previousBlock Block
	for i, block := range c {
		if block.Nonce == 0 {
			previousBlock = c[i-1]
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
	difficulty_target := difficultyTarget(currentBlock.Difficulty)

	hashed := make([]byte,0)
	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	hash_count := 0
	hashes_per_sec := 0
	report_interval := 30
	start_time := time.Now().Unix()
	// hash block with random nonce values until we hit a hash that has enough 0's 
	// at the beginning to satisfy the difficulty requirement
	for {
		currentBlock.Nonce = random.Int()
		hashed = currentBlock.hash()
		hashed_string := fmt.Sprintf("%x", hashed)
		if hashed_string[0:currentBlock.Difficulty] == difficulty_target {
			c[currentBlock_num] = currentBlock
			return c, nil
		}
		hash_count++
		if start_time + int64(report_interval) < time.Now().Unix() {
			hashes_per_sec = hash_count / report_interval
			log.Info(fmt.Sprintf("Hashes per second: %d", hashes_per_sec))
			*hashrate_report <- hashes_per_sec
			hash_count = 0
			start_time = time.Now().Unix()
			return nil, nil
		}
	}
	log.Info(fmt.Sprintf("Block: %+v\n", currentBlock))
	return nil, errors.New("Unable to solve this block!")
}

func getConfig() (username string, clientid string, config_error error) {
	// either produce a new config file or read an existing one

	config := Config{}
	config_filename := "miner_config.json"
	config_default_username := "your username here"

	if _, err := os.Stat(config_filename); os.IsNotExist(err) {
		// generate a new config file
		config.Username = config_default_username
		random := rand.New(rand.NewSource(time.Now().UnixNano()))
		config.ClientId = strings.ToUpper(fmt.Sprintf("%x", random.Int()))
		config_data, _ := json.Marshal(config)

		file, file_err := os.Create(config_filename)
		if file_err != nil {
			return "", "", errors.Wrap(file_err, "Unable to create config file: " + config_filename)
		}
		defer file.Close()

		_, write_err := file.Write(config_data)
		if write_err != nil {
			return "", "", errors.Wrap(file_err, "Unable to write to config file: " + config_filename)
		}
		return "", "", errors.New("No config file found, a new one has been created. Please configure your username and restart the miner.")
	} else {
		config_data, file_err := ioutil.ReadFile(config_filename)
		if err != nil {
			return "", "", errors.Wrap(file_err, "Unable to read config file: " + config_filename)
		}

		if json_err := json.Unmarshal(config_data, &config); json_err != nil {
			return "", "", errors.Wrap(json_err, "Unable to parse config json.")
		}
		if config.Username == config_default_username {
			return "", "", errors.New("Please edit config file and add a username before restarting miner.")
		}
		return config.Username, config.ClientId, nil
	}
	return "", "", nil
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

func toil(hashrate_report *chan int, quit *chan bool) {
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
		newchain, err := mineChain(blockchain, hashrate_report, quit)
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

func main() {
	username, client_id, err := getConfig()
	if err != nil {
		fmt.Printf("%s\n", err)
		fmt.Println("Exiting.")
		return
	}

	hashrate_report := make(chan int)
	quit := make(chan bool)
	go toil(&hashrate_report, &quit)
	go reportStatus(username, client_id, &hashrate_report)
	select {}
}
