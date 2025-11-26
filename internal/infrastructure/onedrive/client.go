package onedrive

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client represents a OneDrive API client
type Client struct {
	httpClient  *http.Client
	accessToken string
	baseURL     string
}

// NewClient creates a new OneDrive client
func NewClient(accessToken string) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		accessToken: accessToken,
		baseURL:     "https://graph.microsoft.com/v1.0",
	}
}

// DriveItem represents a OneDrive item (file or folder)
type DriveItem struct {
	ID               string                 `json:"id"`
	Name             string                 `json:"name"`
	Size             int64                  `json:"size"`
	CreatedDateTime  time.Time              `json:"createdDateTime"`
	ModifiedDateTime time.Time              `json:"modifiedDateTime"`
	File             *FileMetadata          `json:"file,omitempty"`
	Folder           *FolderMetadata        `json:"folder,omitempty"`
	CTag             string                 `json:"cTag"`
	ETag             string                 `json:"eTag"`
	AdditionalData   map[string]interface{} `json:"@microsoft.graph.downloadUrl,omitempty"`
}

// FileMetadata represents file-specific metadata
type FileMetadata struct {
	MimeType string     `json:"mimeType"`
	Hashes   FileHashes `json:"hashes"`
}

// FolderMetadata represents folder-specific metadata
type FolderMetadata struct {
	ChildCount int `json:"childCount"`
}

// FileHashes represents file hash values
type FileHashes struct {
	SHA1Hash     string `json:"sha1Hash"`
	QuickXorHash string `json:"quickXorHash"`
}

// DriveQuota represents drive quota information
type DriveQuota struct {
	Total     int64  `json:"total"`
	Used      int64  `json:"used"`
	Remaining int64  `json:"remaining"`
	Deleted   int64  `json:"deleted"`
	State     string `json:"state"`
}

// Drive represents a OneDrive drive
type Drive struct {
	ID        string     `json:"id"`
	DriveType string     `json:"driveType"`
	Owner     DriveOwner `json:"owner"`
	Quota     DriveQuota `json:"quota"`
}

// DriveOwner represents drive owner information
type DriveOwner struct {
	User DriveUser `json:"user"`
}

// DriveUser represents user information
type DriveUser struct {
	DisplayName string `json:"displayName"`
	Email       string `json:"email"`
}

// UploadSession represents an upload session for large files
type UploadSession struct {
	UploadURL          string    `json:"uploadUrl"`
	ExpirationDateTime time.Time `json:"expirationDateTime"`
}

// GetDrive retrieves drive information
func (c *Client) GetDrive(ctx context.Context) (*Drive, error) {
	url := fmt.Sprintf("%s/me/drive", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %s (status: %d)", string(body), resp.StatusCode)
	}

	var drive Drive
	if err := json.NewDecoder(resp.Body).Decode(&drive); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &drive, nil
}

// UploadSmallFile uploads a file smaller than 4MB
func (c *Client) UploadSmallFile(ctx context.Context, path string, data []byte) (*DriveItem, error) {
	url := fmt.Sprintf("%s/me/drive/root:/%s:/content", c.baseURL, path)

	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %s (status: %d)", string(body), resp.StatusCode)
	}

	var item DriveItem
	if err := json.NewDecoder(resp.Body).Decode(&item); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &item, nil
}

// DownloadFile downloads a file from OneDrive
func (c *Client) DownloadFile(ctx context.Context, itemID string) ([]byte, error) {
	// First get the download URL
	url := fmt.Sprintf("%s/me/drive/items/%s", c.baseURL, itemID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %s (status: %d)", string(body), resp.StatusCode)
	}

	var item map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&item); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	downloadURL, ok := item["@microsoft.graph.downloadUrl"].(string)
	if !ok {
		return nil, fmt.Errorf("download URL not found in response")
	}

	// Download the file content
	req2, err := http.NewRequestWithContext(ctx, "GET", downloadURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create download request: %w", err)
	}

	resp2, err := c.httpClient.Do(req2)
	if err != nil {
		return nil, fmt.Errorf("failed to execute download request: %w", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed with status: %d", resp2.StatusCode)
	}

	return io.ReadAll(resp2.Body)
}

// DeleteFile deletes a file from OneDrive
func (c *Client) DeleteFile(ctx context.Context, itemID string) error {
	url := fmt.Sprintf("%s/me/drive/items/%s", c.baseURL, itemID)

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: %s (status: %d)", string(body), resp.StatusCode)
	}

	return nil
}

// CreateUploadSession creates an upload session for large files
func (c *Client) CreateUploadSession(ctx context.Context, path string) (*UploadSession, error) {
	url := fmt.Sprintf("%s/me/drive/root:/%s:/createUploadSession", c.baseURL, path)

	body := map[string]interface{}{
		"item": map[string]interface{}{
			"@microsoft.graph.conflictBehavior": "replace",
		},
	}

	bodyJSON, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %s (status: %d)", string(respBody), resp.StatusCode)
	}

	var session UploadSession
	if err := json.NewDecoder(resp.Body).Decode(&session); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &session, nil
}

// UploadChunk uploads a chunk to an upload session
func (c *Client) UploadChunk(ctx context.Context, uploadURL string, chunk []byte, rangeStart, rangeEnd, totalSize int64) error {
	req, err := http.NewRequestWithContext(ctx, "PUT", uploadURL, bytes.NewReader(chunk))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Length", fmt.Sprintf("%d", len(chunk)))
	req.Header.Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", rangeStart, rangeEnd, totalSize))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Accept both 200 (intermediate) and 201/202 (complete)
	if resp.StatusCode != http.StatusOK &&
		resp.StatusCode != http.StatusCreated &&
		resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("chunk upload failed: %s (status: %d)", string(body), resp.StatusCode)
	}

	return nil
}
