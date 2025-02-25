package main

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"go.etcd.io/bbolt"
	"log"
	"os"
)

const dbFile = "ozycoin.db"
const blocksBucket = "blocks"
const genesisCoinbaseData = "The Times 03/Jan/2009 Chancellor on brink of second bailout for banks"

type Blockchain struct {
	tip []byte
	db  *bbolt.DB
}

func doExists() bool {
	if _, err := os.Stat(dbFile); os.IsNotExist(err) {
		return false
	}
	return true
}

func NewBlockChain() *Blockchain {
	if !doExists() {
		log.Println("No existing blockchain found. Creating a new first")
		os.Exit(1)
	}
	var tip []byte
	db, err := bbolt.Open(dbFile, 0600, nil)
	if err != nil {
		log.Panic(err)
	}

	err = db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		tip = b.Get([]byte("l"))

		return nil
	})

	if err != nil {
		log.Panic(err)
	}

	return &Blockchain{tip, db}
}

func CreateBlockChain(address string) *Blockchain {
	if doExists() {
		log.Println("Blockchain already exists")
		os.Exit(1)
	}

	var tip []byte

	genesis := NewGenesisBlock(NewCoinBaseTX(address, genesisCoinbaseData))

	db, err := bbolt.Open(dbFile, 0600, nil)
	if err != nil {
		log.Panic(err)
	}

	err = db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))

		b, err := tx.CreateBucket([]byte(blocksBucket))
		if err != nil {
			log.Panic(err)
		}

		err = b.Put(genesis.HeaderHash, genesis.Serialize())
		if err != nil {
			log.Panic(err)
		}

		err = b.Put([]byte("l"), genesis.HeaderHash)
		if err != nil {
			log.Panic(err)
		}

		tip = genesis.HeaderHash

		return nil
	})

	if err != nil {
		log.Panic(err)
	}

	return &Blockchain{tip, db}
}

func (bc *Blockchain) MineBlock(transactions []*Transaction) {
	var tip []byte

	for _, tx := range transactions {
		if bc.VerifyTransaction(tx) == false {
			log.Panic("Transaction is not valid")
		}
	}

	err := bc.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		tip = b.Get([]byte("l"))

		return nil
	})

	block := NewBlock(tip, transactions)

	err = bc.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))

		err := b.Put(block.HeaderHash, block.Serialize())
		if err != nil {
			log.Panic(err)
		}

		err = b.Put([]byte("l"), block.HeaderHash)
		if err != nil {
			log.Panic(err)
		}

		bc.tip = block.HeaderHash

		return nil
	})

	if err != nil {
		log.Panic(err)
	}
}

func (bc *Blockchain) Iterator() *BlockchainIterator {
	return &BlockchainIterator{bc.tip, bc.db}
}

func (bc *Blockchain) FindTransaction(ID []byte) (Transaction, error) {
	iterator := bc.Iterator()

	for {
		block := iterator.Next()
		for _, tx := range block.Transactions {
			if bytes.Compare(tx.ID, ID) == 0 {
				return *tx, nil
			}
		}
		if len(block.PrevBlockHeaderHash) == 0 {
			break
		}
	}
	return Transaction{}, errors.New("transaction not found")
}

func (bc *Blockchain) SignTransaction(tx *Transaction, privateKey ecdsa.PrivateKey) {
	prevTXs := make(map[string]Transaction)

	for _, in := range tx.Vin {
		prevTX, err := bc.FindTransaction(in.Txid)
		if err != nil {
			log.Panic(err)
		}
		prevTXs[hex.EncodeToString(prevTX.ID)] = prevTX
	}

	tx.Sign(privateKey, prevTXs)
}

func (bc *Blockchain) VerifyTransaction(tx *Transaction) bool {
	prevTXs := make(map[string]Transaction)

	for _, in := range tx.Vin {
		prevTX, err := bc.FindTransaction(in.Txid)
		if err != nil {
			log.Panic(err)
		}
		prevTXs[hex.EncodeToString(prevTX.ID)] = prevTX
	}

	return tx.Verify(prevTXs)
}

func (bc *Blockchain) FindUnspentTransaction(publicKeyHash []byte) []*Transaction {
	var unspentTxs []*Transaction
	spentTXOs := make(map[string][]int)
	iterator := bc.Iterator()

	for {
		block := iterator.Next()

		for _, tx := range block.Transactions {
			txID := hex.EncodeToString(tx.ID)

		Outputs:
			for outIdx, out := range tx.Vout {
				// 检查当前输出是否在下个交易被花掉
				if spentTXOs[txID] != nil {
					for _, spentOutIdx := range spentTXOs[txID] {
						if spentOutIdx == outIdx {
							continue Outputs
						}
					}
				}

				// 输出地址为address，表示address未花掉的钱
				if out.IsLockedWithKey(publicKeyHash) {
					unspentTxs = append(unspentTxs, tx)
				}
			}

			if !tx.IsCoinbase() {
				// 输入地址为address的，表示address已经花掉的钱
				for _, in := range tx.Vin {
					if in.UsesKey(publicKeyHash) {
						inTxID := hex.EncodeToString(in.Txid)
						spentTXOs[inTxID] = append(spentTXOs[inTxID], in.Vout)
					}
				}
			}
		}

		if len(block.PrevBlockHeaderHash) == 0 {
			break
		}
	}
	return unspentTxs
}

func (bc *Blockchain) FindSpendableOutputs(publicKeyHash []byte, amount int) (int, map[string][]int) {
	unspentOutputs := make(map[string][]int)
	unspentTXs := bc.FindUnspentTransaction(publicKeyHash)
	accumulated := 0

Work:
	for _, tx := range unspentTXs {
		txId := hex.EncodeToString(tx.ID)
		for outIdx, out := range tx.Vout {
			if out.IsLockedWithKey(publicKeyHash) && accumulated < amount {
				accumulated += out.Value
				unspentOutputs[txId] = append(unspentOutputs[txId], outIdx)

				if accumulated >= amount {
					break Work
				}
			}
		}
	}
	return accumulated, unspentOutputs
}

func (bc *Blockchain) FindUTXO(publicKeyHash []byte) []TXOutput {
	var UTXOs []TXOutput
	unspentTXs := bc.FindUnspentTransaction(publicKeyHash)
	for _, tx := range unspentTXs {
		for _, out := range tx.Vout {
			if out.IsLockedWithKey(publicKeyHash) {
				UTXOs = append(UTXOs, out)
			}
		}
	}

	return UTXOs
}

type BlockchainIterator struct {
	currentHash []byte
	db          *bbolt.DB
}

func (it *BlockchainIterator) Next() *Block {
	var block *Block

	err := it.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		block = DeserializeBlock(b.Get(it.currentHash))
		return nil
	})

	if err != nil {
		log.Panic(err)
	}

	it.currentHash = block.PrevBlockHeaderHash
	return block
}
