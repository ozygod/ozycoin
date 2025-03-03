package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

type CLI struct {
}

func (cli *CLI) Run() {
	cli.validateArgs()

	nodeID := os.Getenv("NODE_ID")
	if nodeID == "" {
		fmt.Printf("NODE_ID env. var is not set!")
		os.Exit(1)
	}

	startNodeCmd := flag.NewFlagSet("start", flag.ExitOnError)
	minerAddress := startNodeCmd.String("m", "", "miner address")

	createWalletCmd := flag.NewFlagSet("createwallet", flag.ExitOnError)

	createBlockCmd := flag.NewFlagSet("create", flag.ExitOnError)
	addressData := createBlockCmd.String("a", "", "your wallet address")

	sendCmd := flag.NewFlagSet("add", flag.ExitOnError)
	fromData := sendCmd.String("f", "", "Source wallet address")
	toData := sendCmd.String("t", "", "Destination wallet address")
	amountData := sendCmd.Int("a", 0, "Amount to send")
	mineData := sendCmd.Bool("m", false, "Mine immediately on the same node")

	balanceCmd := flag.NewFlagSet("balance", flag.ExitOnError)
	balanceData := balanceCmd.String("a", "", "Balance of wallet address")

	printChainCmd := flag.NewFlagSet("print", flag.ExitOnError)
	listAddressesCmd := flag.NewFlagSet("list", flag.ExitOnError)

	switch os.Args[1] {
	case "start":
		err := startNodeCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "createwallet":
		err := createWalletCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
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
	case "list":
		err := listAddressesCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	default:
		cli.printUsage()
		os.Exit(1)
	}

	if startNodeCmd.Parsed() {
		nodeID = os.Getenv("NODE_ID")
		if nodeID == "" {
			startNodeCmd.Usage()
			os.Exit(1)
		}
		cli.startNode(nodeID, *minerAddress)
	}

	if createWalletCmd.Parsed() {
		cli.createWallet(nodeID)
	}

	if createBlockCmd.Parsed() {
		if *addressData == "" {
			createBlockCmd.Usage()
			os.Exit(1)
		}
		cli.createBlockChain(nodeID, *addressData)
	}

	if sendCmd.Parsed() {
		if *fromData == "" || *toData == "" || *amountData <= 0 {
			sendCmd.Usage()
			os.Exit(1)
		}
		cli.send(nodeID, *fromData, *toData, *amountData, *mineData)
	}

	if balanceCmd.Parsed() {
		if *balanceData == "" {
			balanceCmd.Usage()
			os.Exit(1)
		}
		cli.getBalance(nodeID, *balanceData)
	}
	if printChainCmd.Parsed() {
		cli.printChain(nodeID)
	}
	if listAddressesCmd.Parsed() {
		cli.listAddresses(nodeID)
	}
}

const usage = `
Usage:
  start -m MINERADDRESS  			- start a new node
  create -a ADDRESS    			  	- create the new blockchain
  createwallet  	   			  	- create the new wallet address
  list 	   			  				- list all wallet address
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
