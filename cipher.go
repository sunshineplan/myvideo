package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"hash"
	"io"
)

type encrypted struct {
	Key, CT, IV, S string
}

func (e encrypted) decode() (ct, iv, s []byte, err error) {
	ct, err = base64.StdEncoding.DecodeString(e.CT)
	if err != nil {
		return
	}
	iv, err = hex.DecodeString(e.IV)
	if err != nil {
		return
	}
	s, err = hex.DecodeString(e.S)
	return
}

func (e encrypted) decrypt() (string, error) {
	ct, iv, s, err := e.decode()
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(kdf([]byte(e.Key), s, 1, 32, md5.New))
	if err != nil {
		return "", err
	}

	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(ct, ct)

	for i := range ct {
		if ct[i] == '\a' ||
			ct[i] == '\b' ||
			ct[i] == '\f' ||
			ct[i] == '\v' ||
			ct[i] == '\x01' ||
			ct[i] == '\x02' ||
			ct[i] == '\x03' ||
			ct[i] == '\x04' ||
			ct[i] == '\x05' ||
			ct[i] == '\x06' ||
			ct[i] == '\x0e' ||
			ct[i] == '\x0f' ||
			ct[i] == '\x10' {
			ct[i] = byte(' ')
		}
	}

	var pt string
	if err := json.Unmarshal(ct, &pt); err != nil {
		return "", err
	}

	return pt, nil
}

func kdf(password, salt []byte, iterations, keysize int, hash func() hash.Hash) []byte {
	var derivedKey, block []byte
	hasher := hash()
	for len(derivedKey) < keysize {
		if len(block) != 0 {
			io.Copy(hasher, bytes.NewBuffer(block))
		}
		io.Copy(hasher, bytes.NewBuffer(password))
		io.Copy(hasher, bytes.NewBuffer(salt))
		block = hasher.Sum(nil)
		hasher.Reset()

		for i := 1; i < iterations; i++ {
			io.Copy(hasher, bytes.NewBuffer(block))
			block = hasher.Sum(nil)
			hasher.Reset()
		}

		derivedKey = append(derivedKey, block...)
	}

	return derivedKey[0:keysize]
}
