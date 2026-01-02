package audit

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/xuecangming/onedrive-storage/internal/common/types"
	"github.com/xuecangming/onedrive-storage/internal/common/utils"
	"github.com/xuecangming/onedrive-storage/internal/infrastructure/onedrive"
	"github.com/xuecangming/onedrive-storage/internal/repository"
	"github.com/xuecangming/onedrive-storage/internal/service/account"
)

// Service handles audit operations
type Service struct {
	objectRepo     *repository.ObjectRepository
	accountService *account.Service
	currentReport  *types.AuditReport
	mu             sync.Mutex
}

// NewService creates a new audit service
func NewService(objectRepo *repository.ObjectRepository, accountService *account.Service) *Service {
	return &Service{
		objectRepo:     objectRepo,
		accountService: accountService,
	}
}

// StartAudit starts a new audit process
func (s *Service) StartAudit(ctx context.Context) (*types.AuditReport, error) {
	s.mu.Lock()
	if s.currentReport != nil && s.currentReport.Status == "running" {
		s.mu.Unlock()
		return nil, fmt.Errorf("audit already running")
	}

	report := &types.AuditReport{
		ID:        utils.GenerateID(),
		Status:    "running",
		StartTime: time.Now(),
		Issues:    make([]types.AuditIssue, 0),
	}
	s.currentReport = report
	s.mu.Unlock()

	go s.runAudit(context.Background(), report)

	return report, nil
}

// GetStatus returns the current or last audit report
func (s *Service) GetStatus() *types.AuditReport {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.currentReport
}

func (s *Service) runAudit(ctx context.Context, report *types.AuditReport) {
	log.Printf("Starting audit %s", report.ID)
	
	// Audit Objects
	offset := 0
	limit := 100
	for {
		objects, err := s.objectRepo.ListAllObjects(ctx, limit, offset)
		if err != nil {
			log.Printf("Error listing objects: %v", err)
			break
		}
		if len(objects) == 0 {
			break
		}

		for _, obj := range objects {
			report.TotalObjects++
			if !obj.IsChunked {
				s.checkObject(ctx, report, obj)
			}
			report.CheckedCount++
		}
		offset += limit
	}

	// Audit Chunks
	offset = 0
	for {
		chunks, err := s.objectRepo.ListAllChunks(ctx, limit, offset)
		if err != nil {
			log.Printf("Error listing chunks: %v", err)
			break
		}
		if len(chunks) == 0 {
			break
		}

		for _, chunk := range chunks {
			report.TotalChunks++
			s.checkChunk(ctx, report, chunk)
			report.CheckedCount++
		}
		offset += limit
	}

	endTime := time.Now()
	report.EndTime = &endTime
	report.Status = "completed"
	report.Summary = fmt.Sprintf("Checked %d objects and %d chunks. Found %d issues.", report.TotalObjects, report.TotalChunks, len(report.Issues))
	
	log.Printf("Audit %s completed: %s", report.ID, report.Summary)
}

func (s *Service) checkObject(ctx context.Context, report *types.AuditReport, obj *types.Object) {
	if obj.AccountID == "00000000-0000-0000-0000-000000000000" {
		// Local storage, skip for now or implement local check
		return
	}

	err := s.checkRemoteFile(ctx, obj.AccountID, obj.RemoteID)
	if err != nil {
		s.addIssue(report, types.AuditIssue{
			Type:        "missing_file",
			Bucket:      obj.Bucket,
			Key:         obj.Key,
			AccountID:   obj.AccountID,
			RemoteID:    obj.RemoteID,
			Description: fmt.Sprintf("Object missing or inaccessible: %v", err),
		})
	}
}

func (s *Service) checkChunk(ctx context.Context, report *types.AuditReport, chunk *types.ObjectChunk) {
	err := s.checkRemoteFile(ctx, chunk.AccountID, chunk.RemoteID)
	if err != nil {
		s.addIssue(report, types.AuditIssue{
			Type:        "missing_chunk",
			Bucket:      chunk.Bucket,
			Key:         chunk.Key,
			ChunkIndex:  &chunk.ChunkIndex,
			AccountID:   chunk.AccountID,
			RemoteID:    chunk.RemoteID,
			Description: fmt.Sprintf("Chunk missing or inaccessible: %v", err),
		})
	}
}

func (s *Service) checkRemoteFile(ctx context.Context, accountID, remoteID string) error {
	// Get account
	account, err := s.accountService.Get(ctx, accountID)
	if err != nil {
		return fmt.Errorf("account not found: %v", err)
	}

	// Ensure token is valid
	if err := s.accountService.EnsureTokenValid(ctx, account.ID); err != nil {
		return fmt.Errorf("token invalid: %v", err)
	}
	
	// Get fresh account
	account, _ = s.accountService.Get(ctx, account.ID)

	client := onedrive.NewClient(account.AccessToken)
	
	// We need a lightweight way to check existence. 
	// GetDriveItem is not implemented in the client yet, but DownloadFile is.
	// Ideally we should add GetMetadata to onedrive client.
	// For now, let's assume if we can't get it, it's an issue.
	// But downloading the whole file is too heavy.
	// Let's add GetItemMetadata to onedrive client first.
	
	// Since I cannot modify onedrive client in this step easily without context switch,
	// I will assume I can use a method that I will add next.
	_, err = client.GetItem(ctx, remoteID)
	return err
}

func (s *Service) addIssue(report *types.AuditReport, issue types.AuditIssue) {
	s.mu.Lock()
	defer s.mu.Unlock()
	report.Issues = append(report.Issues, issue)
}
