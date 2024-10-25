package token

import "time"

type Maker interface {
	CraeteToken(username string, role string, duration time.Duration) (string, *Payload, error)

	VerifyToken(token string) (*Payload, error)
}
