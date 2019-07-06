package main

import (
	"net/http"
	"html/template"
	"encoding/hex"
	"crypto/x509"
	"strconv"
	"strings"
	"fmt"
)

func WebMining(address []byte) {
	mining = true
	block, err := StartMining(address)
	mining = false
	if err != nil {
		panic(err)
	}
	fmt.Println("mining işlemi başarıyla sonlandı")
	err = SendBlock(block)
	if err != nil {
		panic(err)
	}
}

func webHome(w http.ResponseWriter, r *http.Request) {
	type Data struct {
		Mining bool
	}
	var data Data

	data.Mining = mining

	html, err := template.ParseFiles("html/index.html", "html/genel/header.html", "html/genel/footer.html")
	if err != nil {
		panic(err)
	}
	html.Execute(w, data)
}

func webWallets(w http.ResponseWriter, r *http.Request) {
	type WebWallet struct {
		Id         string
		PrivateKey string
		PublicKey  string
		Address    string
		Amount     int64
	}
	type Data struct {
		Wallets []WebWallet
	}
	var data Data

	if r.Method == "POST" {
		if r.FormValue("trstart") != "" {
			trstart := strings.TrimSpace(r.FormValue("trstart"))
			address := strings.TrimSpace(r.FormValue("address"))
			amount := strings.TrimSpace(r.FormValue("amount"))
			walletid, err := hex.DecodeString(trstart)
			if err != nil {
				panic(err)
			}
			wallet, err := GetWallet(walletid)
			if err != nil {
				panic(err)
			}
			privkey, err := x509.MarshalECPrivateKey(&wallet.PrivateKey)
			if err != nil {
				panic(err)
			}
			amountint, err := strconv.ParseInt(amount, 10, 64)
			if err != nil {
				panic(err)
			}
			transaction, err := NewTransaction(privkey, Base58Decode([]byte(address)), amountint)
			if err != nil {
				panic(err)
			}
			err = StartTransaction(transaction)
			if err != nil {
				panic(err)
			}
			http.Redirect(w, r, r.URL.String(), 301)
			return
		}

		if r.FormValue("deletewallet") != "" {
			wid := r.FormValue("deletewallet")
			walletid, err := hex.DecodeString(wid)
			if err != nil {
				panic(err)
			}
			err = DeleteWallet(walletid)
			if err != nil {
				panic(err)
			}
			http.Redirect(w, r, r.URL.String(), 301)
			return
		}

		var wallet *Wallet
		if r.FormValue("newwallet") != "" {
			wallet = NewWallet()
			err := AddWallet(wallet)
			if err != nil {
				panic(err)
			}
		}
		if r.FormValue("newwallet") == "home" {
			http.Redirect(w, r, "/wallets?id="+hex.EncodeToString(wallet.Id), 301)
			return
		}
		http.Redirect(w, r, r.URL.String(), 301)
		return
	}

	linkid := r.URL.Query().Get("id")
	if linkid != "" { //Cüzdan araması varsa çalıştırılıyor.
		walletid, err := hex.DecodeString(linkid)
		if err == nil {
			wallet, err := GetWallet(walletid)
			if err != nil {
				panic(err)
			}
			if wallet != nil {
				var webwallet WebWallet
				webwallet.Id = hex.EncodeToString(wallet.Id)
				privKey, err := x509.MarshalECPrivateKey(&wallet.PrivateKey)
				if err != nil {
					panic(err)
				}
				webwallet.PrivateKey = string(Base58Encode(privKey))
				webwallet.PublicKey = string(Base58Encode(wallet.PublicKey))
				webwallet.Address = string(Base58Encode(wallet.GetAddress()))
				webwallet.Amount, err = blockchain.GetAddressAmount(wallet.GetAddress())
				if err != nil {
					panic(err)
				}
				data.Wallets = append(data.Wallets, webwallet)
			}
		}
	} else { //Burada tüm cüzdanlar getiriliyor.
		wallets, err := GetWallets()
		if err != nil {
			panic(err)
		}

		for _, v := range wallets { //Cüzdan bilgilerinin ayarlarndığı yer.
			var webwallet WebWallet
			webwallet.Id = hex.EncodeToString(v.Id)
			privKey, err := x509.MarshalECPrivateKey(&v.PrivateKey)
			if err != nil {
				panic(err)
			}

			webwallet.PrivateKey = string(Base58Encode(privKey))
			webwallet.PublicKey = string(Base58Encode(v.PublicKey))
			webwallet.Address = string(Base58Encode(v.GetAddress()))
			webwallet.Amount, err = blockchain.GetAddressAmount(v.GetAddress())
			if err != nil {
				panic(err)
			}
			data.Wallets = append(data.Wallets, webwallet)
		}
	}

	html := template.New("wallets.html")
	html = html.Funcs(template.FuncMap{
		"kalan": func(sayi int, i int) int {
			return sayi % i
		},
		"wlen": func(webwallets []WebWallet) int {
			return len(webwallets)
		},
		"add": func(sayi int) int {
			return sayi + 1
		},
		"sub": func(sayi int) int {
			return sayi - 1
		},
	})
	html.ParseFiles("html/wallets.html", "html/genel/header.html", "html/genel/footer.html")

	html.Execute(w, data)
}

func webBlockchain(w http.ResponseWriter, r *http.Request) {
	type WebBlock struct {
		Timestamp     int64
		Transactions  int
		PrevBlockHash string
		Hash          string
		Nonce         int64
		Height        int64
		Address       string
		Prize         int64
	}
	type Data struct {
		Blocks    []WebBlock
		BlocksLen int
	}
	var data Data

	for _, v := range blockchain.Blocks {
		var webblock WebBlock
		webblock.Timestamp = v.Timestamp
		webblock.Transactions = len(v.Transactions)
		if v.PrevBlockHash != nil {
			webblock.PrevBlockHash = string(Base58Encode(v.PrevBlockHash))
		}
		webblock.Hash = string(Base58Encode(v.Hash))
		webblock.Nonce = v.Nonce
		webblock.Height = v.Height
		webblock.Address = string(Base58Encode(v.Address))
		webblock.Prize = v.Prize
		data.Blocks = append(data.Blocks, webblock)
	}

	data.BlocksLen = len(data.Blocks)

	html := template.New("blockchain.html").Funcs(template.FuncMap{
		"trlen": func(tr []*Transaction) int {
			return len(tr)
		},
	})

	html, err := html.ParseFiles("html/blockchain.html", "html/genel/header.html", "html/genel/footer.html")
	if err != nil {
		panic(err)
	}
	html.Execute(w, data)
}

func webBlockTransactions(w http.ResponseWriter, r *http.Request) {
	type WebBlockTransaction struct {
		Id      string
		Sign    string
		PubKey  string
		Address string
		Amount  int64
	}
	type Data struct {
		BlockTransactions []WebBlockTransaction
		BTLen             int
		Height            int64
	}
	var data Data
	var err error
	height := r.URL.Query().Get("height")
	if height == "" {
		return
	}

	data.Height, err = strconv.ParseInt(height, 10, 64)
	if err != nil {
		panic(err)
	}

	block := blockchain.Blocks[data.Height-1]
	if block == nil {
		return
	}

	transactions := block.Transactions

	for _, v := range transactions {
		var webbt WebBlockTransaction
		webbt.Id = hex.EncodeToString(v.ID)
		webbt.Sign = string(Base58Encode(v.Sign))
		webbt.PubKey = string(Base58Encode(v.PubKey))
		webbt.Address = string(Base58Encode(v.Address))
		webbt.Amount = v.Amount
		data.BlockTransactions = append(data.BlockTransactions, webbt)
	}

	data.BTLen = len(data.BlockTransactions)

	html, err := template.ParseFiles("html/blocktransactions.html", "html/genel/header.html", "html/genel/footer.html")
	if err != nil {
		panic(err)
	}
	html.Execute(w, data)
}

func webIps(w http.ResponseWriter, r *http.Request) {
	type Data struct {
		Ips    []string
		IpsLen int
	}
	var data Data
	var err error

	data.Ips = ips.Ips
	data.IpsLen = len(data.Ips)

	html, err := template.ParseFiles("html/ips.html", "html/genel/header.html", "html/genel/footer.html")
	if err != nil {
		panic(err)
	}
	html.Execute(w, data)
}

func webUtxo(w http.ResponseWriter, r *http.Request) {
	type WebUtxo struct {
		Id      string
		Sign    string
		PubKey  string
		Address string
		Amount  int64
	}
	type Data struct {
		Utxo    []WebUtxo
		UtxoLen int
	}
	var data Data
	var err error

	if r.Method == "POST" {
		if r.FormValue("newtransaction") != "" {
			privkey := r.FormValue("privkey")
			address := r.FormValue("address")
			amount := r.FormValue("amount")
			amountint, err := strconv.ParseInt(amount, 10, 64)
			if err != nil {
				panic(err)
			}

			transaction, err := NewTransaction(Base58Decode([]byte(strings.TrimSpace(privkey))), Base58Decode([]byte(strings.TrimSpace(address))),
				amountint)
			if err != nil {
				panic(err)
			}
			err = StartTransaction(transaction)
			if err != nil {
				panic(err)
			}
		}

		http.Redirect(w, r, "/utxo", 301)
		return
	}

	for _, v := range utxo {
		var webutxo WebUtxo
		webutxo.Id = hex.EncodeToString(v.ID)
		webutxo.Sign = string(Base58Encode(v.Sign))
		webutxo.PubKey = string(Base58Encode(v.PubKey))
		webutxo.Address = string(Base58Encode(v.Address))
		webutxo.Amount = v.Amount
		data.Utxo = append(data.Utxo, webutxo)
	}

	data.UtxoLen = len(data.Utxo)

	html, err := template.ParseFiles("html/utxo.html", "html/genel/header.html", "html/genel/footer.html")
	if err != nil {
		panic(err)
	}
	html.Execute(w, data)
}

func webSettings(w http.ResponseWriter, r *http.Request) {
	host = r.Host
	if r.Method == "POST" {
		if r.FormValue("addip") != "" {
			ip := r.FormValue("ip")
			if IsValidIp(ip) {
				err := AddIps([]string{ip})
				if err != nil {
					panic(err)
				}
				http.Redirect(w, r, "/ips", 301)
				return
			}
		} else if r.FormValue("updatebs") != "" {
			err := blockchain.UpdateBlockchain()
			if err != nil {
				panic(err)
			}
			fmt.Println("blockchain güncellendi")
			err = UpdateSettings()
			if err != nil {
				panic(err)
			}
			fmt.Println("ayarlar güncellendi")
			http.Redirect(w, r, "/blockchain", 301)
			return
		} else if r.FormValue("findnodes") != "" {
			err := FindNodes()
			if err != nil {
				panic(err)
			}
			http.Redirect(w, r, "/ips", 301)
			return
		} else if r.FormValue("setnodes") != "" {
			err := SetNodes()
			if err != nil {
				panic(err)
			}
			http.Redirect(w, r, "/ips", 301)
			return
		}
		http.Redirect(w, r, r.URL.String(), 301)
		return
	}

	html, err := template.ParseFiles("html/settings.html", "html/genel/header.html", "html/genel/footer.html")
	if err != nil {
		panic(err)
	}
	html.Execute(w, settings)
}

func webMining(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		if r.FormValue("mining") != "" {
			fmt.Println("post")
			address := strings.TrimSpace(r.FormValue("address"))
			if address != "" {
				fmt.Println("mining kontrol")
				if !mining { //Mining işlemi yapılmıyorsa yapacak.
					fmt.Println("mining yok")
					if ValidateAddress(string(Base58Decode([]byte(address)))) {
						fmt.Println("mining başladı")
						go WebMining(Base58Decode([]byte(address))) // Mining işlemini asenkron olarak başlatır.
					} else {
						fmt.Println("address doğrulanmadı")
					}
				} else {
					fmt.Println("mining zaten yapılıyor")
				}
			} else {
				fmt.Println("address başarısız")
			}
		}
	}

	http.Redirect(w, r, "/", 301)
	return
}
