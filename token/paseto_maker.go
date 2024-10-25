package token

import (
	"fmt"
	"time"

	"github.com/aead/chacha20poly1305"
	"github.com/o1egl/paseto"
)

var (
	ErrShortSymmetricKey = fmt.Errorf("symmetric key must be %d characters", chacha20poly1305.KeySize)
)

type PasetoMaker struct {
	paseto       *paseto.V2
	symmetricKey []byte
}

func NewPasetoMaker(symmetricKey string) (Maker, error) {
	if len(symmetricKey) != chacha20poly1305.KeySize {
		return nil, ErrShortSymmetricKey
	}

	maker := &PasetoMaker{
		paseto:       paseto.NewV2(),
		symmetricKey: []byte(symmetricKey),
	}

	return maker, nil
}

func (pasetoMaker *PasetoMaker) CreateToken(username string, role string, duration time.Duration) (string, *Payload, error) {
	payload, err := NewPayload(username, role, duration)
	if err != nil {
		return "", nil, err
	}

	token, err := pasetoMaker.paseto.Encrypt(pasetoMaker.symmetricKey, payload, nil)
	return token, payload, err
}

func (pasetoMaker *PasetoMaker) VerifyToken(token string) (*Payload, error) {
	payload := &Payload{}

	err := pasetoMaker.paseto.Decrypt(token, pasetoMaker.symmetricKey, payload, nil)
	if err != nil {
		return nil, ErrInvalidToken
	}

	err = payload.Valid()
	if err != nil {
		return nil, err
	}

	return payload, nil
}
