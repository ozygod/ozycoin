package main

import (
	"fmt"
	"go.etcd.io/bbolt"
	"log"
)

const dbFile = "blockchain.db"
const blocksBucket = "blocks"

type Blockchain struct {
	tip []byte
	db  *bbolt.DB
}

func NewBlockChain() *Blockchain {
	var tip []byte
	db, err := bbolt.Open(dbFile, 0600, nil)
	if err != nil {
		log.Panic(err)
	}

	err = db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))

		if b == nil {
			fmt.Println("No existing blockchain found. Creating a new one...")
			genesis := NewGenesisBlock()

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
		} else {
			tip = b.Get([]byte("l"))
		}

		return nil
	})

	if err != nil {
		log.Panic(err)
	}

	return &Blockchain{tip, db}
}

func (bc *Blockchain) AddBlock(root []byte) {
	var tip []byte
	err := bc.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		tip = b.Get([]byte("l"))

		return nil
	})

	block := NewBlock(tip, root)

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

type BlockchainIterator struct {
	currentHash []byte
	db          *bbolt.DB
}

func (bc *Blockchain) Iterator() *BlockchainIterator {
	return &BlockchainIterator{bc.tip, bc.db}
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
