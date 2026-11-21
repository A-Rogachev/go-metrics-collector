package agent

type Signer interface {
	SignMetric([]byte) (string, error)
}
