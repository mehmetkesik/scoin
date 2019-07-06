package main

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"golang.org/x/crypto/ripemd160"
	"bytes"
	"crypto/elliptic"
	"crypto/rand"
	"math/big"
	"encoding/binary"
	"crypto/x509"
	"github.com/satori/go.uuid"
	"github.com/boltdb/bolt"
	"encoding/gob"
)

const version = byte(0x00)
const addressChecksumLen = 4

type WalletDB struct {
	Id         []byte
	PrivateKey []byte
	PublicKey  []byte
	Amount     int64
}

type Wallet struct {
	Id         []byte
	PrivateKey ecdsa.PrivateKey
	PublicKey  []byte
	Amount     int64
}

func NewWallet() *Wallet {
	id, _ := uuid.NewV4()
	private, public := newKeyPair()
	wallet := Wallet{id.Bytes(), private, public, 0}
	return &wallet
}

func (w Wallet) GetAddress() []byte { //tek tek işlemler yapılıp adres üretiliyor
	pubKeyHash := HashPubKey(w.PublicKey)

	versionedPayload := append([]byte{version}, pubKeyHash...)
	checksum := checksum(versionedPayload)

	fullPayload := append(versionedPayload, checksum...)
	address := Base58Encode(fullPayload)

	return address
}

func PubkeyToAddress(pubKey []byte) []byte {
	pubKeyHash := HashPubKey(pubKey)

	versionedPayload := append([]byte{version}, pubKeyHash...)
	checksum := checksum(versionedPayload)

	fullPayload := append(versionedPayload, checksum...)
	address := Base58Encode(fullPayload)

	return address
}

func HashPubKey(pubKey []byte) []byte {
	publicSHA256 := sha256.Sum256(pubKey)
	RIPEMD160Hasher := ripemd160.New()
	_, err := RIPEMD160Hasher.Write(publicSHA256[:])
	if err != nil {
		panic(err)
	}
	publicRIPEMD160 := RIPEMD160Hasher.Sum(nil)

	return publicRIPEMD160
}

func ValidateAddress(address string) bool {
	pubKeyHash := Base58Decode([]byte(address))
	actualChecksum := pubKeyHash[len(pubKeyHash)-addressChecksumLen:]
	version := pubKeyHash[0]
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-addressChecksumLen]
	targetChecksum := checksum(append([]byte{version}, pubKeyHash...))
	return bytes.Equal(actualChecksum, targetChecksum)
	//return bytes.Compare(actualChecksum, targetChecksum) == 0
}

func checksum(payload []byte) []byte { //iki defa sha alınıp ilk 4 byte ı döndürülüyor
	firstSHA := sha256.Sum256(payload)
	secondSHA := sha256.Sum256(firstSHA[:])
	return secondSHA[:addressChecksumLen]
}

func newKeyPair() (ecdsa.PrivateKey, []byte) { //yeni bir public,private anahtar çifti üretiliyor.
	curve := elliptic.P256()
	private, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		panic(err)
	}
	pubKey := append(private.PublicKey.X.Bytes(), private.PublicKey.Y.Bytes()...)
	return *private, pubKey
}

var b58Alphabet = []byte("123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz")

func Base58Encode(input []byte) []byte {
	var result []byte

	x := big.NewInt(0).SetBytes(input)

	base := big.NewInt(int64(len(b58Alphabet)))
	zero := big.NewInt(0)
	mod := &big.Int{}

	for x.Cmp(zero) != 0 {
		x.DivMod(x, base, mod)
		result = append(result, b58Alphabet[mod.Int64()])
	}

	if input[0] == 0x00 {
		result = append(result, b58Alphabet[0])
	}

	ReverseBytes(result)

	return result
}

func Base58Decode(input []byte) []byte {
	result := big.NewInt(0)

	for _, b := range input {
		charIndex := bytes.IndexByte(b58Alphabet, b)
		result.Mul(result, big.NewInt(58))
		result.Add(result, big.NewInt(int64(charIndex)))
	}

	decoded := result.Bytes()

	if input[0] == b58Alphabet[0] {
		decoded = append([]byte{0x00}, decoded...)
	}

	return decoded
}

func ReverseBytes(data []byte) {
	for i, j := 0, len(data)-1; i < j; i, j = i+1, j-1 {
		data[i], data[j] = data[j], data[i]
	}
}

func IntToBytes(num int64) []byte {
	buff := new(bytes.Buffer)
	err := binary.Write(buff, binary.BigEndian, num)
	if err != nil {
		panic(err)
	}
	return buff.Bytes()
}

func Signature(privKey ecdsa.PrivateKey, data []byte) []byte {
	r, s, err := ecdsa.Sign(rand.Reader, &privKey, data)
	if err != nil {
		return nil
	}
	signature := append(r.Bytes(), s.Bytes()...)
	return signature
}

func Verify(pubKey []byte, data []byte, signature []byte) bool {
	r := big.Int{}
	s := big.Int{}
	sigLen := len(signature)
	r.SetBytes(signature[:(sigLen / 2)])
	s.SetBytes(signature[(sigLen / 2):])

	x := big.Int{}
	y := big.Int{}
	keyLen := len(pubKey)
	x.SetBytes(pubKey[:(keyLen / 2)])
	y.SetBytes(pubKey[(keyLen / 2):])
	curve := elliptic.P256()
	rawPubKey := ecdsa.PublicKey{Curve: curve, X: &x, Y: &y}
	return ecdsa.Verify(&rawPubKey, data, &r, &s)
}

func WalletToWalletDB(wallet *Wallet) (*WalletDB, error) {
	privKey, err := x509.MarshalECPrivateKey(&wallet.PrivateKey)
	if err != nil {
		return nil, err
	}
	walletdb := new(WalletDB)
	walletdb.Id = wallet.Id
	walletdb.PrivateKey = privKey
	walletdb.PublicKey = wallet.PublicKey
	walletdb.Amount = wallet.Amount
	return walletdb, nil
}

func WalletDBToWallet(walletdb *WalletDB) (*Wallet, error) {
	privateKey, err := x509.ParseECPrivateKey(walletdb.PrivateKey)
	if err != nil {
		return nil, err
	}
	wallet := new(Wallet)
	wallet.Id = walletdb.Id
	wallet.PrivateKey = *privateKey
	wallet.PublicKey = walletdb.PublicKey
	wallet.Amount = walletdb.Amount
	return wallet, nil
}

func (self *Wallet) StartTransaction(address []byte, amount int64) error { //Transfer işlemi başalatacak.
	privKey, err := x509.MarshalECPrivateKey(&self.PrivateKey)
	if err != nil {
		return err
	}
	transaction, err := NewTransaction(privKey, address, amount)
	if err != nil {
		return err
	}
	return StartTransaction(transaction)
}

func AddWallet(wallet *Wallet) error { //Cüzdanı veritabanına ekler.eklenmişse eklemez.
	wallets, err := GetWallets()
	if err != nil {
		return err
	}

	for _, v := range wallets { //Daha önce eklenmiş mi diye kontrol ediyor.
		if bytes.Equal(v.Id, wallet.Id) {
			return nil
		}
	}

	db, err := OpenDB()
	if err != nil {
		return err
	}
	defer db.Close()
	err = db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("wallets"))
		walletdb, err := WalletToWalletDB(wallet)
		if err != nil {
			return err
		}
		wbytes, err := walletdb.Serialize()
		if err != nil {
			return err
		}
		err = bucket.Put(wallet.Id, wbytes)
		return err
	})
	return err
}

func GetWallets() ([]*Wallet, error) { //Veritabanındaki cüzdanları getirecek.
	var wallets []*Wallet
	db, err := OpenDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()
	err = db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("wallets"))
		cursor := bucket.Cursor()
		for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
			walletdb, err := DeserializeWalletDB(v)
			if err != nil {
				return err
			}
			wallet, err := WalletDBToWallet(walletdb)
			if err != nil {
				return err
			}
			wallets = append(wallets, wallet)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return wallets, nil
}

func GetWallet(id []byte) (*Wallet, error) {
	var wallet *Wallet
	db, err := OpenDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()
	err = db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("wallets"))
		wdbbytes := bucket.Get(id)
		if wdbbytes != nil {
			walletdb, err := DeserializeWalletDB(wdbbytes)
			if err != nil {
				return err
			}
			wallet, err = WalletDBToWallet(walletdb)
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return wallet, nil
}

func DeleteWallet(id []byte) error {
	db, err := OpenDB()
	if err != nil {
		return err
	}
	defer db.Close()
	err = db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("wallets"))
		err = bucket.Delete(id)
		return err
	})
	return err
}

func (self *WalletDB) Serialize() ([]byte, error) {
	var result bytes.Buffer
	encoder := gob.NewEncoder(&result)
	err := encoder.Encode(self)
	if err != nil {
		return nil, err
	}
	return result.Bytes(), nil
}

func DeserializeWalletDB(d []byte) (*WalletDB, error) {
	var walletdb WalletDB
	decoder := gob.NewDecoder(bytes.NewReader(d))
	err := decoder.Decode(&walletdb)
	if err != nil {
		return nil, err
	}
	return &walletdb, nil
}
