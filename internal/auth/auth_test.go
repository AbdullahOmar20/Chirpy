package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestMakeJWT(t *testing.T){
	userId := uuid.New()
	secret := "qwertyuiopasdfghjklzxcvbnmQWERTYUIOPASDFGHJKLZXCVBNM"
	token, err := MakeJWT(userId, secret, 5 * time.Minute)
	if err != nil || len(token) == 0{
		t.Errorf("error creating token: %v", err)
	}
}
func TestValidateToken(t *testing.T){
	userId := uuid.New()
	secret := "qwertyuiopasdfghjklzxcvbnmQWERTYUIOPASDFGHJKLZXCVBNM"
	token, err := MakeJWT(userId, secret, 5 * time.Minute)
	if err != nil || len(token) == 0{
		t.Errorf("error creating token: %v", err)
	}

	validatedUserId, err := ValidateJWT(token, secret)
	if err != nil{
		t.Errorf("error validating token: %v", err)
	}
	
	if  validatedUserId != userId{
		t.Errorf("error wrong user id. Expected: %s, Found: %s", userId.String(), validatedUserId.String())
	}
}
func TestValidateTokenExpiryDate(t *testing.T){
	userId := uuid.New()
	secret := "qwertyuiopasdfghjklzxcvbnmQWERTYUIOPASDFGHJKLZXCVBNM"
	token, err := MakeJWT(userId, secret, -5 * time.Minute)
	if err != nil || len(token) == 0{
		t.Errorf("error creating token: %v", err)
	}

	validatedUserId, err := ValidateJWT(token, secret)
	if err == nil || validatedUserId != uuid.Nil{
		t.Errorf("error validating token: Expected that token must be expired")
	}
}