package desync

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/pkg/errors"
)

func CreateHash(passphrase string) []byte {
	if passphrase == "" {
		return nil
	}
	hasher := md5.New()
	hasher.Write([]byte(passphrase))
	return []byte(hex.EncodeToString(hasher.Sum(nil)))
}

func Encrypt(toEncrypt, key []byte) ([]byte, error) {
	c, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(c)
	if err != nil {
		fmt.Println(err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, toEncrypt, nil), nil
}

func Decrypt(toDecrypt, key []byte) ([]byte, error) {
	c, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if len(toDecrypt) < nonceSize {
		return nil, errors.New("Decrypt invalid size")
	}
	nonce, toDecrypt := toDecrypt[:nonceSize], toDecrypt[nonceSize:]
	decrypted, err := gcm.Open(nil, nonce, toDecrypt, nil)
	if err != nil {
		return nil, err
	}
	return decrypted, nil
}
