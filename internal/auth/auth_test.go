package auth

import (
	"net/http"
	"strings"
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

func TestGetBearerToken(t *testing.T){
	header := http.Header{}
	token := "Bearer ==qwertyuioasdffgshsf"
	header.Add("Authorization", token)

	actualToken, err := GetBearerToken(header)
	if err != nil{
		t.Errorf("error getting bearer token: %v", err)
	}

	expectedToken, _ := strings.CutPrefix(token, "Bearer ")
	if len(actualToken) == 0 || expectedToken != actualToken{
		t.Errorf("unexpected token: Expected: %s \n Actual: %s", expectedToken, actualToken)
	}
}
func TestGetBearerTokenWithNoAuthorizationHeader(t *testing.T){
	header := http.Header{}

	actualToken, err := GetBearerToken(header)

	if err == nil || len(actualToken) > 0{
		t.Errorf("expected error or token to be empty. Found: %s", actualToken)
	}
	
}
func TestGetBearerTokenWithInvalidTokenPrefix(t *testing.T){
	header := http.Header{}
	token := "==qwertyuioasdffgshsf"
	header.Add("Authorization", token)

	actualToken, err := GetBearerToken(header)
	if err == nil || len(actualToken) > 0{
		t.Errorf("expected invalid token error or token to be empty. Found: %s", actualToken)
	}
}