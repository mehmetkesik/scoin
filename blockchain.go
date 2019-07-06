package main

import (
	"errors"
	"github.com/boltdb/bolt"
	"bytes"
	"encoding/gob"
	"math"
)

type Blockchain struct {
	Blocks []*Block
}

func (self *Blockchain) AddBlock(block *Block) error { //Bloğu doğrulama işlemi yapıp eğer doğruysa blockchaine ekleyecek.
	verify, err := block.VerifyBlock()
	if err != nil {
		return err
	}
	if !verify {
		return errors.New("Block not verified")
	}

	if int64(len(self.Blocks)) >= block.Height {
		return errors.New("Block already added")
	}

	db, err := OpenDB()
	if err != nil {
		return err
	}
	defer db.Close()
	err = db.Update(func(tx *bolt.Tx) error { //Bloğun eklendiği yer.
		bucket := tx.Bucket([]byte("blockchain"))
		blockbytes, err := block.Serialize()
		if err != nil {
			return err
		}
		err = bucket.Put(IntToBytes(block.Height), blockbytes)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}

	self.Blocks = append(self.Blocks, block) //Blogu veritabanına ekledikten sonra kendi üzerinede ekliyor.

	//her 5 blokta bir sıfır durumunu güncellemek için.dakikada 1 blok eklenecek şekilde güncellenecek.
	if block.Height%5 == 0 {
		var prevBlockHeight int64
		if block.Height == 5 {
			prevBlockHeight = 1
		} else {
			prevBlockHeight = block.Height - 5
		}
		prevBlock := blockchain.Blocks[prevBlockHeight]
		difference := block.Timestamp - prevBlock.Timestamp
		//Duruma göre zorluk %150 arttırılır ya da %75  azaltılır.
		if difference > (60 * 11) { // zaman farkı 11 dakikadan büyükse zorluk azaltılır.
			settings.TargetBits = int(math.Round(float64(settings.TargetBits) * 0.75))
		} else if difference < (60 * 9) { // zaman farkı 9 dakikadan küçükse zorluk arttırılır.
			settings.TargetBits = int(math.Round(float64(settings.TargetBits) * 1.5))
		}
	}

	//istenilen bit sayısının veritabanında günceller.
	err = db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("settings"))
		sbytes, err := settings.Serialize()
		if err != nil {
			return err
		}
		err = bucket.Put([]byte("settings"), sbytes)
		if err != nil {
			return err
		}
		return nil
	})

	return err
}

func (self *Blockchain) GetAddressAmount(address []byte) (int64, error) {
	var amount int64
	for _, block := range self.Blocks {
		if bytes.Equal(block.Address, address) {
			amount += block.Prize
		}
		for _, transaction := range block.Transactions {
			if bytes.Equal(transaction.Address, address) {
				amount += transaction.Amount
			}
			senderaddress := PubkeyToAddress(transaction.PubKey)
			if bytes.Equal(senderaddress, address) {
				amount -= transaction.Amount
			}
		}
	}
	return amount, nil
}

func (self *Blockchain) IsTransactionInBlock(transaction *Transaction) (bool, error) {
	for _, v := range self.Blocks {
		for _, v2 := range v.Transactions {
			if bytes.Equal(transaction.ID, v2.ID) {
				return true, nil
			}
		}
	}
	return false, nil
}

func (self *Blockchain) Serialize() ([]byte, error) {
	var result bytes.Buffer
	encoder := gob.NewEncoder(&result)
	err := encoder.Encode(self)
	if err != nil {
		return nil, err
	}
	return result.Bytes(), nil
}

func DeserializeBlockchain(d []byte) (*Blockchain, error) {
	var blockchain Blockchain
	decoder := gob.NewDecoder(bytes.NewReader(d))
	err := decoder.Decode(&blockchain)
	if err != nil {
		return nil, err
	}
	return &blockchain, nil
}
