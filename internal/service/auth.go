package service

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"blog/internal/domain"
	"blog/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	userRepo    *repository.UserRepository
	sessionRepo *repository.SessionRepository
}

func NewAuthService(userRepo *repository.UserRepository, sessionRepo *repository.SessionRepository) *AuthService {
	return &AuthService{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
	}
}

// Login 校验管理员登录并创建 Session
func (s *AuthService) Login(username, password string) (string, *domain.User, error) {
	u, err := s.userRepo.GetByUsername(username)
	if err != nil {
		return "", nil, err
	}
	if u == nil {
		return "", nil, errors.New("用户名或密码错误")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return "", nil, errors.New("用户名或密码错误")
	}

	// 生成安全 Session Token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", nil, fmt.Errorf("failed to generate session token: %w", err)
	}
	token := hex.EncodeToString(tokenBytes)

	// 存入 BoltDB，有效期 7 天
	expires := time.Now().Add(7 * 24 * time.Hour)
	err = s.sessionRepo.Save(token, &repository.SessionData{
		UserID:    u.ID,
		Username:  u.Username,
		ExpiresAt: expires,
	})
	if err != nil {
		return "", nil, fmt.Errorf("failed to save session: %w", err)
	}

	return token, u, nil
}

// ValidateSession 检查 Session 是否有效并返回当前管理员
func (s *AuthService) ValidateSession(token string) (*domain.User, error) {
	if token == "" {
		return nil, errors.New("empty session token")
	}

	data, err := s.sessionRepo.Get(token)
	if err != nil {
		return nil, err
	}

	return s.userRepo.GetByID(data.UserID)
}

// Logout 注销当前会话
func (s *AuthService) Logout(token string) error {
	return s.sessionRepo.Delete(token)
}
