package signing

import (
	"crypto/hmac"
	"errors"
	"hash"
)

type SignatureCodec interface {
	EncodeToString([]byte) string
	DecodeString(string) ([]byte, error)
}

type HMACSigner struct {
	key       *string
	algorithm func() hash.Hash
	codec     SignatureCodec
}

func NewHMACSigner(
	secretKey *string,
	algorithm func() hash.Hash,
	codec SignatureCodec,
) *HMACSigner {
	return &HMACSigner{
		key:       secretKey,
		algorithm: algorithm,
		codec:     codec,
	}
}

func (s *HMACSigner) Sign(data []byte) (string, error) {
	if s.key == nil {
		return "", errors.New("secret key not provided")
	}
	h := hmac.New(s.algorithm, []byte(*s.key))
	_, err := h.Write(data)
	if err != nil {
		return "", err
	}
	signature := h.Sum(nil)
	return s.codec.EncodeToString(signature), nil
}

func (s *HMACSigner) Verify(data []byte, signature string) bool {
	received, err := s.codec.DecodeString(signature)
	if err != nil {
		return false
	}
	h := hmac.New(s.algorithm, []byte(*s.key))
	h.Write(data)
	return hmac.Equal(h.Sum(nil), received)
}
