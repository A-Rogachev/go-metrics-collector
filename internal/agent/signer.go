package agent

type Signer interface {
	Sign([]byte) (string, error)
}
