package account

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/xuecangming/onedrive-storage/internal/common/errors"
	"github.com/xuecangming/onedrive-storage/internal/common/types"
	"github.com/xuecangming/onedrive-storage/internal/infrastructure/onedrive"
	"github.com/xuecangming/onedrive-storage/internal/repository"
)

// Service provides account management operations
type Service struct {
	repo *repository.AccountRepository
}

// NewService creates a new account service
func NewService(repo *repository.AccountRepository) *Service {
	return &Service{repo: repo}
}

// Create creates a new storage account
func (s *Service) Create(ctx context.Context, account *types.StorageAccount) error {
	// Check if account with same email exists
	existing, err := s.repo.GetAccountByEmail(ctx, account.Email)
	if err != nil && err != sql.ErrNoRows {
		return errors.InternalError(err.Error())
	}
	if existing != nil {
		return errors.NewAppError("ACCOUNT_EXISTS", "Account with this email already exists", 409)
	}

	// Generate UUID if not provided
	if account.ID == "" {
		account.ID = uuid.New().String()
	}

	// Set default values
	if account.Status == "" {
		account.Status = "pending"
	}
	if account.Priority == 0 {
		account.Priority = 10
	}

	// Create account
	if err := s.repo.Create(ctx, account); err != nil {
		return errors.InternalError(err.Error())
	}

	return nil
}

// Get retrieves an account by ID
func (s *Service) Get(ctx context.Context, id string) (*types.StorageAccount, error) {
	account, err := s.repo.Get(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.NewAppError("ACCOUNT_NOT_FOUND", "Account not found", 404)
		}
		return nil, errors.InternalError(err.Error())
	}

	return account, nil
}

// List retrieves all accounts
func (s *Service) List(ctx context.Context) ([]*types.StorageAccount, error) {
	accounts, err := s.repo.List(ctx)
	if err != nil {
		return nil, errors.InternalError(err.Error())
	}

	return accounts, nil
}

// Update updates an account
func (s *Service) Update(ctx context.Context, account *types.StorageAccount) error {
	// Check if account exists
	existing, err := s.repo.Get(ctx, account.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			return errors.NewAppError("ACCOUNT_NOT_FOUND", "Account not found", 404)
		}
		return errors.InternalError(err.Error())
	}

	// Preserve created_at
	account.CreatedAt = existing.CreatedAt

	// Update account
	if err := s.repo.Update(ctx, account); err != nil {
		return errors.InternalError(err.Error())
	}

	return nil
}

// Delete deletes an account
func (s *Service) Delete(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		if err == sql.ErrNoRows {
			return errors.NewAppError("ACCOUNT_NOT_FOUND", "Account not found", 404)
		}
		return errors.InternalError(err.Error())
	}

	return nil
}

// RefreshToken refreshes an account's access token
func (s *Service) RefreshToken(ctx context.Context, id string) error {
	account, err := s.repo.Get(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return errors.NewAppError("ACCOUNT_NOT_FOUND", "Account not found", 404)
		}
		return errors.InternalError(err.Error())
	}

	if account.RefreshToken == "" {
		return errors.NewAppError("NO_REFRESH_TOKEN", "No refresh token available", 400)
	}

	// Create auth client
	authConfig := onedrive.AuthConfig{
		ClientID:     account.ClientID,
		ClientSecret: account.ClientSecret,
		TenantID:     account.TenantID,
	}
	auth := onedrive.NewAuth(authConfig)

	// Refresh token
	tokenResp, err := auth.RefreshToken(ctx, account.RefreshToken)
	if err != nil {
		// Mark account as error
		s.repo.UpdateStatus(ctx, id, "error", err.Error())
		return errors.UpstreamError(err.Error())
	}

	// Update account tokens
	expiresAt := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	if err := s.repo.UpdateToken(ctx, id, tokenResp.AccessToken, tokenResp.RefreshToken, expiresAt); err != nil {
		return errors.InternalError(err.Error())
	}

	return nil
}

// SyncSpaceInfo syncs space information from OneDrive
func (s *Service) SyncSpaceInfo(ctx context.Context, id string) error {
	account, err := s.repo.Get(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return errors.NewAppError("ACCOUNT_NOT_FOUND", "Account not found", 404)
		}
		return errors.InternalError(err.Error())
	}

	// Check if token needs refresh
	if time.Now().After(account.TokenExpires.Add(-5 * time.Minute)) {
		if err := s.RefreshToken(ctx, id); err != nil {
			return err
		}
		// Reload account with new token
		account, err = s.repo.Get(ctx, id)
		if err != nil {
			return errors.InternalError(err.Error())
		}
	}

	// Create OneDrive client
	client := onedrive.NewClient(account.AccessToken)

	// Get drive info
	drive, err := client.GetDrive(ctx)
	if err != nil {
		// Mark account as error
		s.repo.UpdateStatus(ctx, id, "error", err.Error())
		return errors.UpstreamError(err.Error())
	}

	// Update space info
	if err := s.repo.UpdateSpaceInfo(ctx, id, drive.Quota.Total, drive.Quota.Used); err != nil {
		return errors.InternalError(err.Error())
	}

	// Mark account as active
	s.repo.UpdateStatus(ctx, id, "active", "")

	return nil
}

// GetActiveAccounts retrieves all active accounts
func (s *Service) GetActiveAccounts(ctx context.Context) ([]*types.StorageAccount, error) {
	accounts, err := s.repo.GetActiveAccounts(ctx)
	if err != nil {
		return nil, errors.InternalError(err.Error())
	}

	return accounts, nil
}

// EnsureTokenValid ensures account has valid token, refreshing if needed
func (s *Service) EnsureTokenValid(ctx context.Context, id string) error {
	account, err := s.repo.Get(ctx, id)
	if err != nil {
		return errors.InternalError(err.Error())
	}

	// Check if token expires within 5 minutes
	if time.Now().After(account.TokenExpires.Add(-5 * time.Minute)) {
		return s.RefreshToken(ctx, id)
	}

	return nil
}
