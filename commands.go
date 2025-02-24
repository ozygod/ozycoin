package main

import (
	"fmt"
	"strconv"
)

func (cli *CLI) printChain() {
	bc := NewBlockChain()
	iterator := bc.Iterator()

	for {
		block := iterator.Next()

		fmt.Printf("Prev. hash: %x\n", block.PrevBlockHeaderHash)
		fmt.Printf("Data: %s\n", block.Root)
		fmt.Printf("Hash: %x\n", block.HeaderHash)
		fmt.Printf("Transactions: %s\n", block.Transactions)
		pow := NewPoW(block)
		fmt.Printf("PoW: %s\n", strconv.FormatBool(pow.Verify()))
		fmt.Println()

		if len(block.PrevBlockHeaderHash) == 0 {
			break
		}
	}
}
