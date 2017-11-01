package main

import "fmt"
import "strconv"
import "io/ioutil"
import "crypto/sha256"
import "net/http"
import "encoding/json"
import "github.com/pkg/errors"

var blockserver string = "http://192.168.0.15:5000/chain"

type Block struct {
	Identifier string `json:"identifier"`
	Nonce int `json:"nonce"`
	Data string `json:"data"`
	Previous_hash string `json:"previous_hash"`
}

func (b *Block) hash() []byte {
	h := sha256.New()
	h.Write([]byte(b.Identifier))
	h.Write([]byte(strconv.Itoa(b.Nonce)))
	h.Write([]byte(b.Data))
	h.Write([]byte(b.Previous_hash))
	return h.Sum(nil)
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

func mineChain(c []Block) {
	var currentBlock Block
	for _, block := range c {
		if block.Identifier == "" {
			currentBlock = block
			break
		}
	}
	fmt.Println(currentBlock)
	//TODO: populate blockid and previous hash and iterate over nonce
}

func main() {
	blockchain, err := getChain()
	if err != nil {
		panic(err)
	}
	//fmt.Println(blockchain[0])
	//fmt.Printf("%x", blockchain[0].hash())
	mineChain(blockchain)
}
