package main

import (
	"fmt"
	"strconv"
)

func main() {
	bc := NewBlockChain()

	bc.AddBlock([]byte("Send 1 BTC to Ivan"))
	bc.AddBlock([]byte("Send 2 more BTC to Ivan"))

	for _, block := range bc.Blocks {
		fmt.Printf("Prev. hash: %x\n", block.PrevBlockHeaderHash)
		fmt.Printf("Data: %s\n", block.Root)
		fmt.Printf("Hash: %x\n", block.HeaderHash)
		pow := NewPoW(block)
		fmt.Printf("PoW: %s\n", strconv.FormatBool(pow.Verify()))
		fmt.Println()
	}
}
