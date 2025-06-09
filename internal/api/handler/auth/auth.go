package auth

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"math/big"
	"net/http"
	"time"

	"github.com/compliance-framework/configuration-service/internal/api"
	"github.com/compliance-framework/configuration-service/internal/api/handler"
	"github.com/compliance-framework/configuration-service/internal/config"
	"github.com/compliance-framework/configuration-service/internal/service/relational"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type UserClaims struct {
	jwt.RegisteredClaims
	GivenName  string `json:"given_name"`
	FamilyName string `json:"family_name"`
}

type AuthHandler struct {
	sugar  *zap.SugaredLogger
	db     *gorm.DB
	config *config.Config
}

type JWK struct {
	Kty string `json:"kty"`
	N   string `json:"n"`
	E   string `json:"e"`
	Alg string `json:"alg,omitempty"`
	Use string `json:"use,omitempty"`
	KID string `json:"kid,omitempty"`
}

func (j *JWK) UnmarhalPublicKey(pubKey *rsa.PublicKey) (*JWK, error) {
	if pubKey == nil {
		return nil, errors.New("public key is nil")
	}

	n := pubKey.N.Bytes()
	e := big.NewInt(int64(pubKey.E)).Bytes()

	return &JWK{
		Kty: "RSA",
		N:   base64.RawURLEncoding.EncodeToString(n),
		E:   base64.RawURLEncoding.EncodeToString(e),
	}, nil
}

func (j *JWK) MarshalPublicKey() (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(j.N)
	if err != nil {
		return nil, err
	}

	eBytes, err := base64.RawURLEncoding.DecodeString(j.E)
	if err != nil {
		return nil, err
	}

	n := new(big.Int).SetBytes(nBytes)
	e := new(big.Int).SetBytes(eBytes)

	return &rsa.PublicKey{
		N: n,
		E: int(e.Int64()),
	}, nil
}

func NewAuthHandler(logger *zap.SugaredLogger, db *gorm.DB, config *config.Config) *AuthHandler {
	return &AuthHandler{
		sugar:  logger,
		db:     db,
		config: config,
	}
}

func (h *AuthHandler) Register(api *echo.Group) {
	api.POST("/login", h.LoginUser)
	api.GET("/publickey.pub", h.GetPublicKeyPEM)
	api.GET("/publickey", h.GetJWK)
}

// LoginUser godoc
// @Summary      Login user

func (h *AuthHandler) LoginUser(ctx echo.Context) error {
	type loginRequest struct {
		Email    string `json:"email" validate:"required,email"`
		Password string `json:"password" validate:"required"`
	}

	type response struct {
		AuthToken string `json:"auth_token"`
	}

	var loginReq loginRequest
	if err := ctx.Bind(&loginReq); err != nil {
		h.sugar.Errorw("Failed to bind login request", "error", err)
		return ctx.JSON(http.StatusBadRequest, api.NewError(err))
	}

	var user relational.User
	if err := h.db.Where("email = ?", loginReq.Email).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			h.sugar.Warnw("User not found", "email", loginReq.Email)
			return ctx.JSON(http.StatusUnauthorized, api.NewError(errors.New("invalid email or password")))
		}
		h.sugar.Errorw("Failed to query user", "error", err)
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	}

	if !user.CheckPassword(loginReq.Password) {
		h.sugar.Warnw("Invalid password attempt", "email", loginReq.Email)
		return ctx.JSON(http.StatusUnauthorized, api.NewError(errors.New("invalid email or password")))
	}

	token, err := generateJWTToken(&user, h.config.JWTPrivateKey)
	if err != nil {
		h.sugar.Errorw("Failed to generate JWT token", "error", err)
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	}

	ret := response{
		AuthToken: *token,
	}

	return ctx.JSON(http.StatusOK, handler.GenericDataResponse[response]{Data: ret})
}

func (h *AuthHandler) GetPublicKeyPEM(ctx echo.Context) error {
	pubASN1, err := x509.MarshalPKIXPublicKey(h.config.JWTPublicKey)
	if err != nil {
		h.sugar.Errorw("Failed to marshal public key", "error", err)
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	}
	pubPem := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubASN1,
	})

	return ctx.String(http.StatusOK, string(pubPem))
}

func (h *AuthHandler) GetJWK(ctx echo.Context) error {
	jwk := &JWK{}
	jwk, err := jwk.UnmarhalPublicKey(h.config.JWTPublicKey)
	if err != nil {
		h.sugar.Errorw("Failed to unmarshal public key to JWK", "error", err)
		return ctx.JSON(http.StatusInternalServerError, api.NewError(err))
	}
	return ctx.JSON(http.StatusOK, jwk)
}

func generateJWTToken(user *relational.User, privateKey *rsa.PrivateKey) (*string, error) {
	now := time.Now()
	claims := UserClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "compliance-framework",
			Subject:   user.Email,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(24 * time.Hour)),
			NotBefore: jwt.NewNumericDate(now),
		},
		GivenName:  user.FirstName,
		FamilyName: user.LastName,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenString, err := token.SignedString(privateKey)
	if err != nil {
		return nil, err
	}

	return &tokenString, nil
}
