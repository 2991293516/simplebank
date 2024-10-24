package util

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHashedPassword(t *testing.T) {
	password := RandomString(6)

	hashedPassword, err := HashedPassword(password)
	require.NoError(t, err)
	require.NotEmpty(t, hashedPassword)

	ok := CheckPassword(password, hashedPassword)
	require.True(t, ok)

	wrongPassword := RandomString(6)
	ok = CheckPassword(wrongPassword, hashedPassword)
	require.False(t, ok)
}
