package secretsmanager

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

// encrypt returns value as base64url(IV || AES-CFB ciphertext).
//
// CFB is deprecated, and unauthenticated: a value decrypted with the wrong
// key comes back as rubbish rather than as an error. It stays because it's
// the mode the stored files were written with, and reading them matters
// more than the mode does.
func encrypt(key []byte, value string) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	// The IV goes in front of the ciphertext. It needs to be unique, not
	// secret, which is why re-encrypting one value gives a different result
	// every time.
	encrypted := make([]byte, aes.BlockSize+len(value))
	iv := encrypted[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}

	cipher.NewCFBEncrypter(block, iv).XORKeyStream(encrypted[aes.BlockSize:], []byte(value))

	return base64.URLEncoding.EncodeToString(encrypted), nil
}

// decrypt reverses encrypt.
func decrypt(key []byte, value string) (string, error) {
	encrypted, err := base64.URLEncoding.DecodeString(value)
	if err != nil {
		return "", err
	}
	if len(encrypted) < aes.BlockSize {
		return "", fmt.Errorf("value is %d bytes, too short to hold an IV", len(encrypted))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	iv, ciphertext := encrypted[:aes.BlockSize], encrypted[aes.BlockSize:]
	decrypted := make([]byte, len(ciphertext))
	cipher.NewCFBDecrypter(block, iv).XORKeyStream(decrypted, ciphertext)

	return string(decrypted), nil
}
