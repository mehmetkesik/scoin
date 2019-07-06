package main

import (
	"encoding/gob"
	"bytes"
)

type Settings struct {
	TargetBits int
	Prize      int64
}

func Server(data []byte) ([]byte, error) {
	postdata, err := DeserializePostData(data)
	if err != nil {
		return nil, err
	}

	switch postdata.Id {
	case BLOCK:
		block, err := DeserializeBlock(postdata.Data)
		if err != nil {
			return nil, err
		}
		verify, err := block.VerifyBlock()
		if err != nil {
			return nil, err
		}
		if verify {
			for _, v := range block.Transactions {
				DeleteUTXOTransaction(v.ID) //Bloktanı transactionları veritabanından siler.
			}
			SendBlock(block) //Block varsa göndermez hata döndürür.
		}
		break
	case TRANSACTION:
		transaction, err := DeserializeTransaction(postdata.Data)
		if err != nil {
			return nil, err
		}
		if transaction.Verify() {
			StartTransaction(transaction) //Transaction varsa göndermez hata döndürür.
		}
		break
	case IPADDRESS:
		senderhost := string(postdata.Data)
		if IsValidIp(senderhost) {
			AddIps([]string{senderhost})
		}
		return ips.Serialize()
		break
	case BLOCKCHAIN:
		return blockchain.Serialize()
		break;
	case SETTING:
		return settings.Serialize()
		break
	}
	return nil, nil
}

func (self *Settings) Serialize() ([]byte, error) {
	var result bytes.Buffer
	encoder := gob.NewEncoder(&result)
	err := encoder.Encode(self)
	if err != nil {
		return nil, err
	}
	return result.Bytes(), nil
}

func DeserializeSettings(d []byte) (*Settings, error) {
	var settings Settings
	decoder := gob.NewDecoder(bytes.NewReader(d))
	err := decoder.Decode(&settings)
	if err != nil {
		return nil, err
	}
	return &settings, nil
}
