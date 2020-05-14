package bless

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJSONPublicKeyToSign(t *testing.T) {
	r := require.New(t)

	pub, _, err := ed25519.GenerateKey(rand.Reader)
	r.NoError(err)

	pubToSign := &PublicKeyToSign{key: pub}

	data, err := json.Marshal(pubToSign)
	r.NoError(err)

	newPubToSign := &PublicKeyToSign{}
	err = json.Unmarshal(data, newPubToSign)
	r.NoError(err)

	r.Equal(pubToSign.key, newPubToSign.key)
}
