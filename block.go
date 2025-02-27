package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"log"
	"strconv"
	"time"
)

type Block struct {
	Timestamp           int64
	PrevBlockHeaderHash []byte
	HeaderHash          []byte
	Root                []byte
	Transactions        []*Transaction
	Nonce               int
}

func (b *Block) SetHash() {
	timestamp := []byte(strconv.FormatInt(b.Timestamp, 10))
	headers := bytes.Join([][]byte{b.PrevBlockHeaderHash, b.Root, timestamp}, []byte{})
	hash := sha256.Sum256(headers)
	b.HeaderHash = hash[:]
}

func (b *Block) Serialize() []byte {
	var result bytes.Buffer
	encoder := gob.NewEncoder(&result)

	err := encoder.Encode(b)
	if err != nil {
		log.Panic(err)
	}

	return result.Bytes()
}

func (b *Block) HashTransactions() []byte {
	var txs [][]byte
	for _, tx := range b.Transactions {
		txs = append(txs, tx.Serialize())
	}
	mTree := NewMerkleTree(txs)

	return mTree.root.Data
}

func DeserializeBlock(data []byte) *Block {
	var block Block

	decoder := gob.NewDecoder(bytes.NewReader(data))
	err := decoder.Decode(&block)
	if err != nil {
		log.Panic(err)
	}
	return &block
}

func NewBlock(prevBlockHeaderHash []byte, transactions []*Transaction) *Block {
	block := &Block{
		Timestamp:           time.Now().Unix(),
		PrevBlockHeaderHash: prevBlockHeaderHash,
		Transactions:        transactions,
		HeaderHash:          []byte{},
		Nonce:               0,
	}
	//block.SetHash()
	pow := NewPoW(block)
	nonce, hash := pow.Run()

	block.HeaderHash = hash
	block.Nonce = nonce
	return block
}

func NewGenesisBlock(coinBase *Transaction) *Block {
	return NewBlock([]byte{}, []*Transaction{coinBase})
}
