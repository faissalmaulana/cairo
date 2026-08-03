package token_service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

var (
	ErrInvalidToken   = errors.New("invalid token")
	ErrExpiredToken   = errors.New("expired token")
	ErrRevokedToken   = errors.New("revoked token")
	ErrRefreshRevoked = errors.New("refresh token revoked")
	ErrNoRedis        = errors.New("no redis client")
)

const (
	issuer      = "cairo"
	audience    = "cairo-api"
	denylistKey = "denylist:%s"
	refreshKey  = "refresh:%s"
)

type TokenService struct {
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
	rdb        *redis.Client
}

func NewTokenService(secret string, accessTTL, refreshTTL time.Duration, rdb *redis.Client) *TokenService {
	return &TokenService{
		secret:     []byte(secret),
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
		rdb:        rdb,
	}
}

type Claims struct {
	jwt.RegisteredClaims
}

func (ts *TokenService) newToken(userID string, ttl time.Duration) (string, *Claims, error) {
	now := time.Now()
	claims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			Issuer:    issuer,
			Audience:  jwt.ClaimStrings{audience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			ID:        uuid.NewString(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(ts.secret)
	if err != nil {
		return "", nil, err
	}

	return signed, claims, nil
}

func (ts *TokenService) GenerateAccessToken(userID string) (string, *Claims, error) {
	return ts.newToken(userID, ts.accessTTL)
}

func (ts *TokenService) GenerateRefreshToken(userID string) (string, *Claims, error) {
	return ts.newToken(userID, ts.refreshTTL)
}

func (ts *TokenService) AccessTTL() time.Duration { return ts.accessTTL }

func (ts *TokenService) RefreshTTL() time.Duration { return ts.refreshTTL }

func (ts *TokenService) parse(raw string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(raw, &Claims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return ts.secret, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

func (ts *TokenService) ParseAccessToken(raw string) (*Claims, error) {
	return ts.parse(raw)
}

func (ts *TokenService) ParseRefreshToken(raw string) (*Claims, error) {
	return ts.parse(raw)
}

func (ts *TokenService) ExtractJTI(raw string) (string, error) {
	token, _, err := new(jwt.Parser).ParseUnverified(raw, &Claims{})
	if err != nil {
		return "", ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return "", ErrInvalidToken
	}

	if claims.ID == "" {
		return "", ErrInvalidToken
	}

	return claims.ID, nil
}

func (ts *TokenService) IsRevoked(ctx context.Context, jti string) (bool, error) {
	if ts.rdb == nil {
		return false, ErrNoRedis
	}

	n, err := ts.rdb.Exists(ctx, fmt.Sprintf(denylistKey, jti)).Result()
	if err != nil {
		return false, err
	}

	return n > 0, nil
}

func (ts *TokenService) Revoke(ctx context.Context, jti string, remaining time.Duration) error {
	if ts.rdb == nil {
		return ErrNoRedis
	}

	if remaining <= 0 {
		return nil
	}

	return ts.rdb.Set(ctx, fmt.Sprintf(denylistKey, jti), 1, remaining).Err()
}

func (ts *TokenService) StoreRefresh(ctx context.Context, jti, userID string) error {
	if ts.rdb == nil {
		return ErrNoRedis
	}

	return ts.rdb.Set(ctx, fmt.Sprintf(refreshKey, jti), userID, ts.refreshTTL).Err()
}

func (ts *TokenService) ConsumeRefresh(ctx context.Context, jti string) (string, error) {
	if ts.rdb == nil {
		return "", ErrNoRedis
	}

	userID, err := ts.rdb.GetDel(ctx, fmt.Sprintf(refreshKey, jti)).Result()
	if err == redis.Nil {
		return "", ErrRefreshRevoked
	}
	if err != nil {
		return "", err
	}

	return userID, nil
}
