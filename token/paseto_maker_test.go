package token

import (
	"simplebank/util"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPasetoMaker(t *testing.T) {
	symmetricKey := util.RandomString(32)
	maker, err := NewPasetoMaker(symmetricKey)
	require.NoError(t, err)
	require.NotEmpty(t, maker)

	username := util.RandomOwner()
	role := DepositorRole
	duration := time.Minute

	issuedAt := time.Now()
	expriedAt := issuedAt.Add(duration)

	token, payload, err := maker.CreateToken(username, role, duration)
	require.NoError(t, err)
	require.NotEmpty(t, token)
	require.NotEmpty(t, payload)

	payload, err = maker.VerifyToken(token)
	require.NoError(t, err)
	require.NotEmpty(t, payload)
	require.Equal(t, username, payload.Username)
	require.Equal(t, role, payload.Role)
	require.WithinDuration(t, issuedAt, payload.IssuedAt, time.Second)
	require.WithinDuration(t, expriedAt, payload.ExpiredAt, time.Second)
}

func TestShortSymmetricKey(t *testing.T) {
	symmetricKey := util.RandomString(30)
	maker, err := NewPasetoMaker(symmetricKey)
	require.Error(t, err)
	require.EqualError(t, err, ErrShortSymmetricKey.Error())
	require.Empty(t, maker)
}

func TestExpiredPasetoToken(t *testing.T) {
	maker, err := NewPasetoMaker(util.RandomString(32))
	require.NoError(t, err)

	username := util.RandomOwner()
	role := DepositorRole
	duration := -time.Minute

	token, payload, err := maker.CreateToken(username, role, duration)
	require.NoError(t, err)
	require.NotEmpty(t, token)
	require.NotEmpty(t, payload)

	payload, err = maker.VerifyToken(token)
	require.Error(t, err)
	require.EqualError(t, err, ErrExpiredToken.Error())
	require.Empty(t, payload)
}
