package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
	"errors"
)

// DecryptCryptoJSAES decrypts strings produced by CryptoJS.AES.encrypt(plaintext, passphrase)
// which is OpenSSL-compatible: Base64("Salted__" + 8-byte salt + AES-256-CBC(ciphertext))
func DecryptCryptoJSAES(encryptedBase64, passphrase string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(encryptedBase64)
	if err != nil {
		return "", err
	}
	if len(raw) < 16 || string(raw[:8]) != "Salted__" {
		return "", errors.New("ciphertext not in OpenSSL 'Salted__' format")
	}

	salt := raw[8:16]
	ciphertext := raw[16:]

	key, iv := evpBytesToKey([]byte(passphrase), salt, 32, 16) // AES-256 key, 16-byte IV

	if len(ciphertext)%aes.BlockSize != 0 {
		return "", errors.New("ciphertext is not a multiple of block size")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	plain := make([]byte, len(ciphertext))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(plain, ciphertext)

	// PKCS#7 unpad with strict check
	if len(plain) == 0 {
		return "", errors.New("empty plaintext")
	}
	pad := int(plain[len(plain)-1])
	if pad == 0 || pad > aes.BlockSize || pad > len(plain) {
		return "", errors.New("invalid padding")
	}
	for i := 0; i < pad; i++ {
		if plain[len(plain)-1-i] != byte(pad) {
			return "", errors.New("invalid padding")
		}
	}
	plain = plain[:len(plain)-pad]

	return string(plain), nil
}

// EVP_BytesToKey (SHA-256) for secure key derivation from passphrase
func evpBytesToKey(pass, salt []byte, keyLen, ivLen int) (key, iv []byte) {
	var d, out []byte
	for len(out) < keyLen+ivLen {
		h := sha256.New()
		h.Write(d)
		h.Write(pass)
		h.Write(salt)
		d = h.Sum(nil)
		out = append(out, d...)
	}
	return out[:keyLen], out[keyLen : keyLen+ivLen]
}
