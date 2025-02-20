package main

import "fmt"

func main() {
	fmt.Println("Hello World")
	bc := NewBlockChain()

	bc.AddBlock([]byte("Send 1 BTC to Ivan"))
	bc.AddBlock([]byte("Send 2 more BTC to Ivan"))

	for _, block := range bc.Blocks {
		fmt.Printf("Prev. hash: %x\n", block.PrevBlockHeaderHash)
		fmt.Printf("Data: %s\n", block.Root)
		fmt.Printf("Hash: %x\n", block.HeaderHash)
		fmt.Println()
	}
}
