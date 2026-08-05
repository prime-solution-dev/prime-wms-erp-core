package models

import "github.com/golang-jwt/jwt/v5"

type AuthenJWTClaims struct {
	CompanyCode string `json:"company_code"`
	User        string `json:"user"`
	jwt.RegisteredClaims
}
