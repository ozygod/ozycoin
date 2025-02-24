package main

import (
	"go.etcd.io/bbolt"
	"log"
)

func main() {
	bc := NewBlockChain()
	defer func(db *bbolt.DB) {
		err := db.Close()
		if err != nil {
			log.Panic(err)
		}
	}(bc.db)

	cli := CLI{bc}
	cli.Run()
}
