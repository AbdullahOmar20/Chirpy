package auth

import (
	"time"
	"errors"

	"github.com/alexedwards/argon2id"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func HashPassword(password string) (string, error){
	hash, err := argon2id.CreateHash(password, argon2id.DefaultParams)
	if err != nil{
		return "", err
	}

	return hash, nil
}

func CheckPasswordHash(password, hash string) (bool, error){
	match, err := argon2id.ComparePasswordAndHash(password, hash)
	if err != nil{
		return false, err
	}

	return match, nil
}

func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error){
	Token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer: "chirpy-access",
		IssuedAt: jwt.NewNumericDate(time.Now().UTC()),
		ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(expiresIn)),
		Subject: userID.String(),
	})

	TokenString, err := Token.SignedString([]byte(tokenSecret))
	if err != nil{
		return "", err
	}

	return TokenString, nil
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error){
	registrationClaims := jwt.RegisteredClaims{}
	_, err := jwt.ParseWithClaims(tokenString, &registrationClaims, func(token *jwt.Token) (any, error){
		return []byte(tokenSecret), nil
	})
	if err != nil{
		return uuid.Nil, err
	}

	userId, err := registrationClaims.GetSubject()
	if err != nil{
		return uuid.Nil, err
	}

	if expiry, err := registrationClaims.GetExpirationTime(); err != nil || time.Now().After(expiry.Time){
		return uuid.Nil, errors.New("Token is expired")
	}

	userIdUUID, err := uuid.Parse(userId)
	if err != nil{
		return uuid.Nil, err
	}

	return userIdUUID, nil
}