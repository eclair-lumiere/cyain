package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	
	"golang.org/x/crypto/ripemd160"
	
	"cyain/utils"
)

const (
	version            = byte(0x01)
	addressChecksumLen = 4
	wallet_file        = "wallet.db"
)

type Wallet struct {
	PrivateKey ecdsa.PrivateKey
	PublicKey  []byte
}

type Wallets struct {
	Wallets map[string]*Wallet
}

func NewWallet() *Wallet {
	private, public := newKeyPair()
	wallet := Wallet{
		private,
		public,
	}
	
	return &wallet
}

func newKeyPair() (ecdsa.PrivateKey, []byte) {
	curve := elliptic.P256()
	private, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		log.Panic(err)
	}
	pubKey := append(private.PublicKey.X.Bytes(), private.PublicKey.Y.Bytes()...)
	
	return *private, pubKey
}

func (w Wallet) GetAddress() []byte {
	pubKeyHash := HashPubKey(w.PublicKey)
	
	versionedPayload := append([]byte{version}, pubKeyHash...)
	checksum := checksum(versionedPayload)
	
	fullPayload := append(versionedPayload, checksum...)
	address := utils.Base58Encode(fullPayload)
	
	return address
}

func HashPubKey(pubKey []byte) []byte {
	publicSHA256 := sha256.Sum256(pubKey)
	
	RIPEMD160Hasher := ripemd160.New()
	_, err := RIPEMD160Hasher.Write(publicSHA256[:])
	if err != nil {
		log.Panic(err)
	}
	publicRIPEMD160 := RIPEMD160Hasher.Sum(nil)
	
	return publicRIPEMD160
}

func checksum(payload []byte) []byte {
	firstSHA := sha256.Sum256(payload)
	secondSHA := sha256.Sum256(firstSHA[:])
	
	return secondSHA[:addressChecksumLen]
}

func NewWallets(nodeid string) (*Wallets, error) {
	wallets := Wallets{}
	wallets.Wallets = make(map[string]*Wallet)
	
	err := wallets.LoadFromFile(nodeid)
	
	return &wallets, err
}

func (ws *Wallets) CreateWallet() string {
	wallet := NewWallet()
	address := fmt.Sprintf("%s", wallet.GetAddress())
	
	ws.Wallets[address] = wallet
	
	return address
}

func (ws *Wallets) GetAddresses() []string {
	var addresses []string
	
	for address := range ws.Wallets {
		addresses = append(addresses, address)
	}
	
	return addresses
}

func (ws Wallets) GetWallet(address string) Wallet {
	return *ws.Wallets[address]
}

func (ws *Wallets) LoadFromFile(nodeid string) error {
	wallet_file := fmt.Sprintf(wallet_file, nodeid)
	if _, err := os.Stat(wallet_file); os.IsNotExist(err) {
		return err
	}
	
	fileContent, err := ioutil.ReadFile(wallet_file)
	if err != nil {
		log.Panic(err)
	}
	
	var wallets Wallets
	gob.Register(elliptic.P256())
	decoder := gob.NewDecoder(bytes.NewReader(fileContent))
	err = decoder.Decode(&wallets)
	if err != nil {
		log.Panic(err)
	}
	
	ws.Wallets = wallets.Wallets
	
	return nil
}

func (ws Wallets) SaveToFile() {
	var content bytes.Buffer
	
	gob.Register(elliptic.P256())
	
	encoder := gob.NewEncoder(&content)
	err := encoder.Encode(ws)
	if err != nil {
		log.Panic(err)
	}
	
	err = ioutil.WriteFile(wallet_file, content.Bytes(), 0644)
	if err != nil {
		log.Panic(err)
	}
}

func ValidateAddress(address string) bool {
	pubKeyHash := utils.Base58Decode([]byte(address))
	actualChecksum := pubKeyHash[len(pubKeyHash)-addressChecksumLen:]
	version := pubKeyHash[0]
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-addressChecksumLen]
	targetChecksum := checksum(append([]byte{version}, pubKeyHash...))
	
	return bytes.Compare(actualChecksum, targetChecksum) == 0
}
