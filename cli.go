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

	createWalletCmd := flag.NewFlagSet("createwallet", flag.ExitOnError)

	createBlockCmd := flag.NewFlagSet("create", flag.ExitOnError)
	addressData := createBlockCmd.String("a", "", "your wallet address")

	sendCmd := flag.NewFlagSet("add", flag.ExitOnError)
	fromData := sendCmd.String("f", "", "Source wallet address")
	toData := sendCmd.String("t", "", "Destination wallet address")
	amountData := sendCmd.Int("a", 0, "Amount to send")

	balanceCmd := flag.NewFlagSet("balance", flag.ExitOnError)
	balanceData := balanceCmd.String("a", "", "Balance of wallet address")

	printChainCmd := flag.NewFlagSet("print", flag.ExitOnError)
	listAddressesCmd := flag.NewFlagSet("list", flag.ExitOnError)

	switch os.Args[1] {
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

	if createWalletCmd.Parsed() {
		cli.createWallet()
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
	if listAddressesCmd.Parsed() {
		cli.listAddresses()
	}
}

const usage = `
Usage:
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
