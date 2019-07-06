package main

import (
	"time"
	"math"
	"errors"
	"math/big"
)

func StartMining(address []byte) (*Block, error) {
	trlen := len(utxo)
	if trlen < 1 {
		return nil, errors.New("insufficient transaction, be at least 3 transactions")
	}
	var trvarmi bool
	if trlen > 100 {
		trlen = 100
		trvarmi = true
	}
	block, err := Mining(utxo[:trlen], address) //mining işlemi sonucunu döndürür.,10 - 100 arasında işlemi yapar.
	if err != nil {
		return nil, err
	}

	//Burada işlenmiş transactionlar veritabanından silinecek.
	for _, v := range block.Transactions {
		err = DeleteUTXOTransaction(v.ID)
		if err != nil {
			return nil, err
		}
	}

	//Veritabanından sildikten sonra kendi üzerindende silecek.
	if trvarmi {
		utxo = utxo[trlen:]
	} else {
		utxo = make([]*Transaction, 0)
	}

	return block, nil
}

func Mining(transactions []*Transaction, address []byte) (*Block, error) { //Aldığı transactionları mine edip block oluşturacak.
	bclen := len(blockchain.Blocks)
	var lastblock *Block

	if bclen == 0 {
		lastblock = new(Block)
	} else {
		lastblock = blockchain.Blocks[bclen-1]
	}

	height := lastblock.Height + 1

	target := big.NewInt(1)
	target.Lsh(target, uint(256-settings.TargetBits)) //2 üzeri 256-targetbits şeklinde bir sayı bulmuş oluruz.
	var hashint big.Int
	var nonce int64
	block := &Block{time.Now().Unix(), transactions, lastblock.Hash, []byte(""),
		nonce, height, address, settings.Prize}
	for nonce < math.MaxInt64 {
		block.Nonce = nonce
		hash, err := block.CalculateHash()
		if err != nil {
			return nil, err
		}
		hashint.SetBytes(hash)
		if hashint.Cmp(target) == -1 { //Bulduğumuz hash değeri hedef sayımızdan daha küçükse istenilen olmuş demektir.
			block.Hash = hash
			return block, nil
		}
		nonce++
	}
	return nil, errors.New("Block nonce not found")
}
