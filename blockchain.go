package main

import (
	"encoding/hex"
	"go.etcd.io/bbolt"
	"log"
	"os"
)

const dbFile = "blockchain.db"
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
	db, err := bbolt.Open(dbFile, 0600, nil)
	if err != nil {
		log.Panic(err)
	}

	err = db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))

		genesis := NewGenesisBlock(NewCoinBaseTX(address, genesisCoinbaseData))

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

func (bc *Blockchain) FindUnspentTransaction(address string) []*Transaction {
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
					for _, spentOut := range spentTXOs[txID] {
						if spentOut == outIdx {
							continue Outputs
						}
					}
				}

				// 输出地址为address，表示address未花掉的钱
				if out.CanBeUnlockedWith(address) {
					unspentTxs = append(unspentTxs, tx)
				}
			}

			if !tx.IsCoinbase() {
				// 输入地址为address的，表示address已经花掉的钱
				for _, in := range tx.Vin {
					if in.CanUnlockOutputWith(address) {
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

func (bc *Blockchain) FindSpendableOutputs(address string, amount int) (int, map[string][]int) {
	unspentOutputs := make(map[string][]int)
	unspentTXs := bc.FindUnspentTransaction(address)
	accumulated := 0

Work:
	for _, tx := range unspentTXs {
		txId := hex.EncodeToString(tx.ID)
		for outIdx, out := range tx.Vout {
			if out.CanBeUnlockedWith(address) && accumulated < amount {
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

func (bc *Blockchain) FindUTXO(address string) []TXOutput {
	var UTXOs []TXOutput
	unspentTXs := bc.FindUnspentTransaction(address)
	for _, tx := range unspentTXs {
		for _, out := range tx.Vout {
			if out.CanBeUnlockedWith(address) {
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
