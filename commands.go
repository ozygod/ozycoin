package main

import (
	"fmt"
	"strconv"
)

func (cli *CLI) addBlock(data string) {
	cli.bc.AddBlock([]byte(data))
	fmt.Println("Successfully added block")
}

func (cli *CLI) printChain() {
	iterator := cli.bc.Iterator()

	for {
		block := iterator.Next()

		fmt.Printf("Prev. hash: %x\n", block.PrevBlockHeaderHash)
		fmt.Printf("Data: %s\n", block.Root)
		fmt.Printf("Hash: %x\n", block.HeaderHash)
		pow := NewPoW(block)
		fmt.Printf("PoW: %s\n", strconv.FormatBool(pow.Verify()))
		fmt.Println()

		if len(block.PrevBlockHeaderHash) == 0 {
			break
		}
	}
}
