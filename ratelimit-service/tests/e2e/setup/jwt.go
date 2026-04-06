package setup

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"math/big"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JWTGenerator struct {
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	kid        string
}

func NewJWTGenerator() (*JWTGenerator, error) {

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	return &JWTGenerator{
		privateKey: privateKey,
		publicKey:  &privateKey.PublicKey,
		kid:        "test-key-1",
	}, nil
}

func (g *JWTGenerator) GenerateToken(userID string) (string, error) {
	claims := jwt.MapClaims{
		"sub": userID,
		"iss": "https://auth.example.com",
		"exp": time.Now().Add(1 * time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = g.kid

	return token.SignedString(g.privateKey)
}

func (g *JWTGenerator) GetJWKS() string {

	n := base64.RawURLEncoding.EncodeToString(g.publicKey.N.Bytes())
	e := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(g.publicKey.E)).Bytes())

	return `{
        "keys": [
            {
                "kty": "RSA",
                "use": "sig",
                "alg": "RS256",
                "kid": "` + g.kid + `",
                "n": "` + n + `",
                "e": "` + e + `"
            }
        ]
    }`
}

func (g *JWTGenerator) ApplyJWSToKubernetes() error {
	//jwks := g.GetJWKS()

	return nil
}
