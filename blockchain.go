package main

type Blockchain struct {
	Blocks []*Block
}

func NewBlockChain() *Blockchain {
	return &Blockchain{[]*Block{NewGenesisBlock()}}
}

func (bc *Blockchain) AddBlock(root []byte) {
	prevBlock := bc.Blocks[len(bc.Blocks)-1]
	block := NewBlock(prevBlock.HeaderHash, root)
	bc.Blocks = append(bc.Blocks, block)
}
