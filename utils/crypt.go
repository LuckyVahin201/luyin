package utils

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	memory = 64 * 1024
	time = 3
	threads = 4
	keyLen = 32
) // according to https://www.rfc-editor.org/rfc/rfc9106.html#name-recommendations
// the owasp recommendation is a bit lower but ig we'll stick with the rfc one
// it doesn't matter too much since changing these values won't affect already made passwords

var ErrInvalidHash = errors.New("given hash is invalid")

type keyParams struct {
	memory int
	time int
	threads int
	keyLen int
	version int
	salt []byte
	key []byte
}

func generateSalt() []byte {
	salt := make([]byte, 32)
	rand.Read(salt) // according to the docs rand never fails: https://pkg.go.dev/crypto/rand#Read 
	return salt
}

func HashPassword(password string) string {
	salt := generateSalt()
	key := argon2.IDKey([]byte(password), salt, time, memory, threads, keyLen)

	hash := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		memory,
		time,
		threads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	)
	return hash
}

func decodeHash(hash string) (keyParams, error) {
	params := keyParams{}
	var temp string

	_, err := fmt.Sscanf(hash, "$argon2id$v=%d$m=%d,t=%d,p=%d$%s", 
		&params.version, &params.memory, &params.time, &params.threads, &temp,
	)
	if err != nil {
		return keyParams{}, ErrInvalidHash
	}

	salt, key, found := strings.Cut(temp, "$")
	if !found {
		return keyParams{}, ErrInvalidHash
	}

	params.salt, err = base64.RawStdEncoding.Strict().DecodeString(salt)
	if err != nil {
		return keyParams{}, ErrInvalidHash
	}

	params.key, err = base64.RawStdEncoding.Strict().DecodeString(key)
	if err != nil {
		return keyParams{}, ErrInvalidHash
	}
	params.keyLen = len(params.key)

	return params, nil
}

func CompareHashes(hash string, password string) bool {
	params, err := decodeHash(hash)
	if err != nil {
		return false
	}

	fmt.Println(params)

	key := argon2.IDKey([]byte(password), params.salt, uint32(params.time), uint32(params.memory), uint8(params.threads), uint32(params.keyLen))
	if subtle.ConstantTimeCompare(params.key, key) == 1 {
		return true
	}

	return false
}