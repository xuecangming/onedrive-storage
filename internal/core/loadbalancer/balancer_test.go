package loadbalancer

import (
	"context"
	"testing"

	"github.com/xuecangming/onedrive-storage/internal/common/types"
)

func createTestAccounts() []*types.StorageAccount {
	return []*types.StorageAccount{
		{
			ID:         "account-1",
			Name:       "Account 1",
			Status:     "active",
			TotalSpace: 1000,
			UsedSpace:  200,
			Priority:   10,
		},
		{
			ID:         "account-2",
			Name:       "Account 2",
			Status:     "active",
			TotalSpace: 1000,
			UsedSpace:  500,
			Priority:   20,
		},
		{
			ID:         "account-3",
			Name:       "Account 3",
			Status:     "active",
			TotalSpace: 1000,
			UsedSpace:  100,
			Priority:   5,
		},
	}
}

func TestNewBalancer(t *testing.T) {
	balancer := NewBalancer(StrategyLeastUsed)

	if balancer == nil {
		t.Error("NewBalancer returned nil")
	}
	if balancer.strategy != StrategyLeastUsed {
		t.Errorf("strategy = %v, want %v", balancer.strategy, StrategyLeastUsed)
	}
}

func TestSelectAccount_NoAccounts(t *testing.T) {
	balancer := NewBalancer(StrategyLeastUsed)
	ctx := context.Background()

	_, err := balancer.SelectAccount(ctx, []*types.StorageAccount{}, 100)

	if err == nil {
		t.Error("expected error when no accounts available")
	}
}

func TestSelectAccount_NoSpaceAvailable(t *testing.T) {
	balancer := NewBalancer(StrategyLeastUsed)
	ctx := context.Background()

	accounts := []*types.StorageAccount{
		{
			ID:         "account-1",
			Status:     "active",
			TotalSpace: 1000,
			UsedSpace:  999,
		},
	}

	_, err := balancer.SelectAccount(ctx, accounts, 100)

	if err == nil {
		t.Error("expected error when no space available")
	}
}

func TestSelectAccount_LeastUsed(t *testing.T) {
	balancer := NewBalancer(StrategyLeastUsed)
	ctx := context.Background()
	accounts := createTestAccounts()

	selected, err := balancer.SelectAccount(ctx, accounts, 100)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Account 3 has least usage (10%)
	if selected.ID != "account-3" {
		t.Errorf("selected = %v, want account-3 (least used)", selected.ID)
	}
}

func TestSelectAccount_RoundRobin(t *testing.T) {
	balancer := NewBalancer(StrategyRoundRobin)
	ctx := context.Background()
	accounts := createTestAccounts()

	// First round - should get accounts in order
	for i := 0; i < 3; i++ {
		selected, err := balancer.SelectAccount(ctx, accounts, 100)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expectedID := accounts[i].ID
		if selected.ID != expectedID {
			t.Errorf("round %d: selected = %v, want %v", i, selected.ID, expectedID)
		}
	}

	// Should wrap around
	selected, err := balancer.SelectAccount(ctx, accounts, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if selected.ID != "account-1" {
		t.Errorf("wrap around: selected = %v, want account-1", selected.ID)
	}
}

func TestSelectAccount_Weighted(t *testing.T) {
	balancer := NewBalancer(StrategyWeighted)
	ctx := context.Background()
	accounts := createTestAccounts()

	// Run multiple times and check distribution
	counts := make(map[string]int)
	iterations := 1000

	for i := 0; i < iterations; i++ {
		selected, err := balancer.SelectAccount(ctx, accounts, 100)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		counts[selected.ID]++
	}

	// Account 2 has highest priority (20), should be selected most often
	if counts["account-2"] < counts["account-1"] || counts["account-2"] < counts["account-3"] {
		t.Errorf("account-2 with highest priority should be selected most often, got: %v", counts)
	}
}

func TestSelectAccount_InactiveAccountsFiltered(t *testing.T) {
	balancer := NewBalancer(StrategyLeastUsed)
	ctx := context.Background()

	accounts := []*types.StorageAccount{
		{
			ID:         "account-1",
			Status:     "inactive",
			TotalSpace: 1000,
			UsedSpace:  0,
			Priority:   10,
		},
		{
			ID:         "account-2",
			Status:     "active",
			TotalSpace: 1000,
			UsedSpace:  500,
			Priority:   10,
		},
	}

	selected, err := balancer.SelectAccount(ctx, accounts, 100)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if selected.ID != "account-2" {
		t.Errorf("selected = %v, want account-2 (only active account)", selected.ID)
	}
}

func TestSelectAccount_UnsyncedAccountAccepted(t *testing.T) {
	balancer := NewBalancer(StrategyLeastUsed)
	ctx := context.Background()

	accounts := []*types.StorageAccount{
		{
			ID:         "account-1",
			Status:     "active",
			TotalSpace: 0, // Not synced yet
			UsedSpace:  0,
		},
	}

	selected, err := balancer.SelectAccount(ctx, accounts, 100)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if selected.ID != "account-1" {
		t.Errorf("selected = %v, want account-1", selected.ID)
	}
}

func TestGetUsageStats_Empty(t *testing.T) {
	balancer := NewBalancer(StrategyLeastUsed)

	stats := balancer.GetUsageStats([]*types.StorageAccount{})

	if stats["total_accounts"] != 0 {
		t.Errorf("total_accounts = %v, want 0", stats["total_accounts"])
	}
}

func TestGetUsageStats_WithAccounts(t *testing.T) {
	balancer := NewBalancer(StrategyLeastUsed)
	accounts := createTestAccounts()

	stats := balancer.GetUsageStats(accounts)

	if stats["total_accounts"] != 3 {
		t.Errorf("total_accounts = %v, want 3", stats["total_accounts"])
	}
	if stats["active_accounts"] != 3 {
		t.Errorf("active_accounts = %v, want 3", stats["active_accounts"])
	}
	if stats["total_space"] != int64(3000) {
		t.Errorf("total_space = %v, want 3000", stats["total_space"])
	}
	if stats["used_space"] != int64(800) {
		t.Errorf("used_space = %v, want 800", stats["used_space"])
	}
	if stats["available_space"] != int64(2200) {
		t.Errorf("available_space = %v, want 2200", stats["available_space"])
	}
}

func TestFilterAvailableAccounts(t *testing.T) {
	balancer := NewBalancer(StrategyLeastUsed)

	accounts := []*types.StorageAccount{
		{ID: "1", Status: "active", TotalSpace: 1000, UsedSpace: 100},
		{ID: "2", Status: "inactive", TotalSpace: 1000, UsedSpace: 0},
		{ID: "3", Status: "active", TotalSpace: 1000, UsedSpace: 950},
		{ID: "4", Status: "active", TotalSpace: 0, UsedSpace: 0}, // unsynced
	}

	filtered := balancer.filterAvailableAccounts(accounts, 100)

	if len(filtered) != 2 {
		t.Errorf("len(filtered) = %v, want 2", len(filtered))
	}

	// Should include account-1 (has space) and account-4 (unsynced)
	ids := make(map[string]bool)
	for _, a := range filtered {
		ids[a.ID] = true
	}
	if !ids["1"] {
		t.Error("account-1 should be included (has enough space)")
	}
	if !ids["4"] {
		t.Error("account-4 should be included (unsynced, assumed unlimited)")
	}
}

func TestSelectLeastUsed_SingleAccount(t *testing.T) {
	balancer := NewBalancer(StrategyLeastUsed)
	accounts := []*types.StorageAccount{
		{ID: "only", Status: "active", TotalSpace: 1000, UsedSpace: 500},
	}

	selected := balancer.selectLeastUsed(accounts)

	if selected.ID != "only" {
		t.Errorf("selected = %v, want 'only'", selected.ID)
	}
}

func TestSelectRoundRobin_ThreadSafety(t *testing.T) {
	balancer := NewBalancer(StrategyRoundRobin)
	accounts := createTestAccounts()
	ctx := context.Background()

	// Run concurrent selections
	done := make(chan bool)
	errChan := make(chan error, 1000)

	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				_, err := balancer.SelectAccount(ctx, accounts, 0)
				if err != nil {
					select {
					case errChan <- err:
					default:
					}
				}
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Check for any errors
	select {
	case err := <-errChan:
		t.Errorf("unexpected error during concurrent access: %v", err)
	default:
		// No errors
	}
}
