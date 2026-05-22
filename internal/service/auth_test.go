package service

import (
	"context"
	"testing"

	"github.com/Bajusz15/go-backend-home-assignment/internal/model"
	"github.com/Bajusz15/go-backend-home-assignment/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

type mockUserRepo struct {
	users map[string]*model.User
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{users: make(map[string]*model.User)}
}

func (m *mockUserRepo) Create(_ context.Context, user *model.User) error {
	if _, exists := m.users[user.Email]; exists {
		return repository.ErrDuplicateEmail
	}
	user.ID = "test-user-id"
	m.users[user.Email] = user
	return nil
}

func (m *mockUserRepo) FindByEmail(_ context.Context, email string) (*model.User, error) {
	u, ok := m.users[email]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return u, nil
}

func (m *mockUserRepo) FindByID(_ context.Context, id string) (*model.User, error) {
	for _, u := range m.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, repository.ErrNotFound
}

func TestAuthService_Register(t *testing.T) {
	repo := newMockUserRepo()
	svc := NewAuthService(repo, "test-secret")

	t.Run("success", func(t *testing.T) {
		resp, err := svc.Register(context.Background(), RegisterInput{
			Email:    "test@example.com",
			Password: "password123",
			Name:     "Test User",
		})

		require.NoError(t, err)
		assert.NotEmpty(t, resp.Token)
		assert.Equal(t, "test@example.com", resp.User.Email)
		assert.Equal(t, model.RoleCustomer, resp.User.Role)
		assert.Equal(t, "Test User", resp.User.Name)
	})

	t.Run("duplicate email", func(t *testing.T) {
		_, err := svc.Register(context.Background(), RegisterInput{
			Email:    "test@example.com",
			Password: "password123",
			Name:     "Another User",
		})

		assert.ErrorIs(t, err, ErrEmailTaken)
	})
}

func TestAuthService_Login(t *testing.T) {
	repo := newMockUserRepo()
	svc := NewAuthService(repo, "test-secret")

	hash, _ := bcrypt.GenerateFromPassword([]byte("correctpassword"), bcrypt.DefaultCost)
	repo.users["user@example.com"] = &model.User{
		ID:           "user-1",
		Email:        "user@example.com",
		PasswordHash: string(hash),
		Name:         "Test User",
		Role:         model.RoleCustomer,
	}

	t.Run("success", func(t *testing.T) {
		resp, err := svc.Login(context.Background(), LoginInput{
			Email:    "user@example.com",
			Password: "correctpassword",
		})

		require.NoError(t, err)
		assert.NotEmpty(t, resp.Token)
		assert.Equal(t, "user-1", resp.User.ID)
	})

	t.Run("wrong password", func(t *testing.T) {
		_, err := svc.Login(context.Background(), LoginInput{
			Email:    "user@example.com",
			Password: "wrongpassword",
		})

		assert.ErrorIs(t, err, ErrInvalidCredentials)
	})

	t.Run("user not found", func(t *testing.T) {
		_, err := svc.Login(context.Background(), LoginInput{
			Email:    "nonexistent@example.com",
			Password: "password",
		})

		assert.ErrorIs(t, err, ErrInvalidCredentials)
	})
}
