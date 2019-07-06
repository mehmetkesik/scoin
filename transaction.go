package main

import (
	"bytes"
	"github.com/satori/go.uuid"
	"crypto/x509"
	"encoding/gob"
	"github.com/boltdb/bolt"
	"errors"
)

type Transaction struct {
	ID      []byte
	Sign    []byte
	PubKey  []byte
	Address []byte
	Amount  int64
}

func AddTransaction(transaction *Transaction) error { //Transactionı veritabanına ekleyecek.
	addressamount, err := blockchain.GetAddressAmount(PubkeyToAddress(transaction.PubKey))
	if err != nil {
		return err
	}
	if addressamount < transaction.Amount { //Hesapta yeterli para olup olmadığını kontrol ediyor.
		return errors.New("there is not enough amount in address")
	}

	if !ValidateAddress(string(transaction.Address)) {
		return errors.New("Address not verified")
	}

	if !transaction.Verify() { //Transactionı doğrulama işlemi yapar.
		return errors.New("Transaction not verified")
	}

	for _, v := range utxo {
		if bytes.Equal(v.ID, transaction.ID) {
			return errors.New("Transaction already added")
		}
	}

	db, err := OpenDB()
	if err != nil {
		return err
	}
	defer db.Close()

	err = db.Update(func(tx *bolt.Tx) error {
		utxo := tx.Bucket([]byte("utxo"))
		tbytes, err := transaction.Serialize()
		if err != nil {
			return err
		}
		err = utxo.Put(transaction.ID, tbytes)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}

	utxo = append(utxo, transaction) //Kendü üzerinde güncelleme yapıyor.

	return nil
}

func StartTransaction(transaction *Transaction) error { //Transactionı veritabanına ekleyip ağa gönderecek.
	err := AddTransaction(transaction)
	if err != nil {
		return err
	}

	transactionbytes, err := transaction.Serialize()
	if err != nil {
		return err
	}

	for _, ip := range ips.Ips {
		SendPost(ip, TRANSACTION, transactionbytes)
	}
	return nil
}

func (self *Transaction) Verify() bool {
	data := bytes.Join([][]byte{self.Address}, IntToBytes(self.Amount))
	return Verify(self.PubKey, data, self.Sign) //Gönderilecek adresi imzaladığımız için onu doğruluyoruz.
}

func NewTransaction(privateKey []byte, address []byte, amount int64) (*Transaction, error) {
	privKey, err := x509.ParseECPrivateKey(privateKey)
	if err != nil {
		return nil, err
	}
	data := bytes.Join([][]byte{address}, IntToBytes(amount))
	sign := Signature(*privKey, data)
	id, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}
	pubKey := append(privKey.PublicKey.X.Bytes(), privKey.PublicKey.Y.Bytes()...)
	t := Transaction{id.Bytes(), sign, pubKey, address, amount}
	return &t, nil
}

func DeleteUTXOTransaction(id []byte) error {
	db, err := OpenDB()
	if err != nil {
		return err
	}
	defer db.Close()
	err = db.Update(func(tx *bolt.Tx) error {
		utxo := tx.Bucket([]byte("utxo"))
		return utxo.Delete(id)
	})

	//Burada kendi üzerinde güncelleme yapıyor.
	newutxo := make([]*Transaction, 0)
	for _, tr := range utxo {
		if !bytes.Equal(tr.ID, id) {
			newutxo = append(newutxo, tr)
		}
	}
	utxo = newutxo

	return err
}

func (self *Transaction) Serialize() ([]byte, error) {
	var result bytes.Buffer
	encoder := gob.NewEncoder(&result)
	err := encoder.Encode(self)
	if err != nil {
		return nil, err
	}
	return result.Bytes(), nil
}

func DeserializeTransaction(d []byte) (*Transaction, error) {
	var transaction Transaction
	decoder := gob.NewDecoder(bytes.NewReader(d))
	err := decoder.Decode(&transaction)
	if err != nil {
		return nil, err
	}
	return &transaction, nil
}
