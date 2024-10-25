package token

import (
	"errors"
	"time"

	"github.com/dgrijalva/jwt-go"
)

const minSecretKeyLength = 32

var (
	ErrShortSecretKey = errors.New("secret key is too short")
	ErrInvalidToken   = errors.New("invalid token")
)

type JwtMaker struct {
	secretKey string
}

func NewJwtMaker(secretKey string) (Maker, error) {
	if len(secretKey) < minSecretKeyLength {
		return nil, ErrExpiredToken
	}
	return &JwtMaker{secretKey}, nil
}

func (jwtMaker *JwtMaker) CreateToken(username string, role string, duration time.Duration) (string, *Payload, error) {
	// 创建负载
	payload, err := NewPayload(username, role, duration)
	if err != nil {
		return "", nil, err
	}

	// 创建token
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)
	token, err := jwtToken.SignedString([]byte(jwtMaker.secretKey))
	return token, payload, err
}

func (jwtMaker *JwtMaker) VerifyToken(token string) (*Payload, error) {
	// 检验token的签名算法是否正确，并获取密钥
	keyFunc := func(token *jwt.Token) (interface{}, error) {
		_, ok := token.Method.(*jwt.SigningMethodHMAC)
		if !ok {
			return nil, ErrInvalidToken
		}
		return []byte(jwtMaker.secretKey), nil
	}

	jwtToken, err := jwt.ParseWithClaims(token, &Payload{}, keyFunc)
	if err != nil {
		verr, ok := err.(*jwt.ValidationError)
		if ok && errors.Is(verr.Inner, ErrExpiredToken) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	payload, ok := jwtToken.Claims.(*Payload)
	if !ok {
		return nil, ErrInvalidToken
	}

	err = payload.Valid()
	if err != nil {
		return nil, err
	}

	return payload, nil
}
