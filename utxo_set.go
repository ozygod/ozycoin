package main

import (
	"encoding/hex"
	"go.etcd.io/bbolt"
	"log"
)

const utxoBucket = "chainstate"

type UTXOSet struct {
	Blockchain *Blockchain
}

func (set UTXOSet) ReIndex() {
	db := set.Blockchain.db
	bucketName := []byte(utxoBucket)

	err := db.Update(func(tx *bbolt.Tx) error {
		err := tx.DeleteBucket(bucketName)
		if err != nil && err != bbolt.ErrBucketNotFound {
			log.Panic(err)
		}
		_, err = tx.CreateBucket(bucketName)

		return err
	})
	if err != nil {
		log.Panic(err)
	}

	UTXO := set.Blockchain.FindUTXO()

	err = db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketName)

		for txID, outs := range UTXO {
			key, _ := hex.DecodeString(txID)
			err := b.Put(key, outs.Serialize())
			if err != nil {
				log.Panic(err)
			}
		}
		return nil
	})
}

func (set UTXOSet) FindSpendableOutputs(publicKeyHash []byte, amount int) (int, map[string][]int) {
	unspentOutputs := make(map[string][]int)
	accumulated := 0

	err := set.Blockchain.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(utxoBucket))
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			txID := hex.EncodeToString(k)
			outs := DeserializeOutputs(v)

			for outIdx, out := range outs.Outputs {
				if out.IsLockedWithKey(publicKeyHash) && accumulated < amount {
					accumulated += out.Value
					unspentOutputs[txID] = append(unspentOutputs[txID], outIdx)
				}
			}
		}
		return nil
	})
	if err != nil {
		log.Panic(err)
	}
	return accumulated, unspentOutputs
}

func (set UTXOSet) FindUTXO(publicKeyHash []byte) []TXOutput {
	var UTXOs []TXOutput

	err := set.Blockchain.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(utxoBucket))
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			outs := DeserializeOutputs(v)

			for _, out := range outs.Outputs {
				if out.IsLockedWithKey(publicKeyHash) {
					UTXOs = append(UTXOs, out)
				}
			}
		}
		return nil
	})
	if err != nil {
		log.Panic(err)
	}
	return UTXOs
}

func (set UTXOSet) Update(block *Block) {
	db := set.Blockchain.db

	err := db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(utxoBucket))
		for _, btx := range block.Transactions {
			if btx.IsCoinbase() == false {
				for _, in := range btx.Vin {
					newOutputs := TXOutputs{}
					outsBytes := b.Get(in.Txid)
					outs := DeserializeOutputs(outsBytes)

					for outIdx, out := range outs.Outputs {
						if outIdx != in.Vout {
							newOutputs.Outputs = append(newOutputs.Outputs, out)
						}
					}
					if len(newOutputs.Outputs) == 0 {
						err := b.Delete(in.Txid)
						if err != nil {
							log.Panic(err)
						}
					} else {
						err := b.Put(in.Txid, newOutputs.Serialize())
						if err != nil {
							log.Panic(err)
						}
					}
				}
			}

			newOutputs := TXOutputs{}
			for _, out := range btx.Vout {
				newOutputs.Outputs = append(newOutputs.Outputs, out)
			}
			err := b.Put(btx.ID, newOutputs.Serialize())
			if err != nil {
				log.Panic(err)
			}
		}
		return nil
	})
	if err != nil {
		log.Panic(err)
	}
}
