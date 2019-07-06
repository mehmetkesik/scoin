package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"

	"github.com/boltdb/bolt"
)

var (
	blockchain *Blockchain
	utxo       []*Transaction
	ips        *Ips
	wallets    []*Wallet
	settings   *Settings

	dbname string = "scoin.db"
	mining bool   = false
	host   string = ""
)

func web(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" { //Server için
		data, err := ioutil.ReadAll(r.Body)
		if err != nil {
			panic(err)
		}
		result, err := Server(data)
		if err != nil {
			panic(err)
		}
		w.Write(result)
		return
	}

	//Burası anasayfa için
	webHome(w, r)
}

func main() {
	blockchain = new(Blockchain)
	utxo = make([]*Transaction, 0)
	ips = new(Ips)
	wallets = make([]*Wallet, 0)
	settings = new(Settings)

	if _, err := os.Stat(dbname); os.IsNotExist(err) {
		fmt.Println("init is working..")
		Init()
		fmt.Println("init finished")
	} else {
		fmt.Println("database is uploading..")
		Upload()
		fmt.Println("database uploaded")
	}

	http.Handle("/html/", http.StripPrefix("/html/", http.FileServer(http.Dir("html"))))
	http.HandleFunc("/", web)
	http.HandleFunc("/wallets", webWallets)
	http.HandleFunc("/blockchain", webBlockchain)
	http.HandleFunc("/blocktransactions", webBlockTransactions)
	http.HandleFunc("/ips", webIps)
	http.HandleFunc("/utxo", webUtxo)
	http.HandleFunc("/settings", webSettings)
	http.HandleFunc("/mining", webMining)

	sayi := 8080
	for {
		host = "locahost:" + strconv.Itoa(sayi)
		fmt.Println("info:", "trying", host)
		if err := http.ListenAndServe(":"+strconv.Itoa(sayi), nil); err != nil {
			fmt.Println("error port: ", sayi)
		}
		if sayi > 9000 {
			fmt.Println("connection failed")
			break
		}
		sayi += 1
	}
}

func OpenDB() (*bolt.DB, error) {
	db, err := bolt.Open(dbname, 0600, nil)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func Init() error { //Burada ilk başta oluşturulması gereken veritabanları oluşturulacak.
	db, err := OpenDB()
	if err != nil {
		return err
	}
	defer db.Close()
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucket([]byte("blockchain"))
		if err != nil {
			return err
		}
		_, err = tx.CreateBucket([]byte("utxo"))
		if err != nil {
			return err
		}
		_, err = tx.CreateBucket([]byte("wallets"))
		if err != nil {
			return err
		}
		_, err = tx.CreateBucket([]byte("ips"))
		if err != nil {
			return err
		}
		settingsBucket, err := tx.CreateBucket([]byte("settings"))
		if err != nil {
			return err
		}
		settings.TargetBits = 16
		settings.Prize = 50
		sbytes, err := settings.Serialize()
		if err != nil {
			return err
		}
		settingsBucket.Put([]byte("settings"), sbytes)
		return nil
	})
	return err
}

func Upload() error {
	db, err := OpenDB()
	if err != nil {
		return err
	}
	defer db.Close()

	err = db.View(func(tx *bolt.Tx) error { //Bloğun aynı yüksekliğine ait bir block daha önce eklenmiş mi ?
		bucket := tx.Bucket([]byte("blockchain"))
		err = bucket.ForEach(func(k, v []byte) error {
			block, err := DeserializeBlock(v)
			if err != nil {
				return err
			}
			blockchain.Blocks = append(blockchain.Blocks, block)
			return nil
		})
		if err != nil {
			return err
		}

		bucket = tx.Bucket([]byte("utxo"))
		err = bucket.ForEach(func(k, v []byte) error {
			transaction, err := DeserializeTransaction(v)
			if err != nil {
				return err
			}
			utxo = append(utxo, transaction)
			return nil
		})
		if err != nil {
			return err
		}

		bucket = tx.Bucket([]byte("wallets"))
		err = bucket.ForEach(func(k, v []byte) error {
			walletdb, err := DeserializeWalletDB(v)
			if err != nil {
				return err
			}
			wallet, err := WalletDBToWallet(walletdb)
			if err != nil {
				return err
			}
			wallets = append(wallets, wallet)
			return nil
		})
		if err != nil {
			return err
		}

		bucket = tx.Bucket([]byte("ips"))
		err = bucket.ForEach(func(k, v []byte) error {
			ips.Ips = append(ips.Ips, string(v))
			return nil
		})
		if err != nil {
			return err
		}

		bucket = tx.Bucket([]byte("settings"))
		err = bucket.ForEach(func(k, v []byte) error {
			settings, err = DeserializeSettings(v)
			if err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			return err
		}

		return nil
	})
	return err
}
