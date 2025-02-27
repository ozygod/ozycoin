package main

import (
	"fmt"
	"go.etcd.io/bbolt"
	"log"
	"strconv"
)

func (cli *CLI) printChain() {
	bc := NewBlockChain()
	iterator := bc.Iterator()

	for {
		block := iterator.Next()

		fmt.Printf("============ Block %x ============\n", block.HeaderHash)
		fmt.Printf("Prev. block: %x\n", block.PrevBlockHeaderHash)
		pow := NewPoW(block)
		fmt.Printf("PoW: %s\n\n", strconv.FormatBool(pow.Verify()))
		for _, tx := range block.Transactions {
			fmt.Println(tx)
		}
		fmt.Printf("\n\n")

		if len(block.PrevBlockHeaderHash) == 0 {
			break
		}
	}
}

func (cli *CLI) createBlockChain(address string) {
	if !ValidateAddress(address) {
		log.Panic("Address is not valid")
	}
	bc := CreateBlockChain(address)
	defer func(db *bbolt.DB) {
		err := db.Close()
		if err != nil {
			log.Panic(err)
		}
	}(bc.db)

	set := UTXOSet{bc}
	set.ReIndex()

	fmt.Println("Done!")
}

func (cli *CLI) send(from, to string, amount int) {
	if !ValidateAddress(from) {
		log.Panic("Sender Address is not valid")
	}
	if !ValidateAddress(to) {
		log.Panic("Recipient Address is not valid")
	}

	bc := NewBlockChain()
	set := UTXOSet{bc}
	defer func(db *bbolt.DB) {
		err := db.Close()
		if err != nil {
			log.Panic(err)
		}
	}(bc.db)

	tx := NewUTXOTransaction(from, to, amount, &set)
	newBlock := bc.MineBlock([]*Transaction{tx})
	set.Update(newBlock)
	fmt.Println("Paid Successfully!")
}

func (cli *CLI) getBalance(address string) {
	if !ValidateAddress(address) {
		log.Panic("Address is not valid")
	}
	bc := NewBlockChain()
	set := UTXOSet{bc}
	defer func(db *bbolt.DB) {
		err := db.Close()
		if err != nil {
			log.Panic(err)
		}
	}(bc.db)

	publicKeyHash := GetPublicKeyHash(address)
	UTXOs := set.FindUTXO(publicKeyHash)
	balance := 0
	for _, utxo := range UTXOs {
		balance += utxo.Value
	}
	fmt.Printf("Balance of '%s': %d\n", address, balance)
}

func (cli *CLI) listAddresses() {
	wallets, err := NewWallets()
	if err != nil {
		log.Panic(err)
	}
	addresses := wallets.GetAddresses()
	for _, address := range addresses {
		fmt.Println(address)
	}
}

func (cli *CLI) createWallet() {
	wallets, _ := NewWallets()
	address := wallets.CreateWallet()
	err := wallets.SaveToFile()
	if err != nil {
		log.Panic(err)
	}

	fmt.Println("Your wallet address is", address)
}
