package main

import (
	"bytes"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net"
)

const protocol = "tcp"
const nodeVersion = 1
const commandLength = 12

// commands
const (
	VERSION    = "version"
	GET_BLOCKS = "getBlocks"
	INV        = "inv"
	GET_DATA   = "getdata"
	BLOCK      = "block"
	TX         = "tx"
	ADDR       = "addr"
)

var nodeAddress string
var miningAddress string
var knownNodes = []string{"localhost:3000"}
var blocksInTransit = [][]byte{}
var mempool = make(map[string]Transaction)

type version struct {
	Version    int
	BestHeight int
	AddrFrom   string
}

type getblocks struct {
	AddrFrom string
}

type inv struct {
	AddrFrom string
	Type     string
	Items    [][]byte
}

type getdata struct {
	AddrFrom string
	Type     string
	ID       []byte
}

type block struct {
	AddrFrom string
	Block    []byte
}

type tx struct {
	AddrFrom    string
	Transaction []byte
}

type addr struct {
	AddrList []string
}

func sendData(addr string, data []byte) {
	conn, err := net.Dial(protocol, addr)
	if err != nil {
		fmt.Printf("%s is not available\n", addr)
		var updateNodes []string

		for _, node := range knownNodes {
			if node != addr {
				updateNodes = append(updateNodes, node)
			}
		}

		knownNodes = updateNodes
		return
	}
	defer func(conn net.Conn) {
		err := conn.Close()
		if err != nil {
			log.Println(err)
		}
	}(conn)

	_, err = io.Copy(conn, bytes.NewReader(data))
	if err != nil {
		log.Panic(err)
	}
}

func sendVersion(addr string, bc *Blockchain) {
	bestHeight := bc.GetBestHeight()
	payload := gobEncode(version{nodeVersion, bestHeight, nodeAddress})

	request := append(commandToBytes(VERSION), payload...)

	sendData(addr, request)
}

func sendGetBlocks(addr string) {
	payload := gobEncode(getblocks{nodeAddress})
	request := append(commandToBytes(GET_BLOCKS), payload...)

	sendData(addr, request)
}

func sendInv(addr, t string, items [][]byte) {
	payload := gobEncode(inv{nodeAddress, t, items})
	request := append(commandToBytes(INV), payload...)

	sendData(addr, request)
}

func sendGetData(addr, t string, hash []byte) {
	payload := gobEncode(getdata{nodeAddress, t, hash})
	request := append(commandToBytes(GET_DATA), payload...)

	sendData(addr, request)
}

func sendBlock(addr string, b *Block) {
	payload := gobEncode(block{nodeAddress, b.Serialize()})
	request := append(commandToBytes(BLOCK), payload...)

	sendData(addr, request)
}

func sendTx(addr string, t *Transaction) {
	payload := gobEncode(tx{nodeAddress, t.Serialize()})
	request := append(commandToBytes(TX), payload...)

	sendData(addr, request)
}

func sendAddr(address string) {
	nodes := addr{knownNodes}
	nodes.AddrList = append(nodes.AddrList, nodeAddress)
	payload := gobEncode(nodes)
	request := append(commandToBytes(ADDR), payload...)

	sendData(address, request)
}

func handleAddr(request []byte, bc *Blockchain) {
	var buffer bytes.Buffer
	var payload addr

	buffer.Write(request[commandLength:])
	dec := gob.NewDecoder(&buffer)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	knownNodes = append(knownNodes, payload.AddrList...)
	fmt.Printf("known nodes: %d\n", len(knownNodes))
	for _, node := range knownNodes {
		sendGetBlocks(node)
	}
}

func handleTx(request []byte, bc *Blockchain) {
	var buffer bytes.Buffer
	var payload tx

	buffer.Write(request[commandLength:])
	dec := gob.NewDecoder(&buffer)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	txData := payload.Transaction
	tx := DeserializeTransaction(txData)
	mempool[hex.EncodeToString(tx.ID)] = tx

	if nodeAddress == knownNodes[0] {
		for _, node := range knownNodes {
			if node != nodeAddress && node != payload.AddrFrom {
				sendInv(node, TX, [][]byte{tx.ID})
			}
		}
	} else {
		if len(mempool) >= 2 && len(miningAddress) > 0 {
		MineTransactions:
			var txs []*Transaction

			for id := range mempool {
				tx := mempool[id]
				if bc.VerifyTransaction(&tx) {
					txs = append(txs, &tx)
				}
			}

			if len(txs) == 0 {
				fmt.Println("All transactions are invalid! Waiting for new ones...")
				return
			}

			cbTx := NewCoinBaseTX(miningAddress, "")
			txs = append(txs, cbTx)

			newBlock := bc.MineBlock(txs)
			set := UTXOSet{bc}
			set.Update(newBlock)

			fmt.Println("New block mined!")

			for _, tx := range txs {
				txID := hex.EncodeToString(tx.ID)
				delete(mempool, txID)
			}

			for _, node := range knownNodes {
				if node != nodeAddress {
					sendInv(node, TX, [][]byte{newBlock.HeaderHash})
				}
			}

			if len(mempool) > 0 {
				goto MineTransactions
			}
		}
	}
}

func handleBlock(request []byte, bc *Blockchain) {
	var buffer bytes.Buffer
	var payload block

	buffer.Write(request[commandLength:])
	dec := gob.NewDecoder(&buffer)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	blockData := payload.Block
	newBlock := DeserializeBlock(blockData)

	fmt.Println("Received a new block")
	// todo verify new block
	bc.AddBlock(newBlock)

	fmt.Printf("Added block %x\n", newBlock.HeaderHash)

	set := UTXOSet{bc}
	set.Update(newBlock)

	if len(blocksInTransit) > 0 {
		blockHash := blocksInTransit[0]
		sendGetData(payload.AddrFrom, BLOCK, blockHash)

		blocksInTransit = blocksInTransit[1:]
	}
}

func handleGetData(request []byte, bc *Blockchain) {
	var buffer bytes.Buffer
	var payload getdata

	buffer.Write(request[commandLength:])
	dec := gob.NewDecoder(&buffer)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	if payload.Type == BLOCK {
		block := bc.GetBlock(payload.ID)

		sendBlock(payload.AddrFrom, &block)
	} else if payload.Type == TX {
		txId := hex.EncodeToString(payload.ID)
		tx := mempool[txId]

		sendTx(payload.AddrFrom, &tx)
	}
}

func handleInv(request []byte, bc *Blockchain) {
	var buffer bytes.Buffer
	var payload inv

	buffer.Write(request[commandLength:])
	dec := gob.NewDecoder(&buffer)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	fmt.Printf("Recevied inventory with %d %s\n", len(payload.Items), payload.Type)

	if payload.Type == BLOCK {
		blocksInTransit = payload.Items

		blockHash := payload.Items[0]
		sendGetData(payload.AddrFrom, BLOCK, blockHash)

		var newTransit [][]byte
		for _, out := range blocksInTransit {
			if bytes.Compare(blockHash, out) != 0 {
				newTransit = append(newTransit, out)
			}
		}
		blocksInTransit = newTransit
	} else if payload.Type == TX {
		txId := payload.Items[0]

		if mempool[hex.EncodeToString(txId)].ID == nil {
			sendGetData(payload.AddrFrom, TX, txId)
		}
	}
}

func handleGetBlocks(request []byte, bc *Blockchain) {
	var buffer bytes.Buffer
	var payload getblocks

	buffer.Write(request[commandLength:])
	dec := gob.NewDecoder(&buffer)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	blocks := bc.GetBlockHashes()
	sendInv(payload.AddrFrom, BLOCK, blocks)
}

func handleVersion(request []byte, bc *Blockchain) {
	var buffer bytes.Buffer
	var payload version

	buffer.Write(request[commandLength:])
	dec := gob.NewDecoder(&buffer)
	err := dec.Decode(&payload)
	if err != nil {
		fmt.Println("Error decoding version")
	}

	myBestHeight := bc.GetBestHeight()
	foreignerBestHeight := payload.BestHeight

	if myBestHeight < foreignerBestHeight {
		sendGetBlocks(payload.AddrFrom)
	} else if myBestHeight > foreignerBestHeight {
		sendVersion(payload.AddrFrom, bc)
	}

	if !nodeIsKnown(payload.AddrFrom) {
		knownNodes = append(knownNodes, payload.AddrFrom)
	}
}

func handleConnection(conn net.Conn, bc *Blockchain) {
	request, err := io.ReadAll(conn)
	if err != nil {
		log.Fatal(err)
	}
	command := bytesToCommand(request[:commandLength])
	fmt.Println("Received command:", command)

	switch command {
	case VERSION:
		handleVersion(request, bc)
		break
	case GET_BLOCKS:
		handleGetBlocks(request, bc)
		break
	case INV:
		handleInv(request, bc)
		break
	case GET_DATA:
		handleGetData(request, bc)
		break
	case BLOCK:
		handleBlock(request, bc)
		break
	case TX:
		handleTx(request, bc)
		break
	case ADDR:
		handleAddr(request, bc)
		break
	default:
		fmt.Println("Unknown command:", command)
	}
	err = conn.Close()
	if err != nil {
		log.Fatal(err)
	}
}

func StartServer(nodeId, minerAddress string) {
	nodeAddress = fmt.Sprintf("localhost:%s", nodeId)
	miningAddress = minerAddress
	ln, err := net.Listen(protocol, nodeAddress)
	defer func(ln net.Listener) {
		err := ln.Close()
		if err != nil {
			panic(err)
		}
	}(ln)
	if err != nil {
		panic(err)
	}

	bc := NewBlockChain(nodeId)

	if nodeAddress != knownNodes[0] {
		sendVersion(knownNodes[0], bc)
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			panic(err)
		}
		go handleConnection(conn, bc)
	}
}

func commandToBytes(command string) []byte {
	var b [commandLength]byte
	for i, c := range command {
		b[i] = byte(c)
	}
	return b[:]
}

func bytesToCommand(bytes []byte) string {
	var command []byte
	for _, b := range bytes {
		if b != 0x0 {
			command = append(command, b)
		}
	}

	return fmt.Sprintf("%s", command)
}

func gobEncode(v interface{}) []byte {
	var buffer bytes.Buffer

	dec := gob.NewEncoder(&buffer)
	err := dec.Encode(v)
	if err != nil {
		fmt.Println("Error encoding version")
	}
	return buffer.Bytes()
}

func nodeIsKnown(addr string) bool {
	for _, knownNode := range knownNodes {
		if addr == knownNode {
			return true
		}
	}
	return false
}
