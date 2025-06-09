package auth

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
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

func NewAuthHandler(logger *zap.SugaredLogger, db *gorm.DB, config *config.Config) *AuthHandler {
	return &AuthHandler{
		sugar:  logger,
		db:     db,
		config: config,
	}
}

func (h *AuthHandler) Register(api *echo.Group) {
	api.POST("/login", h.LoginUser)
	api.GET("/publickey", h.GetPublicKey)
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

func (h *AuthHandler) GetPublicKey(ctx echo.Context) error {
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
