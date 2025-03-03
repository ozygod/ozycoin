package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/gob"
	"fmt"
	"golang.org/x/crypto/ripemd160"
	"log"
	"math/big"
	"os"
)

const addressChecksumLen = 4
const pkhVersion = byte(0x00)
const walletFile = "wallet_%s.dat"

type Wallet struct {
	PrivateKey ecdsa.PrivateKey
	PublicKey  []byte
}

type Wallets struct {
	Wallets map[string]*Wallet
}

func init() {
	gob.Register(elliptic.P256())
	gob.Register(elliptic.P384())
	gob.Register(elliptic.P521()) // 如果您可能使用其他曲线，也需要注册
}

func NewWallet() *Wallet {
	private, public := newKeyPair()
	wallet := Wallet{private, public}
	return &wallet
}

func newKeyPair() (ecdsa.PrivateKey, []byte) {
	curve := elliptic.P256()
	private, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		log.Panic(err)
	}
	public := append(private.PublicKey.X.Bytes(), private.PublicKey.Y.Bytes()...)
	return *private, public
}

func (w Wallet) GetAddress() []byte {
	pubKeyHash := HashPubKey(w.PublicKey)

	versionedPayload := append([]byte{pkhVersion}, pubKeyHash...)
	checksum := checkSum(versionedPayload)
	fullPayload := append(versionedPayload, checksum...)
	address := Base58Encode(fullPayload)

	return address
}

func ValidateAddress(address string) bool {
	pubKeyHash := Base58Decode([]byte(address))
	actualChecksum := pubKeyHash[len(pubKeyHash)-addressChecksumLen:]
	version := pubKeyHash[0]
	pubKeyHash = pubKeyHash[len([]byte{version}) : len(pubKeyHash)-addressChecksumLen]
	targetChecksum := checkSum(append([]byte{version}, pubKeyHash...))

	return bytes.Compare(actualChecksum, targetChecksum) == 0
}

func HashPubKey(pubKey []byte) []byte {
	pubKeyHash := sha256.Sum256(pubKey)
	RIPEMD160Hasher := ripemd160.New()
	_, err := RIPEMD160Hasher.Write(pubKeyHash[:])
	if err != nil {
		log.Panic(err)
	}
	pubKeyRIPEMD160 := RIPEMD160Hasher.Sum(nil)
	return pubKeyRIPEMD160
}

func checkSum(payload []byte) []byte {
	firstHash := sha256.Sum256(payload)
	secondHash := sha256.Sum256(firstHash[:])

	return secondHash[:addressChecksumLen]
}

func GetPublicKeyHash(address string) []byte {
	pubKeyHash := Base58Decode([]byte(address))
	pubKeyHash = pubKeyHash[len([]byte{pkhVersion}) : len(pubKeyHash)-addressChecksumLen]
	return pubKeyHash
}

func NewWallets(nodeId string) (*Wallets, error) {
	wallets := Wallets{}
	wallets.Wallets = make(map[string]*Wallet)

	err := wallets.LoadFromFile(nodeId)
	return &wallets, err
}

func (ws *Wallets) GetAddresses() []string {
	var addresses []string
	for _, wallet := range ws.Wallets {
		addresses = append(addresses, string(wallet.GetAddress()))
	}
	return addresses
}

func (ws Wallets) GetWallet(address string) Wallet {
	return *ws.Wallets[address]
}

func (ws *Wallets) LoadFromFile(nodeId string) error {
	path := fmt.Sprintf(walletFile, nodeId)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return err
	}

	fileContent, err := os.ReadFile(path)
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

func (ws Wallets) SaveToFile(nodeId string) error {
	path := fmt.Sprintf(walletFile, nodeId)
	var content bytes.Buffer

	gob.Register(elliptic.P256())

	encoder := gob.NewEncoder(&content)
	err := encoder.Encode(ws)
	if err != nil {
		log.Panic(err)
	}

	err = os.WriteFile(path, content.Bytes(), 0644)
	if err != nil {
		log.Panic(err)
	}
	return err
}

func (ws *Wallets) CreateWallet() string {
	wallet := NewWallet()
	address := fmt.Sprintf("%s", wallet.GetAddress())
	ws.Wallets[address] = wallet
	return address
}

// GobEncode 自定义 Wallet 类型的 Gob 序列化方法
func (w Wallet) GobEncode() ([]byte, error) {
	curveName := ""
	switch w.PrivateKey.Curve {
	case elliptic.P256():
		curveName = "P256"
	case elliptic.P384():
		curveName = "P384"
	case elliptic.P521():
		curveName = "P521"
	default:
		return nil, fmt.Errorf("unsupported curve type for gob encoding")
	}

	buf := new(bytes.Buffer)
	encoder := gob.NewEncoder(buf)

	err := encoder.Encode(curveName) // 序列化曲线名称
	if err != nil {
		return nil, err
	}
	err = encoder.Encode(w.PrivateKey.D) // 序列化私钥 D
	if err != nil {
		return nil, err
	}
	err = encoder.Encode(w.PrivateKey.PublicKey.X) // 序列化公钥 X
	if err != nil {
		return nil, err
	}
	err = encoder.Encode(w.PrivateKey.PublicKey.Y) // 序列化公钥 Y
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// GobDecode 自定义 Wallet 类型的 Gob 反序列化方法
func (w *Wallet) GobDecode(data []byte) error {
	buf := bytes.NewBuffer(data)
	decoder := gob.NewDecoder(buf)

	var curveName string
	err := decoder.Decode(&curveName) // 反序列化曲线名称
	if err != nil {
		return err
	}

	var d *big.Int
	err = decoder.Decode(&d) // 反序列化私钥 D
	if err != nil {
		return err
	}

	var x *big.Int
	err = decoder.Decode(&x) // 反序列化公钥 X
	if err != nil {
		return err
	}

	var y *big.Int
	err = decoder.Decode(&y) // 反序列化公钥 Y
	if err != nil {
		return err
	}

	var curve elliptic.Curve
	switch curveName {
	case "P256":
		curve = elliptic.P256()
	case "P384":
		curve = elliptic.P384()
	case "P521":
		curve = elliptic.P521()
	default:
		return fmt.Errorf("unsupported curve name: %s", curveName)
	}

	w.PrivateKey = ecdsa.PrivateKey{
		D: d,
		PublicKey: ecdsa.PublicKey{
			Curve: curve,
			X:     x,
			Y:     y,
		},
	}
	w.PublicKey = append(w.PrivateKey.PublicKey.X.Bytes(), w.PrivateKey.PublicKey.Y.Bytes()...)

	return nil
}
