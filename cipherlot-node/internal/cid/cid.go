package cid

import (
	"encoding/base32"
	"errors"
	"strings"
)

// Build CIDv1 (base32, multibase 'b'), codec=raw (0x55), multihash sha2-256 (0x12, len 32)
func FromDigestSHA256(digest []byte) string {
	// multihash = [0x12, 32] + digest
	mh := append([]byte{0x12, 32}, digest...)
	// cid bytes = [0x01(version1), 0x55(raw)] + multihash
	cidBytes := append([]byte{0x01, 0x55}, mh...)
	enc := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(cidBytes)
	return "b" + strings.ToLower(enc)
}

// Decode CIDv1 and return the underlying sha256 digest (32 bytes).
func ToDigestSHA256(cid string) ([]byte, error) {
	if cid == "" || cid[0] != 'b' {
		return nil, errors.New("cid must start with 'b' multibase")
	}
	s := strings.ToUpper(cid[1:])
	raw, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(s)
	if err != nil {
		return nil, err
	}
	if len(raw) < 4 || raw[0] != 0x01 {
		return nil, errors.New("unsupported cid (expect v1)")
	}
	// raw[1] = codec (0x55 raw expected for blobs; we tolerate others for manifests)
	mhCode := raw[2]
	mhLen := int(raw[3])
	if mhCode != 0x12 || mhLen != 32 {
		return nil, errors.New("unsupported multihash (expect sha2-256 len 32)")
	}
	if len(raw) < 4+mhLen {
		return nil, errors.New("truncated cid multihash")
	}
	return raw[4 : 4+mhLen], nil
}
