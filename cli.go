package main

import (
	"flag"
	"fmt"
	"go.etcd.io/bbolt"
	"log"
	"os"
)

type CLI struct {
}

func (cli *CLI) Run() {
	cli.validateArgs()

	createBlockCmd := flag.NewFlagSet("create", flag.ExitOnError)
	addressData := createBlockCmd.String("a", "", "your wallet address")

	sendCmd := flag.NewFlagSet("add", flag.ExitOnError)
	fromData := sendCmd.String("f", "", "Source wallet address")
	toData := sendCmd.String("t", "", "Destination wallet address")
	amountData := sendCmd.Int("a", 0, "Amount to send")

	balanceCmd := flag.NewFlagSet("balance", flag.ExitOnError)
	balanceData := balanceCmd.String("a", "", "Balance of wallet address")

	printChainCmd := flag.NewFlagSet("print", flag.ExitOnError)

	switch os.Args[1] {
	case "create":
		err := createBlockCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "send":
		err := sendCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "balance":
		err := balanceCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "print":
		err := printChainCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	default:
		cli.printUsage()
		os.Exit(1)
	}

	if createBlockCmd.Parsed() {
		if *addressData == "" {
			createBlockCmd.Usage()
			os.Exit(1)
		}
		cli.createBlockChain(*addressData)
	}

	if sendCmd.Parsed() {
		if *fromData == "" || *toData == "" || *amountData <= 0 {
			sendCmd.Usage()
			os.Exit(1)
		}
		cli.send(*fromData, *toData, *amountData)
	}

	if balanceCmd.Parsed() {
		if *balanceData == "" {
			balanceCmd.Usage()
			os.Exit(1)
		}
		cli.getBalance(*balanceData)
	}
	if printChainCmd.Parsed() {
		cli.printChain()
	}
}

func (cli *CLI) createBlockChain(address string) {
	bc := CreateBlockChain(address)
	defer func(db *bbolt.DB) {
		err := db.Close()
		if err != nil {
			log.Panic(err)
		}
	}(bc.db)
	fmt.Println("Done!")
}

func (cli *CLI) send(from, to string, amount int) {
	bc := NewBlockChain()
	defer func(db *bbolt.DB) {
		err := db.Close()
		if err != nil {
			log.Panic(err)
		}
	}(bc.db)

	tx := NewUTXOTransaction(from, to, amount, bc)
	bc.MineBlock([]*Transaction{tx})
	fmt.Println("Paid Successfully!")
}

func (cli *CLI) getBalance(address string) {
	bc := NewBlockChain()
	defer func(db *bbolt.DB) {
		err := db.Close()
		if err != nil {
			log.Panic(err)
		}
	}(bc.db)

	UTXOs := bc.FindUTXO(address)
	balance := 0
	for _, utxo := range UTXOs {
		balance += utxo.Value
	}
	fmt.Printf("Balance of '%s': %d\n", address, balance)
}

const usage = `
Usage:
  create -a ADDRESS    			  	- create the new blockchain
  send -f FROM -t TO -a AMOUNT		- Send AMOUNT of coins from FROM address to TO
  balance -a ADDRESS    			- balance of the address
  print               			  	- print all the blocks of the blockchain
`

func (cli *CLI) printUsage() {
	fmt.Println(usage)
}

func (cli *CLI) validateArgs() {
	if len(os.Args) < 2 {
		cli.printUsage()
		os.Exit(1)
	}
}
