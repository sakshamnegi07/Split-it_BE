package models

import "github.com/golang-jwt/jwt"

type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
}

type LoginCredentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type Claims struct {
	Id    uint   `json:"id"`
	Email string `json:"email"`
	jwt.StandardClaims
}
