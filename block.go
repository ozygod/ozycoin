package main

import (
	"bytes"
	"crypto/sha256"
	"strconv"
	"time"
)

type Block struct {
	Timestamp           int64
	PrevBlockHeaderHash []byte
	HeaderHash          []byte
	Root                []byte
}

func NewBlock(prevBlockHeaderHash []byte, root []byte) *Block {
	block := &Block{
		Timestamp:           time.Now().Unix(),
		PrevBlockHeaderHash: prevBlockHeaderHash,
		Root:                root,
		HeaderHash:          []byte{},
	}
	block.SetHash()
	return block
}

func (b *Block) SetHash() {
	timestamp := []byte(strconv.FormatInt(b.Timestamp, 10))
	headers := bytes.Join([][]byte{b.PrevBlockHeaderHash, b.Root, timestamp}, []byte{})
	hash := sha256.Sum256(headers)
	b.HeaderHash = hash[:]
}

func NewGenesisBlock() *Block {
	return NewBlock([]byte{}, []byte("Genesis Block"))
}
