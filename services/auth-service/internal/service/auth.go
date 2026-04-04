package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gtrade/services/auth-service/internal/repository"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid token")
)

type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
}

type tokenClaims struct {
	Type string `json:"type"`
	jwt.RegisteredClaims
}

type AuthService struct {
	repo       *repository.AuthRepository
	jwtSecret  []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func NewAuthService(repo *repository.AuthRepository, jwtSecret string) *AuthService {
	return &AuthService{
		repo:       repo,
		jwtSecret:  []byte(jwtSecret),
		accessTTL:  15 * time.Minute,
		refreshTTL: 7 * 24 * time.Hour,
	}
}

func (s *AuthService) Register(ctx context.Context, email, password string) (*TokenPair, error) {
	if email == "" || password == "" {
		return nil, fmt.Errorf("email and password are required")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user, err := s.repo.CreateUser(ctx, email, string(hash))
	if err != nil {
		if errors.Is(err, repository.ErrEmailExists) {
			return nil, repository.ErrEmailExists
		}
		return nil, err
	}

	return s.issueTokenPair(ctx, user.ID, user.Email)
}

func (s *AuthService) Login(ctx context.Context, email, password string) (*TokenPair, error) {
	if email == "" || password == "" {
		return nil, ErrInvalidCredentials
	}

	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	return s.issueTokenPair(ctx, user.ID, user.Email)
}

func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*TokenPair, error) {
	claims, err := s.parseToken(refreshToken)
	if err != nil || claims.Type != "refresh" {
		return nil, ErrInvalidToken
	}

	userID, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil {
		return nil, ErrInvalidToken
	}

	stored, err := s.repo.GetRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, err
	}
	if stored == nil || stored.RevokedAt != nil || time.Now().After(stored.ExpiresAt) {
		return nil, ErrInvalidToken
	}

	if err := s.repo.RevokeRefreshToken(ctx, refreshToken); err != nil {
		return nil, err
	}

	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrInvalidToken
	}

	return s.issueTokenPair(ctx, user.ID, user.Email)
}

func (s *AuthService) issueTokenPair(ctx context.Context, userID int64, email string) (*TokenPair, error) {
	now := time.Now()
	accessExp := now.Add(s.accessTTL)
	refreshExp := now.Add(s.refreshTTL)
	subject := strconv.FormatInt(userID, 10)

	accessClaims := tokenClaims{
		Type: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   subject,
			Issuer:    "auth-service",
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(accessExp),
		},
	}
	accessClaims.Audience = []string{"gtrade-api"}

	refreshClaims := tokenClaims{
		Type: "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   subject,
			Issuer:    "auth-service",
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(refreshExp),
		},
	}
	refreshClaims.Audience = []string{"gtrade-auth"}
	refreshClaims.ID = fmt.Sprintf("%d-%d", userID, now.UnixNano())

	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString(s.jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("sign access token: %w", err)
	}

	refreshToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).SignedString(s.jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("sign refresh token: %w", err)
	}

	if err := s.repo.SaveRefreshToken(ctx, userID, refreshToken, refreshExp); err != nil {
		return nil, err
	}

	_ = email
	return &TokenPair{AccessToken: accessToken, RefreshToken: refreshToken, ExpiresIn: int64(s.accessTTL.Seconds())}, nil
}

func (s *AuthService) parseToken(raw string) (*tokenClaims, error) {
	claims := &tokenClaims{}
	token, err := jwt.ParseWithClaims(raw, claims, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return s.jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}
