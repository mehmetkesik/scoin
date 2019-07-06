package main

import (
	"bytes"
	"encoding/gob"
	"crypto/sha256"
	"errors"
)

type Block struct {
	Timestamp     int64
	Transactions  []*Transaction
	PrevBlockHash []byte
	Hash          []byte //height hariç diğer verileri birleştirip hash değerini alacak.
	Nonce         int64
	Height        int64
	Address       []byte
	Prize         int64
}

func (self *Block) VerifyBlock() (bool, error) { //Bloğu doğlama işlemini yapacak.
	var lastblock *Block
	if len(blockchain.Blocks) != 0 {
		lastblock = blockchain.Blocks[len(blockchain.Blocks)-1]
	}

	if lastblock == nil {
		if self.Height != 1 {
			return false, nil
		}
	} else {
		if lastblock.Height+1 != self.Height { //Block yüksekliği son bloğundakinden bir fazlamı kontrol eder.
			return false, nil
		}
	}

	hash, err := self.CalculateHash()
	if err != nil {
		return false, err
	}

	if !bytes.Equal(self.Hash, hash) {
		return false, nil
	}

	for _, v := range self.Transactions {
		result, err := blockchain.IsTransactionInBlock(v)
		if err != nil {
			return false, err
		}
		if result {
			return false, nil
		}
	}
	return true, nil
}

func SendBlock(block *Block) error { //bloğu veritabanına ekleyip ağa gönderecek
	verify, err := block.VerifyBlock()
	if err != nil {
		return err
	}
	if !verify {
		return errors.New("Block not verified")
	}

	err = blockchain.AddBlock(block) //Bloğu veritabanına ekliyoruz.
	if err != nil {
		return err
	}

	blockbytes, err := block.Serialize()
	if err != nil {
		return err
	}

	for _, ip := range ips.Ips {
		SendPost(ip, BLOCK, blockbytes)
	}
	return nil
}

func (self *Block) CalculateHash() ([]byte, error) {
	transactionbytes := make([]byte, 0)
	for _, v := range self.Transactions {
		vbytes, err := v.Serialize()
		if err != nil {
			return nil, err
		}
		transactionbytes = bytes.Join([][]byte{transactionbytes}, vbytes)
	}
	headers := bytes.Join([][]byte{IntToBytes(self.Timestamp), transactionbytes, self.PrevBlockHash, IntToBytes(self.Nonce),
		IntToBytes(self.Height), self.Address, IntToBytes(self.Prize)}, []byte{})
	hash := sha256.Sum256(headers)
	return hash[:], nil
}

func (self *Block) Serialize() ([]byte, error) {
	var result bytes.Buffer
	encoder := gob.NewEncoder(&result)
	err := encoder.Encode(self)
	if err != nil {
		return nil, err
	}
	return result.Bytes(), nil
}

func DeserializeBlock(d []byte) (*Block, error) {
	var block Block
	decoder := gob.NewDecoder(bytes.NewReader(d))
	err := decoder.Decode(&block)
	if err != nil {
		return nil, err
	}
	return &block, nil
}