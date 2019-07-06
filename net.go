package main

import (
	"github.com/boltdb/bolt"
	"github.com/satori/go.uuid"
	"bytes"
	"encoding/gob"
	"net/http"
	"io/ioutil"
	"errors"
	"strings"
	"strconv"
	"net"
)

type PostData struct {
	Id   int
	Data []byte
}

type Ips struct {
	Ips []string
}

const (
	BLOCK       = 0
	TRANSACTION = 1
	IPADDRESS   = 2
	BLOCKCHAIN  = 3
	SETTING     = 4
	ACTIVE      = 5
)

func (self *Blockchain) UpdateBlockchain() error {
	var bc *Blockchain
	for _, v := range ips.Ips {
		body, err := SendPost(v, BLOCKCHAIN, nil)
		if err == nil {
			bc, err = DeserializeBlockchain(body)
			if err != nil {
				return err
			}
			break
		}
	}

	if bc == nil {
		return nil
	}

	newblocks := bc.Blocks[len(self.Blocks):]

	deletetransactions := make([]*Transaction, 0)
	//Burada blockchaini güncelleme işlemi yapacak.
	db, err := OpenDB()
	if err != nil {
		return err
	}
	err = db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("blockchain"))
		for _, block := range newblocks {
			bbytes, err := block.Serialize()
			if err != nil {
				return err
			}
			err = bucket.Put(IntToBytes(block.Height), bbytes)
			if err != nil {
				return err
			}
			for _, transaction := range block.Transactions {
				deletetransactions = append(deletetransactions, transaction)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	db.Close()

	for _, transaction := range deletetransactions {
		DeleteUTXOTransaction(transaction.ID) //Blocklardaki transactionları utxo'dan siliyor.
	}

	self.Blocks = bc.Blocks //Veritabanına ekledikten sonra kendi üzerinde güncelledi.

	return nil
}

func UpdateSettings() error {
	var s *Settings
	var body []byte
	var err error
	for _, v := range ips.Ips {
		body, err = SendPost(v, SETTING, nil)
		if err == nil {
			s, err = DeserializeSettings(body)
			if err != nil {
				return err
			}
			break
		}
	}

	if s == nil {
		return nil
	}

	db, err := OpenDB()
	if err != nil {
		return err
	}
	defer db.Close()
	err = db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("settings"))
		err = bucket.Put([]byte("settings"), body)
		return err
	})
	if err != nil {
		return err
	}

	settings = s //Burada kendi üzerinde güncelleme yapıyor.

	return nil
}

func FindNodes() error { //nodeları bulacak sabit ipye başvuru yapacak.
	var controlips []string
	var tut []string
	tut = ips.Ips
	for len(tut) > 0 {
		controlips = tut
		tut = make([]string, 0)
		for _, v := range controlips {
			body, err := SendPost(v, IPADDRESS, []byte(host))
			if err != nil {
				continue //Burada eğer istek yapılan ip adresinde hata varsa bu ip adresini atlıyor.
			}
			nodes, err := DeserializeIps(body)
			if err != nil {
				return err
			}
			addedips, err := AddNodes(nodes)
			if err != nil {
				return err
			}

			if len(addedips) == 0 || len(ips.Ips) >= 1000 {
				return nil
			}
			tut = append(tut, addedips...)
		}
	}
	return nil
}

func AddNodes(nodes *Ips) ([]string, error) { //Nodeları veritabanına ekler.varsa eklemez.yeni eklenmiş ipleri döndürür.
	var addedips []string
	newips := nodes.Ips
	if len(ips.Ips) >= 1000 {
		return nil, errors.New("enough ip addresses added")
	}
	db, err := OpenDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()
	err = db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("ips"))
		var durum bool
		for _, v := range newips {
			durum = true
			for _, v2 := range ips.Ips {
				if v == v2 {
					durum = false
					break
				}
			}
			if durum {
				id, err := uuid.NewV4()
				if err != nil {
					return err
				}
				err = bucket.Put(id.Bytes(), []byte(v))
				if err != nil {
					return err
				}
				addedips = append(addedips, v)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	ips.Ips = append(ips.Ips, addedips...) //ipleri veritabanına ekledikten sonra kendi üzerinde güncelliyor.

	return addedips, nil
}

func SetNodes() error { // nodeların aktifliğini kontrol ederek aktif değilse siler.
	var deleteips []string
	for _, v := range ips.Ips {
		_, err := SendPost(v, ACTIVE, nil)
		if err != nil {
			deleteips = append(deleteips, v)
		}
	}
	err := DeleteIps(deleteips)
	return err
}

func AddIps(ipsx []string) error {
	db, err := OpenDB()
	if err != nil {
		return err
	}
	defer db.Close()
	err = db.Update(func(tx *bolt.Tx) error {
		netBucket := tx.Bucket([]byte("ips"))
		var varmi bool
		for _, i := range ipsx {
			varmi = false
			for _, j := range ips.Ips {
				if i == j {
					varmi = true
				}
			}
			if !varmi {
				id, err := uuid.NewV4()
				if err != nil {
					return err
				}
				err = netBucket.Put(id.Bytes(), []byte(i))
				if err != nil {
					return err
				}
				ips.Ips = append(ips.Ips, i) //Kendi üzerinde güncelleme yapıyor.
			}
		}
		return nil
	})
	return err
}

func DeleteIps(ipsx []string) error {
	db, err := OpenDB()
	if err != nil {
		return err
	}
	defer db.Close()
	err = db.Update(func(tx *bolt.Tx) error {
		netBucket := tx.Bucket([]byte("ips"))
		var durum bool
		ips.Ips = make([]string, 0)
		netBucket.ForEach(func(k, v []byte) error {
			durum = false
			for _, v2 := range ipsx {
				if string(v) == v2 {
					durum = true
					break
				}
			}
			if durum {
				err = netBucket.Delete(k)
				if err != nil {
					return err
				}
			} else {
				ips.Ips = append(ips.Ips, string(v)) //Kendi üzerine ekleme işlemi yapıyor.
			}
			return nil
		})
		return nil
	})
	return err
}

func SendPost(host string, id int, data []byte) ([]byte, error) {
	var postdata = &PostData{id, data}
	postbytes, err := postdata.Serialize()
	if err != nil {
		return nil, err
	}
	resp, err := http.Post("http://"+host, "application/octet-stream", bytes.NewBuffer(postbytes))
	if err != nil {
		return nil, err
	}
	return ioutil.ReadAll(resp.Body)
}

func IsValidIp(ip string) bool {
	if ip != "" {
		tut := strings.Split(ip, ":")
		if len(tut) == 2 {
			ipbody := tut[0]
			ipport := tut[1]
			_, err := strconv.Atoi(ipport)
			if err == nil {
				if ipbody == "localhost" {
					//Burası tamamdır.
					return true
				} else {
					ipnet := net.ParseIP(ipbody)
					if ipnet.To4() == nil {
						if ipnet.To16() != nil {
							//Burası tamamdır.
							return true
						}
					} else {
						//Burası tamamdır.
						return true
					}
				}
			}
		}
	}
	return false
}

func (self *PostData) Serialize() ([]byte, error) {
	var result bytes.Buffer
	encoder := gob.NewEncoder(&result)
	err := encoder.Encode(self)
	if err != nil {
		return nil, err
	}
	return result.Bytes(), nil
}

func DeserializePostData(d []byte) (*PostData, error) {
	var postdata PostData
	decoder := gob.NewDecoder(bytes.NewReader(d))
	err := decoder.Decode(&postdata)
	if err != nil {
		return nil, err
	}
	return &postdata, nil
}

func (self *Ips) Serialize() ([]byte, error) {
	var result bytes.Buffer
	encoder := gob.NewEncoder(&result)
	err := encoder.Encode(self)
	if err != nil {
		return nil, err
	}
	return result.Bytes(), nil
}

func DeserializeIps(d []byte) (*Ips, error) {
	var ipsx Ips
	decoder := gob.NewDecoder(bytes.NewReader(d))
	err := decoder.Decode(&ipsx)
	if err != nil {
		return nil, err
	}
	return &ipsx, nil
}
