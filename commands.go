package main

import (
	"fmt"
	"go.etcd.io/bbolt"
	"log"
	"strconv"
)

func (cli *CLI) printChain(nodeId string) {
	bc := NewBlockChain(nodeId)
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

func (cli *CLI) createBlockChain(nodeId, address string) {
	if !ValidateAddress(address) {
		log.Panic("Address is not valid")
	}
	bc := CreateBlockChain(nodeId, address)
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

func (cli *CLI) send(nodeId, from, to string, amount int, mineNow bool) {
	if !ValidateAddress(from) {
		log.Panic("Sender Address is not valid")
	}
	if !ValidateAddress(to) {
		log.Panic("Recipient Address is not valid")
	}

	bc := NewBlockChain(nodeId)
	set := UTXOSet{bc}
	defer func(db *bbolt.DB) {
		err := db.Close()
		if err != nil {
			log.Panic(err)
		}
	}(bc.db)

	tx := NewUTXOTransaction(nodeId, from, to, amount, &set)

	if mineNow {
		cbTx := NewCoinBaseTX(from, "")
		txs := []*Transaction{cbTx, tx}

		newBlock := bc.MineBlock(txs)
		set.Update(newBlock)
	} else {
		sendTx(knownNodes[0], tx)
	}

	fmt.Println("Paid Successfully!")
}

func (cli *CLI) getBalance(nodeId, address string) {
	if !ValidateAddress(address) {
		log.Panic("Address is not valid")
	}
	bc := NewBlockChain(nodeId)
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

func (cli *CLI) listAddresses(nodeId string) {
	wallets, err := NewWallets(nodeId)
	if err != nil {
		log.Panic(err)
	}
	addresses := wallets.GetAddresses()
	for _, address := range addresses {
		fmt.Println(address)
	}
}

func (cli *CLI) createWallet(nodeId string) {
	wallets, _ := NewWallets(nodeId)
	address := wallets.CreateWallet()
	err := wallets.SaveToFile(nodeId)
	if err != nil {
		log.Panic(err)
	}

	fmt.Println("Your wallet address is", address)
}

func (cli *CLI) startNode(nodeId, minerAddress string) {
	fmt.Printf("Starting Node %s...\n", nodeId)
	if len(minerAddress) > 0 {
		if !ValidateAddress(minerAddress) {
			log.Panic("Address is not valid")
		} else {
			log.Println("Mining is on, address to receive rewards: ", minerAddress)
		}
	}
	StartServer(nodeId, minerAddress)
}
