package agent

import (
	"crypto/hmac"
	"encoding/base64"
	"errors"
	"hash"
)

type signerFunc func() hash.Hash

type MetricSigner struct {
	key       *string
	algorithm signerFunc
}

func NewMetricSigner(secretKey *string, hashFunc signerFunc) *MetricSigner {
	return &MetricSigner{
		key:       secretKey,
		algorithm: hashFunc,
	}
}

func (m *MetricSigner) SignMetric(value []byte) (string, error) {
	if m.key == nil {
		return "", errors.New("secret key not provided")
	}
	h := hmac.New(m.algorithm, []byte(*m.key))
	h.Write(value)
	dst := h.Sum(nil)

	return base64.StdEncoding.EncodeToString(dst), nil
}
