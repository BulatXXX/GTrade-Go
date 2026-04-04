package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
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
	ErrUserNotFound       = errors.New("user not found")
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
	resetTTL   time.Duration
	verifyTTL  time.Duration
}

func NewAuthService(repo *repository.AuthRepository, jwtSecret string) *AuthService {
	return &AuthService{
		repo:       repo,
		jwtSecret:  []byte(jwtSecret),
		accessTTL:  15 * time.Minute,
		refreshTTL: 7 * 24 * time.Hour,
		resetTTL:   time.Hour,
		verifyTTL:  24 * time.Hour,
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

	return s.issueTokenPair(ctx, user.ID)
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

	return s.issueTokenPair(ctx, user.ID)
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

	return s.issueTokenPair(ctx, user.ID)
}

func (s *AuthService) RequestPasswordReset(ctx context.Context, email string) (string, error) {
	if email == "" {
		return "", ErrUserNotFound
	}

	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return "", err
	}
	if user == nil {
		return "", nil
	}

	token, err := generateOpaqueToken()
	if err != nil {
		return "", err
	}

	if err := s.repo.SavePasswordResetToken(ctx, user.ID, token, time.Now().Add(s.resetTTL)); err != nil {
		return "", err
	}

	return token, nil
}

func (s *AuthService) ConfirmPasswordReset(ctx context.Context, token, newPassword string) error {
	if token == "" || newPassword == "" {
		return ErrInvalidToken
	}

	resetToken, err := s.repo.GetPasswordResetToken(ctx, token)
	if err != nil {
		return err
	}
	if resetToken == nil || resetToken.UsedAt != nil || time.Now().After(resetToken.ExpiresAt) {
		return ErrInvalidToken
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	if err := s.repo.UpdateUserPassword(ctx, resetToken.UserID, string(hash)); err != nil {
		return err
	}
	if err := s.repo.UsePasswordResetToken(ctx, token); err != nil {
		return err
	}
	return nil
}

func (s *AuthService) RequestEmailVerification(ctx context.Context, email string) (string, error) {
	if email == "" {
		return "", ErrUserNotFound
	}

	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return "", err
	}
	if user == nil {
		return "", ErrUserNotFound
	}
	if user.EmailVerified {
		return "", nil
	}

	token, err := generateOpaqueToken()
	if err != nil {
		return "", err
	}

	if err := s.repo.SaveEmailVerificationToken(ctx, user.ID, token, time.Now().Add(s.verifyTTL)); err != nil {
		return "", err
	}

	return token, nil
}

func (s *AuthService) VerifyEmail(ctx context.Context, token string) error {
	if token == "" {
		return ErrInvalidToken
	}

	verifyToken, err := s.repo.GetEmailVerificationToken(ctx, token)
	if err != nil {
		return err
	}
	if verifyToken == nil || verifyToken.UsedAt != nil || time.Now().After(verifyToken.ExpiresAt) {
		return ErrInvalidToken
	}

	if err := s.repo.MarkEmailVerified(ctx, verifyToken.UserID); err != nil {
		return err
	}
	if err := s.repo.UseEmailVerificationToken(ctx, token); err != nil {
		return err
	}
	return nil
}

func (s *AuthService) issueTokenPair(ctx context.Context, userID int64) (*TokenPair, error) {
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

func generateOpaqueToken() (string, error) {
	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	return hex.EncodeToString(raw[:]), nil
}
