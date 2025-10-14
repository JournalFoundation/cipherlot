package cid

import (
	"crypto/sha256"
	"encoding/base32"
	"errors"
	"strings"
)

func FromBytesSHA256(b []byte) string {
	sum := sha256.Sum256(b)
	mh := append([]byte{0x12, 32}, sum[:]...)
	cidBytes := append([]byte{0x01, 0x55}, mh...)
	enc := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(cidBytes)
	return "b" + strings.ToLower(enc)
}

func ToDigestSHA256(c string) ([]byte, error) {
	if c == "" || c[0] != 'b' {
		return nil, errors.New("cid must start with 'b'")
	}
	s := strings.ToUpper(c[1:])
	raw, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(s)
	if err != nil {
		return nil, err
	}
	if len(raw) < 4 || raw[0] != 0x01 || raw[2] != 0x12 || raw[3] != 32 || len(raw) < 36 {
		return nil, errors.New("unsupported cid")
	}
	return raw[4:36], nil
}
