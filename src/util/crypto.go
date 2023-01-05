package util

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"hash"
	"math/big"
	"strings"

	"golang.org/x/text/encoding/unicode"
)

//
const DATA_BLOCK_SIZE = 4096

// AES-128
const AES_KEY_128 = "0123456789ABCDEF"

// AES-256
const AES_KEY_256 = "0123456789abcdef0123456789ABCDEF"

// AES-128-IV
const AES_IV_128 = "0123456789012345"

// AES-256-IV
const AES_IV_256 = "01234567890123456789012345678901"

//
func MD5(text string) string {
	hash := md5.New()
	_, err := hash.Write([]byte(text))
	if err != nil {
		return ""
	}
	return strings.ToUpper(hex.EncodeToString(hash.Sum(nil)))
}

func SHA1(text string) string {
	hash := sha1.New()
	_, err := hash.Write([]byte(text))
	if err != nil {
		return ""
	}
	return strings.ToUpper(hex.EncodeToString(hash.Sum(nil)))
}

func SHA256(text string) string {
	hash := sha256.New()
	_, err := hash.Write([]byte(text))
	if err != nil {
		return ""
	}
	return strings.ToUpper(hex.EncodeToString(hash.Sum(nil)))
}

func HashMD5Init() hash.Hash {
	hash := md5.New()
	return hash
}

func HashSHA256Init() hash.Hash {
	hash := sha256.New()
	return hash
}

func HashData(hh hash.Hash, data []byte) int {
	llen := len(data)
	tlen := 0
	lblock := DATA_BLOCK_SIZE
	for tlen < llen {
		if llen-tlen >= DATA_BLOCK_SIZE {
			lblock = DATA_BLOCK_SIZE
		} else {
			lblock = llen - tlen
		}

		var temp = data[tlen : tlen+lblock]
		l, err := hh.Write(temp)
		if err != nil {
			l = 0
			tlen = -1
			break
		}

		tlen = tlen + l
	}
	return tlen
}

func HashMD5Final(hh hash.Hash) []byte {
	return hh.Sum(nil)
}

func HashSHA256Final(hh hash.Hash) []byte {
	return hh.Sum(nil)
}

//
func ECCCurve256() elliptic.Curve {
	return elliptic.P256()
}

// ECC generate key
func ECCGenkey() (*ecdsa.PrivateKey, *ecdsa.PublicKey, error) {
	private_key, err := ecdsa.GenerateKey(ECCCurve256(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	return private_key, &private_key.PublicKey, nil
}

func ECCkeyBitsSize(key *ecdsa.PublicKey) int {
	return key.Curve.Params().BitSize
}

func ECCKeyByteLen(key *ecdsa.PublicKey) int {
	return (key.Curve.Params().BitSize + 7) >> 3
}

func ECCX509PrivateKeyEncoding(key *ecdsa.PrivateKey) string {
	buffer, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return ""
	}
	return strings.ToUpper(hex.EncodeToString(buffer))
}

func ECCX509PrivateKeyDecoding(text string) *ecdsa.PrivateKey {
	buffer, err := hex.DecodeString(text)
	if err != nil {
		return nil
	}
	key, err := x509.ParseECPrivateKey(buffer)
	if err != nil {
		return nil
	}
	return key
}

func ECCPrivateKeyEncodingX(key *ecdsa.PrivateKey) string {
	buffer := key.D.Bytes()
	return strings.ToUpper(hex.EncodeToString(buffer))
}

func ECCPrivateKeyDecodingX(text string) *ecdsa.PrivateKey {
	buffer, err := hex.DecodeString(text)
	if err != nil {
		return nil
	}
	length := len(buffer)

	var key ecdsa.PrivateKey
	key.Curve = ECCCurve256()
	key.PublicKey.Curve = ECCCurve256()

	key.D = big.NewInt(int64(length))
	key.D.SetBytes(buffer)
	if key.D.Sign() <= 0 {
		return nil
	}

	x, y := key.Curve.ScalarBaseMult(key.D.Bytes())
	if !key.Curve.IsOnCurve(x, y) {
		return nil
	}

	key.PublicKey.X = x
	key.PublicKey.Y = y
	return &key
}

func ECCX509PrivateKeyEncodingP8(key *ecdsa.PrivateKey) string {
	buffer, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return ""
	}
	return strings.ToUpper(hex.EncodeToString(buffer))
}

func ECCX509PrivateKeyDecodingP8(text string) *ecdsa.PrivateKey {
	buffer, err := hex.DecodeString(text)
	if err != nil {
		return nil
	}
	key, err := x509.ParsePKCS8PrivateKey(buffer)
	if err != nil {
		return nil
	}
	return key.(*ecdsa.PrivateKey)
}

func ECCPublicKeyData(key *ecdsa.PublicKey) []byte {
	buffer := elliptic.Marshal(key.Curve, key.X, key.Y)
	//buffer, err := x509.MarshalPKIXPublicKey(key)
	//if err != nil {
	//	return ""
	//}
	return buffer
}

func ECCPublicKeyParseData(data []byte) *ecdsa.PublicKey {
	if data == nil {
		return nil
	}

	x, y := elliptic.Unmarshal(ECCCurve256(), data)
	if x == nil || y == nil {
		return nil
	}

	return &ecdsa.PublicKey{
		Curve: ECCCurve256(),
		X:     x,
		Y:     y,
	}
}

func ECCPublicKeyEncoding(key *ecdsa.PublicKey) string {
	buffer := ECCPublicKeyData(key)
	return strings.ToUpper(hex.EncodeToString(buffer))
}

func ECCPublicKeyDecoding(text string) *ecdsa.PublicKey {
	buffer, err := hex.DecodeString(text)
	if err != nil {
		return nil
	}
	return ECCPublicKeyParseData(buffer)
}

func ECCGenSharedKey(prikeyA *ecdsa.PrivateKey, pubkeyB *ecdsa.PublicKey) []byte {
	x, _ := prikeyA.Curve.ScalarMult(pubkeyB.X, pubkeyB.Y, prikeyA.D.Bytes())
	hash := sha256.Sum256(x.Bytes())
	return hash[:]
}

func ECCGenSharedKeyEncoding(prikeyA *ecdsa.PrivateKey, pubkeyB *ecdsa.PublicKey) string {
	buffer := ECCGenSharedKey(prikeyA, pubkeyB)
	return strings.ToUpper(hex.EncodeToString(buffer))
}

func ECCSign(hash []byte, key *ecdsa.PrivateKey) ([]byte, error) {
	r, s, err := ecdsa.Sign(rand.Reader, key, hash)
	if err != nil {
		return nil, err
	}

	rx := r.Bytes()
	sx := s.Bytes()
	if len(rx) > 0xFF || len(sx) > 0xFF {
		return nil, errors.New("ECC sign length error")
	}

	var data bytes.Buffer
	data.WriteByte('M')
	data.WriteByte('S')
	data.WriteByte(uint8(len(rx)))
	data.Write(rx)
	data.WriteByte(uint8(len(rx)))
	data.Write(sx)
	return data.Bytes(), nil
}

func ECCVerify(hash []byte, sign []byte, key *ecdsa.PublicKey) (int, error) {
	var data *bytes.Buffer = bytes.NewBuffer(sign)
	FA, _ := data.ReadByte()
	FB, _ := data.ReadByte()
	if FA != 'M' || FB != 'S' {
		return -2, errors.New("ECC sign data format error")
	}
	rl, _ := data.ReadByte()
	rx := make([]byte, rl)
	data.Read(rx)
	sl, _ := data.ReadByte()
	sx := make([]byte, sl)
	data.Read(sx)
	r := big.NewInt(int64(rl))
	s := big.NewInt(int64(sl))
	r.SetBytes(rx)
	s.SetBytes(sx)
	result := ecdsa.Verify(key, hash, r, s)
	if !result {
		return -1, errors.New("ECC verify failed")
	}
	return 0, nil
}

func ECCSignEncoding(hash []byte, key *ecdsa.PrivateKey) (string, error) {
	sign, err := ECCSign(hash, key)
	if err != nil {
		return "", err
	}
	return strings.ToUpper(hex.EncodeToString(sign)), nil
}

func ECCSignData(data []byte, key *ecdsa.PrivateKey) ([]byte, error) {
	hash := HashSHA256Init()
	HashData(hash, data)
	digest := HashSHA256Final(hash)

	return ECCSign(digest, key)
}

func ECCSignDataEncoding(data []byte, key *ecdsa.PrivateKey) (string, error) {
	sign, err := ECCSignData(data, key)
	if err != nil {
		return "", err
	}
	return strings.ToUpper(hex.EncodeToString(sign)), nil
}

// 0: ok
// -1: failed
// -2: error
func ECCVerifyData(data []byte, sign []byte, key *ecdsa.PublicKey) (int, error) {
	hash := HashSHA256Init()
	HashData(hash, data)
	digest := HashSHA256Final(hash)

	return ECCVerify(digest, sign, key)
}

func ECCVerifyDataDecoding(data []byte, sign string, key *ecdsa.PublicKey) (int, error) {
	sign_data, err := hex.DecodeString(sign)
	if err != nil {
		return -2, err
	}
	return ECCVerifyData(data, sign_data, key)
}

func ECCEncrypt(pubkey *ecdsa.PublicKey) {

}

func ECCDecrypt(prikey *ecdsa.PrivateKey) {
}

//
func AESPKCS7Padding(cipher []byte, block int) []byte {
	padding := block - len(cipher)%block
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(cipher, padtext...)
}

func AESPKCS7Unpadding(data []byte) []byte {
	length := len(data)
	unpadding := int(data[length-1])
	return data[:(length - unpadding)]
}

func AESKey(key []byte) ([]byte, []byte) {
	if len(key) <= 16 {
		key_128 := append(key, []byte(AES_KEY_128)...)[0:16]
		return key_128, []byte(AES_IV_128)
	}
	key_256 := append(key, []byte(AES_KEY_256)...)[0:32]
	return key_256, []byte(AES_IV_256)
}

//AES加密,CBC
func AESEncrypt(data, key []byte) ([]byte, error) {
	key, iv := AESKey(key)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	block_size := block.BlockSize()
	data = AESPKCS7Padding(data, block_size)
	block_mode := cipher.NewCBCEncrypter(block, iv[:block_size])
	buffer := make([]byte, len(data))
	block_mode.CryptBlocks(buffer, data)
	return buffer, nil
}

//AES解密,CBC
func AESDecrypt(buffer, key []byte) ([]byte, error) {
	key, iv := AESKey(key)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	block_size := block.BlockSize()
	block_mode := cipher.NewCBCDecrypter(block, iv[:block_size])
	data := make([]byte, len(buffer))
	block_mode.CryptBlocks(data, buffer)
	data = AESPKCS7Unpadding(data)
	return data, nil
}

func AESEncryptString(data []byte, key []byte) (string, error) {
	buffer, err := AESEncrypt(data, key)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(buffer), nil
}

func AESDecryptString(text string, key []byte) ([]byte, error) {
	buffer, err := base64.StdEncoding.DecodeString(text)
	if err != nil {
		return nil, err
	}

	data, err := AESDecrypt(buffer, key)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func AESEncryptStringUTF8(text string, key []byte) (string, error) {
	encoder := unicode.UTF8.NewEncoder()
	data, err := encoder.Bytes([]byte(text))
	if err != nil {
		return "", err
	}
	return AESEncryptString(data, key)
}

func AESDecryptStringUTF8(text string, key []byte) (string, error) {

	data, err := AESDecryptString(text, key)
	if err != nil {
		return "", err
	}

	decoder := unicode.UTF8.NewDecoder()
	data, err = decoder.Bytes(data)
	if err != nil {
		return "", err
	}

	return string(data), nil
}
