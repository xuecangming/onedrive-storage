package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/xuecangming/onedrive-storage/internal/common/types"
)

// AccountRepository handles storage account data access
type AccountRepository struct {
	db *sql.DB
}

// NewAccountRepository creates a new account repository
func NewAccountRepository(db *sql.DB) *AccountRepository {
	return &AccountRepository{db: db}
}

// Create creates a new storage account
func (r *AccountRepository) Create(ctx context.Context, account *types.StorageAccount) error {
	query := `
		INSERT INTO storage_accounts (
			id, name, email, client_id, client_secret, tenant_id,
			refresh_token, access_token, token_expires,
			total_space, used_space, status, priority,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`

	now := time.Now()
	_, err := r.db.ExecContext(ctx, query,
		account.ID, account.Name, account.Email,
		account.ClientID, account.ClientSecret, account.TenantID,
		account.RefreshToken, account.AccessToken, account.TokenExpires,
		account.TotalSpace, account.UsedSpace,
		account.Status, account.Priority,
		now, now,
	)

	if err != nil {
		return err
	}

	account.CreatedAt = now
	account.UpdatedAt = now

	return nil
}

// Get retrieves an account by ID
func (r *AccountRepository) Get(ctx context.Context, id string) (*types.StorageAccount, error) {
	query := `
		SELECT id, name, email, client_id, client_secret, tenant_id,
		       COALESCE(refresh_token, ''), COALESCE(access_token, ''), token_expires,
		       COALESCE(total_space, 0), COALESCE(used_space, 0), 
		       COALESCE(status, 'pending'), COALESCE(priority, 0),
		       last_sync, error_message, created_at, updated_at
		FROM storage_accounts
		WHERE id = $1
	`

	account := &types.StorageAccount{}
	var lastSync, tokenExpires sql.NullTime
	var errorMessage sql.NullString

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&account.ID, &account.Name, &account.Email,
		&account.ClientID, &account.ClientSecret, &account.TenantID,
		&account.RefreshToken, &account.AccessToken, &tokenExpires,
		&account.TotalSpace, &account.UsedSpace,
		&account.Status, &account.Priority,
		&lastSync, &errorMessage,
		&account.CreatedAt, &account.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	if lastSync.Valid {
		account.LastSync = lastSync.Time
	}
	if tokenExpires.Valid {
		account.TokenExpires = tokenExpires.Time
	}
	if errorMessage.Valid {
		account.ErrorMessage = errorMessage.String
	}

	return account, nil
}

// List retrieves all accounts
func (r *AccountRepository) List(ctx context.Context) ([]*types.StorageAccount, error) {
	query := `
		SELECT id, name, email, client_id, client_secret, tenant_id,
		       COALESCE(refresh_token, ''), COALESCE(access_token, ''), token_expires,
		       COALESCE(total_space, 0), COALESCE(used_space, 0), 
		       COALESCE(status, 'pending'), COALESCE(priority, 0),
		       last_sync, error_message, created_at, updated_at
		FROM storage_accounts
		WHERE id != '00000000-0000-0000-0000-000000000000'
		ORDER BY priority DESC, created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []*types.StorageAccount
	for rows.Next() {
		account := &types.StorageAccount{}
		var lastSync, tokenExpires sql.NullTime
		var errorMessage sql.NullString

		if err := rows.Scan(
			&account.ID, &account.Name, &account.Email,
			&account.ClientID, &account.ClientSecret, &account.TenantID,
			&account.RefreshToken, &account.AccessToken, &tokenExpires,
			&account.TotalSpace, &account.UsedSpace,
			&account.Status, &account.Priority,
			&lastSync, &errorMessage,
			&account.CreatedAt, &account.UpdatedAt,
		); err != nil {
			return nil, err
		}

		if lastSync.Valid {
			account.LastSync = lastSync.Time
		}
		if tokenExpires.Valid {
			account.TokenExpires = tokenExpires.Time
		}
		if errorMessage.Valid {
			account.ErrorMessage = errorMessage.String
		}

		accounts = append(accounts, account)
	}

	return accounts, nil
}

// Update updates an account
func (r *AccountRepository) Update(ctx context.Context, account *types.StorageAccount) error {
	query := `
		UPDATE storage_accounts
		SET name = $2, email = $3, client_id = $4, client_secret = $5, tenant_id = $6,
		    refresh_token = $7, access_token = $8, token_expires = $9,
		    total_space = $10, used_space = $11, status = $12, priority = $13,
		    last_sync = $14, error_message = $15, updated_at = $16
		WHERE id = $1
	`

	now := time.Now()
	_, err := r.db.ExecContext(ctx, query,
		account.ID, account.Name, account.Email,
		account.ClientID, account.ClientSecret, account.TenantID,
		account.RefreshToken, account.AccessToken, account.TokenExpires,
		account.TotalSpace, account.UsedSpace,
		account.Status, account.Priority,
		account.LastSync, account.ErrorMessage,
		now,
	)

	if err != nil {
		return err
	}

	account.UpdatedAt = now
	return nil
}

// UpdateToken updates account tokens
func (r *AccountRepository) UpdateToken(ctx context.Context, id, accessToken, refreshToken string, expiresAt time.Time) error {
	query := `
		UPDATE storage_accounts
		SET access_token = $2, refresh_token = $3, token_expires = $4, updated_at = $5
		WHERE id = $1
	`

	_, err := r.db.ExecContext(ctx, query, id, accessToken, refreshToken, expiresAt, time.Now())
	return err
}

// UpdateSpaceInfo updates account space information
func (r *AccountRepository) UpdateSpaceInfo(ctx context.Context, id string, totalSpace, usedSpace int64) error {
	query := `
		UPDATE storage_accounts
		SET total_space = $2, used_space = $3, last_sync = $4, updated_at = $5
		WHERE id = $1
	`

	now := time.Now()
	_, err := r.db.ExecContext(ctx, query, id, totalSpace, usedSpace, now, now)
	return err
}

// UpdateStatus updates account status
func (r *AccountRepository) UpdateStatus(ctx context.Context, id, status, errorMessage string) error {
	query := `
		UPDATE storage_accounts
		SET status = $2, error_message = $3, updated_at = $4
		WHERE id = $1
	`

	_, err := r.db.ExecContext(ctx, query, id, status, errorMessage, time.Now())
	return err
}

// Delete deletes an account
func (r *AccountRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM storage_accounts WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// GetActiveAccounts retrieves all active accounts
func (r *AccountRepository) GetActiveAccounts(ctx context.Context) ([]*types.StorageAccount, error) {
	query := `
		SELECT id, name, email, client_id, client_secret, tenant_id,
		       COALESCE(refresh_token, ''), COALESCE(access_token, ''), token_expires,
		       COALESCE(total_space, 0), COALESCE(used_space, 0), 
		       COALESCE(status, 'pending'), COALESCE(priority, 0),
		       last_sync, error_message, created_at, updated_at
		FROM storage_accounts
		WHERE status = 'active' AND id != '00000000-0000-0000-0000-000000000000'
		ORDER BY priority DESC, used_space ASC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []*types.StorageAccount
	for rows.Next() {
		account := &types.StorageAccount{}
		var lastSync, tokenExpires sql.NullTime
		var errorMessage sql.NullString

		if err := rows.Scan(
			&account.ID, &account.Name, &account.Email,
			&account.ClientID, &account.ClientSecret, &account.TenantID,
			&account.RefreshToken, &account.AccessToken, &tokenExpires,
			&account.TotalSpace, &account.UsedSpace,
			&account.Status, &account.Priority,
			&lastSync, &errorMessage,
			&account.CreatedAt, &account.UpdatedAt,
		); err != nil {
			return nil, err
		}

		if lastSync.Valid {
			account.LastSync = lastSync.Time
		}
		if tokenExpires.Valid {
			account.TokenExpires = tokenExpires.Time
		}
		if errorMessage.Valid {
			account.ErrorMessage = errorMessage.String
		}

		accounts = append(accounts, account)
	}

	return accounts, nil
}

// GetAccountByEmail retrieves an account by email
func (r *AccountRepository) GetAccountByEmail(ctx context.Context, email string) (*types.StorageAccount, error) {
	query := `
		SELECT id, name, email, client_id, client_secret, tenant_id,
		       COALESCE(refresh_token, ''), COALESCE(access_token, ''), token_expires,
		       COALESCE(total_space, 0), COALESCE(used_space, 0), 
		       COALESCE(status, 'pending'), COALESCE(priority, 0),
		       last_sync, error_message, created_at, updated_at
		FROM storage_accounts
		WHERE email = $1
	`

	account := &types.StorageAccount{}
	var lastSync, tokenExpires sql.NullTime
	var errorMessage sql.NullString

	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&account.ID, &account.Name, &account.Email,
		&account.ClientID, &account.ClientSecret, &account.TenantID,
		&account.RefreshToken, &account.AccessToken, &tokenExpires,
		&account.TotalSpace, &account.UsedSpace,
		&account.Status, &account.Priority,
		&lastSync, &errorMessage,
		&account.CreatedAt, &account.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	if lastSync.Valid {
		account.LastSync = lastSync.Time
	}
	if tokenExpires.Valid {
		account.TokenExpires = tokenExpires.Time
	}
	if errorMessage.Valid {
		account.ErrorMessage = errorMessage.String
	}

	return account, nil
}
