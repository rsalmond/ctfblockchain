package main

import "fmt"
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

//var blockserver string = "http://192.168.0.15:5000/chain"
var blockserver string = "http://localhost:5000/chain"

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
	resp, err := http.Get(blockserver)
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
	resp, err := http.Post(blockserver, "application/json", bytes.NewBuffer(blocks_json))
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

func mineChain(c []Block) ([]Block, error) {
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
	}

	log.Info(fmt.Sprintf("Working on block number %d at difficulty rating %d", currentBlock_num, currentBlock.Difficulty))

	currentBlock.set_blockid()
	currentBlock.Previous_hash = fmt.Sprintf("%x", previousBlock.hash())
	difficulty_target := difficultyTarget(currentBlock.Difficulty)

	hashed := make([]byte,0)
	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	hash_count := 0
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
		if start_time + 10 < time.Now().Unix() {
			log.Info(fmt.Sprintf("Hashes per second: %d", hash_count / 10))
			hash_count = 0
			start_time = time.Now().Unix()
		}
	}
	log.Info(fmt.Sprintf("Block: %+v\n", currentBlock))
	return nil, errors.New("Unable to solve this block!")
}

func main() {
	for {
		// fetch blocks
		blockchain, err := getChain()
		if err != nil {
			fmt.Printf("Chain fetcher exited with message: %s\n", err)
			fmt.Println("Exiting.")
			return
		}

		// mine a block
		newchain, err := mineChain(blockchain)
		if err != nil {
			fmt.Printf("Miner exited with message: %s\n", err)
			fmt.Println("Exiting.")
			return
		}

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
