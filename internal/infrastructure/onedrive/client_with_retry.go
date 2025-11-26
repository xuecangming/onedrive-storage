package onedrive

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/xuecangming/onedrive-storage/internal/core/logger"
	"github.com/xuecangming/onedrive-storage/internal/core/retry"
)

// ClientWithRetry wraps OneDrive client with retry logic
type ClientWithRetry struct {
	client       *Client
	retryConfig  *retry.Config
	logger       logger.Logger
}

// NewClientWithRetry creates a new OneDrive client with retry support
func NewClientWithRetry(accessToken string, retryConfig *retry.Config, log logger.Logger) *ClientWithRetry {
	if retryConfig == nil {
		retryConfig = retry.DefaultConfig()
	}
	if log == nil {
		log = logger.GetGlobalLogger()
	}

	return &ClientWithRetry{
		client:      NewClient(accessToken),
		retryConfig: retryConfig,
		logger:      log,
	}
}

// isRetryableError determines if an error should be retried
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// Retry on network errors
	if strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "connection") ||
		strings.Contains(errStr, "EOF") {
		return true
	}

	// Retry on specific HTTP status codes
	if strings.Contains(errStr, "429") || // Too Many Requests
		strings.Contains(errStr, "500") || // Internal Server Error
		strings.Contains(errStr, "502") || // Bad Gateway
		strings.Contains(errStr, "503") || // Service Unavailable
		strings.Contains(errStr, "504") { // Gateway Timeout
		return true
	}

	return false
}

// UploadSmallFile uploads a small file with retry
func (c *ClientWithRetry) UploadSmallFile(ctx context.Context, path string, data []byte) (*DriveItem, error) {
	var item *DriveItem
	var err error

	c.logger.Info("Uploading small file",
		logger.String("path", path),
		logger.Int("size", len(data)))

	err = retry.DoWithContextAndRetryable(ctx, func(ctx context.Context) error {
		item, err = c.client.UploadSmallFile(ctx, path, data)
		if err != nil {
			c.logger.Warn("Upload attempt failed",
				logger.String("path", path),
				logger.Error(err))
		}
		return err
	}, c.retryConfig, isRetryableError)

	if err != nil {
		c.logger.Error("Upload failed after retries",
			logger.String("path", path),
			logger.Error(err))
		return nil, err
	}

	c.logger.Info("Upload successful",
		logger.String("path", path),
		logger.String("item_id", item.ID))

	return item, nil
}

// CreateUploadSession creates an upload session with retry
func (c *ClientWithRetry) CreateUploadSession(ctx context.Context, path string) (*UploadSession, error) {
	var session *UploadSession
	var err error

	c.logger.Info("Creating upload session", logger.String("path", path))

	err = retry.DoWithContextAndRetryable(ctx, func(ctx context.Context) error {
		session, err = c.client.CreateUploadSession(ctx, path)
		if err != nil {
			c.logger.Warn("Create session attempt failed",
				logger.String("path", path),
				logger.Error(err))
		}
		return err
	}, c.retryConfig, isRetryableError)

	if err != nil {
		c.logger.Error("Create session failed after retries",
			logger.String("path", path),
			logger.Error(err))
		return nil, err
	}

	c.logger.Info("Upload session created",
		logger.String("path", path),
		logger.String("upload_url", session.UploadURL))

	return session, nil
}

// UploadChunk uploads a file chunk with retry
func (c *ClientWithRetry) UploadChunk(ctx context.Context, uploadURL string, data []byte, start, end, total int64) error {
	var err error

	c.logger.Debug("Uploading chunk",
		logger.Int64("start", start),
		logger.Int64("end", end),
		logger.Int64("total", total))

	err = retry.DoWithContextAndRetryable(ctx, func(ctx context.Context) error {
		err = c.client.UploadChunk(ctx, uploadURL, data, start, end, total)
		if err != nil {
			c.logger.Warn("Chunk upload attempt failed",
				logger.Int64("start", start),
				logger.Int64("end", end),
				logger.Error(err))
		}
		return err
	}, c.retryConfig, isRetryableError)

	if err != nil {
		c.logger.Error("Chunk upload failed after retries",
			logger.Int64("start", start),
			logger.Int64("end", end),
			logger.Error(err))
		return err
	}

	c.logger.Info("Chunk upload completed",
		logger.Int64("start", start),
		logger.Int64("end", end))

	return nil
}

// DownloadFile downloads a file with retry
func (c *ClientWithRetry) DownloadFile(ctx context.Context, itemID string) ([]byte, error) {
	var data []byte
	var err error

	c.logger.Info("Downloading file", logger.String("item_id", itemID))

	err = retry.DoWithContextAndRetryable(ctx, func(ctx context.Context) error {
		data, err = c.client.DownloadFile(ctx, itemID)
		if err != nil {
			c.logger.Warn("Download attempt failed",
				logger.String("item_id", itemID),
				logger.Error(err))
		}
		return err
	}, c.retryConfig, isRetryableError)

	if err != nil {
		c.logger.Error("Download failed after retries",
			logger.String("item_id", itemID),
			logger.Error(err))
		return nil, err
	}

	c.logger.Info("Download successful",
		logger.String("item_id", itemID),
		logger.Int("size", len(data)))

	return data, nil
}

// DeleteFile deletes a file with retry
func (c *ClientWithRetry) DeleteFile(ctx context.Context, itemID string) error {
	var err error

	c.logger.Info("Deleting file", logger.String("item_id", itemID))

	err = retry.DoWithContextAndRetryable(ctx, func(ctx context.Context) error {
		err = c.client.DeleteFile(ctx, itemID)
		if err != nil {
			c.logger.Warn("Delete attempt failed",
				logger.String("item_id", itemID),
				logger.Error(err))
		}
		return err
	}, c.retryConfig, isRetryableError)

	if err != nil {
		c.logger.Error("Delete failed after retries",
			logger.String("item_id", itemID),
			logger.Error(err))
		return err
	}

	c.logger.Info("Delete successful", logger.String("item_id", itemID))

	return nil
}

// GetDrive gets drive information with retry
func (c *ClientWithRetry) GetDrive(ctx context.Context) (*Drive, error) {
	var drive *Drive
	var err error

	c.logger.Debug("Getting drive info")

	err = retry.DoWithContextAndRetryable(ctx, func(ctx context.Context) error {
		drive, err = c.client.GetDrive(ctx)
		if err != nil {
			c.logger.Warn("Get drive info attempt failed", logger.Error(err))
		}
		return err
	}, c.retryConfig, isRetryableError)

	if err != nil {
		c.logger.Error("Get drive info failed after retries", logger.Error(err))
		return nil, err
	}

	c.logger.Debug("Drive info retrieved",
		logger.Int64("total", drive.Quota.Total),
		logger.Int64("used", drive.Quota.Used))

	return drive, nil
}

// isHTTPRetryable checks if an HTTP status code is retryable
func isHTTPRetryable(statusCode int) bool {
	return statusCode == http.StatusTooManyRequests || // 429
		statusCode == http.StatusInternalServerError || // 500
		statusCode == http.StatusBadGateway || // 502
		statusCode == http.StatusServiceUnavailable || // 503
		statusCode == http.StatusGatewayTimeout // 504
}

// UpdateAccessToken updates the access token
func (c *ClientWithRetry) UpdateAccessToken(token string) {
	c.client.accessToken = token
	c.logger.Debug("Access token updated")
}

// GetClient returns the underlying OneDrive client
func (c *ClientWithRetry) GetClient() *Client {
	return c.client
}

// SetRetryConfig updates the retry configuration
func (c *ClientWithRetry) SetRetryConfig(config *retry.Config) {
	c.retryConfig = config
	c.logger.Debug("Retry config updated",
		logger.Int("max_attempts", config.MaxAttempts))
}

// HealthCheck performs a health check on the OneDrive connection
func (c *ClientWithRetry) HealthCheck(ctx context.Context) error {
	c.logger.Debug("Performing health check")

	_, err := c.GetDrive(ctx)
	if err != nil {
		c.logger.Error("Health check failed", logger.Error(err))
		return fmt.Errorf("OneDrive health check failed: %w", err)
	}

	c.logger.Debug("Health check passed")
	return nil
}
