package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"minimart-pos/backend/internal/auth"
	"minimart-pos/backend/internal/config"
	"minimart-pos/backend/internal/domain"
	"minimart-pos/backend/internal/store"
)

type AuthService struct {
	store *store.DB
	cfg   *config.Config
	now   func() time.Time
}

func NewAuthService(st *store.DB, cfg *config.Config) *AuthService {
	return &AuthService{store: st, cfg: cfg, now: time.Now}
}

type TokenPair struct {
	AccessToken  string
	RefreshToken string // raw; caller puts it in an httpOnly cookie
	ExpiresIn    int    // access token seconds
	User         store.User
}

// Register creates a staff account. Caller must already be authorised
// (manager+) — enforced by the handler's RBAC middleware.
func (s *AuthService) Register(ctx context.Context, storeID uuid.UUID, username, password string, role domain.Role) (store.User, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return store.User{}, domain.Validation("ต้องระบุชื่อผู้ใช้")
	}
	if len(password) < 6 {
		return store.User{}, domain.Validation("รหัสผ่านต้องยาวอย่างน้อย 6 ตัว")
	}
	if !role.Valid() {
		return store.User{}, domain.Validation("บทบาทไม่ถูกต้อง")
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return store.User{}, err
	}
	u, err := s.store.CreateUser(ctx, store.CreateUserParams{
		StoreID:      storeID,
		Username:     username,
		PasswordHash: hash,
		Role:         string(role),
	})
	if err != nil {
		if isUniqueViolation(err) {
			return store.User{}, domain.Conflict("ชื่อผู้ใช้นี้มีอยู่แล้ว")
		}
		return store.User{}, err
	}
	return u, nil
}

func (s *AuthService) Login(ctx context.Context, username, password string) (TokenPair, error) {
	u, err := s.store.GetActiveUserByUsername(ctx, strings.TrimSpace(username))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return TokenPair{}, domain.ErrInvalidCredentials
		}
		return TokenPair{}, err
	}
	ok, err := auth.VerifyPassword(password, u.PasswordHash)
	if err != nil || !ok {
		return TokenPair{}, domain.ErrInvalidCredentials
	}
	return s.issuePair(ctx, u)
}

func (s *AuthService) Refresh(ctx context.Context, rawRefresh string) (TokenPair, error) {
	rt, err := s.store.GetRefreshTokenByHash(ctx, auth.HashRefreshToken(rawRefresh))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return TokenPair{}, domain.ErrUnauthorized
		}
		return TokenPair{}, err
	}
	if rt.RevokedAt.Valid || rt.ExpiresAt.Time.Before(s.now()) {
		return TokenPair{}, domain.ErrUnauthorized
	}
	u, err := s.store.GetUserByID(ctx, rt.UserID)
	if err != nil {
		return TokenPair{}, domain.ErrUnauthorized
	}
	if !u.IsActive {
		return TokenPair{}, domain.ErrUnauthorized
	}

	// Rotate: mint a new refresh token, then revoke the old one pointing to it.
	pair, newID, err := s.mintPair(ctx, u)
	if err != nil {
		return TokenPair{}, err
	}
	if err := s.store.RevokeRefreshToken(ctx, store.RevokeRefreshTokenParams{
		ID:         rt.ID,
		ReplacedBy: store.PgUUID(newID),
	}); err != nil {
		return TokenPair{}, err
	}
	return pair, nil
}

func (s *AuthService) Logout(ctx context.Context, rawRefresh string) error {
	rt, err := s.store.GetRefreshTokenByHash(ctx, auth.HashRefreshToken(rawRefresh))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil // already gone; treat as success
		}
		return err
	}
	return s.store.RevokeRefreshToken(ctx, store.RevokeRefreshTokenParams{ID: rt.ID})
}

func (s *AuthService) GetUser(ctx context.Context, id uuid.UUID) (store.User, error) {
	return s.store.GetUserByID(ctx, id)
}

// ChangePassword verifies the current password, sets a new one, and revokes all
// of the user's refresh tokens (forcing other sessions to re-login).
func (s *AuthService) ChangePassword(ctx context.Context, userID uuid.UUID, current, newPassword string) error {
	u, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		return domain.ErrUnauthorized
	}
	ok, err := auth.VerifyPassword(current, u.PasswordHash)
	if err != nil || !ok {
		return domain.ErrInvalidCredentials
	}
	if len(newPassword) < 6 {
		return domain.Validation("รหัสผ่านใหม่ต้องยาวอย่างน้อย 6 ตัว")
	}
	hash, err := auth.HashPassword(newPassword)
	if err != nil {
		return err
	}
	if err := s.store.UpdateUserPassword(ctx, store.UpdateUserPasswordParams{ID: userID, PasswordHash: hash}); err != nil {
		return err
	}
	// Invalidate existing sessions after a password change.
	_ = s.store.RevokeAllUserTokens(ctx, userID)
	return nil
}

func (s *AuthService) issuePair(ctx context.Context, u store.User) (TokenPair, error) {
	pair, _, err := s.mintPair(ctx, u)
	return pair, err
}

// mintPair issues an access token and persists a new refresh token, returning
// the pair and the new refresh-token row id (for rotation bookkeeping).
func (s *AuthService) mintPair(ctx context.Context, u store.User) (TokenPair, uuid.UUID, error) {
	access, err := auth.IssueAccessToken(s.cfg.JWTAccessSecret, u.ID, u.StoreID, u.Role, s.cfg.AccessTokenTTL, s.now())
	if err != nil {
		return TokenPair{}, uuid.Nil, err
	}
	raw, hash, err := auth.NewRefreshToken()
	if err != nil {
		return TokenPair{}, uuid.Nil, err
	}
	rt, err := s.store.CreateRefreshToken(ctx, store.CreateRefreshTokenParams{
		UserID:    u.ID,
		StoreID:   u.StoreID,
		TokenHash: hash,
		ExpiresAt: store.PgTime(s.now().Add(s.cfg.RefreshTokenTTL)),
	})
	if err != nil {
		return TokenPair{}, uuid.Nil, err
	}
	return TokenPair{
		AccessToken:  access,
		RefreshToken: raw,
		ExpiresIn:    int(s.cfg.AccessTokenTTL.Seconds()),
		User:         u,
	}, rt.ID, nil
}
