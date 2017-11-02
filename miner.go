package main

import "fmt"
//import "strings"
import "bytes"
import "strconv"
//import "time"
import "io/ioutil"
//import "math/rand"
import "crypto/sha256"
import "net/http"
import "encoding/json"
import "github.com/pkg/errors"

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
	//b.Identifier = strings.ToUpper(fmt.Sprintf("%x", b.hash())[0:32])
	b.Identifier = "CCCB46AF6C3AAE0F48DA3E97AEBA4A4F"
}

func getChain() (blocks []Block, err error) {
	// fetch and decode blockchain state json
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
	resp, err := http.Post(blockserver, "application/json", bytes.NewBuffer(blocks_json))
	if err != nil {
		return errors.Wrap(err, "Failed to post updated blockchain to blockserver.")
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "Blockserver did not validate chain update." + string(body))
	}

	fmt.Println(string(body))
	return nil
}

func difficultyTarget(target int) string {
	/*var difficulty_target []byte
	for i := 0; i < target; i++ {
		difficulty_target = append(difficulty_target, 0)
	}
	return difficulty_target*/
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
	//TODO: stop mining if all blocks are solved

	fmt.Printf("Working on block number %d at difficulty rating %d\n", currentBlock_num, currentBlock.Difficulty)

	currentBlock.set_blockid()
	currentBlock.Previous_hash = fmt.Sprintf("%x", previousBlock.hash())
	difficulty_target := difficultyTarget(currentBlock.Difficulty)

	hashed := make([]byte,0)
	//for i := 0; i < 1000000000; i++ {
	for i := 56955062; i < 56955065; i++ {
		currentBlock.Nonce = i
		hashed = currentBlock.hash()
		hashed_string := fmt.Sprintf("%x", hashed)
		fmt.Printf("%x\n", hashed)
		//fmt.Println(hashed_string)
		if hashed_string[0:currentBlock.Difficulty] == difficulty_target {
			//fmt.Printf("%d\n", hashed)
			//fmt.Println(currentBlock)
			c[currentBlock_num] = currentBlock
			//fmt.Println("miner returning")
			return c, nil
		}
	}
	//fmt.Printf("%v\n", currentBlock)
	return nil, errors.New("Unable to solve this block!")
}

func main() {
	blockchain, err := getChain()
	if err != nil {
		panic(err)
	}
	// for testing 
	blockchain[1].set_blockid()
	//fmt.Println(blockchain[1])

	newchain, err := mineChain(blockchain)
	if err != nil {
		panic(err)
	}

	json_chain, err := json.Marshal(newchain);
	if  err != nil {
		panic(err)
	}
	if err := postChain(json_chain); err != nil {
		panic(err)
	}
}
