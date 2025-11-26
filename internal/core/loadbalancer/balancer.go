package loadbalancer

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/xuecangming/onedrive-storage/internal/common/types"
)

// Strategy represents load balancing strategy
type Strategy string

const (
	// StrategyLeastUsed selects account with lowest usage
	StrategyLeastUsed Strategy = "least_used"
	// StrategyRoundRobin cycles through accounts
	StrategyRoundRobin Strategy = "round_robin"
	// StrategyWeighted uses priority-based weighted random selection
	StrategyWeighted Strategy = "weighted"
)

// Balancer handles account selection for load balancing
type Balancer struct {
	strategy      Strategy
	currentIndex  int
	mu            sync.Mutex
	rand          *rand.Rand
}

// NewBalancer creates a new load balancer
func NewBalancer(strategy Strategy) *Balancer {
	return &Balancer{
		strategy: strategy,
		rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// SelectAccount selects an account based on the load balancing strategy
func (b *Balancer) SelectAccount(ctx context.Context, accounts []*types.StorageAccount, requiredSpace int64) (*types.StorageAccount, error) {
	if len(accounts) == 0 {
		return nil, fmt.Errorf("no accounts available")
	}

	// Filter accounts with enough space
	availableAccounts := b.filterAvailableAccounts(accounts, requiredSpace)
	if len(availableAccounts) == 0 {
		return nil, fmt.Errorf("no accounts with enough space available")
	}

	switch b.strategy {
	case StrategyLeastUsed:
		return b.selectLeastUsed(availableAccounts), nil
	case StrategyRoundRobin:
		return b.selectRoundRobin(availableAccounts), nil
	case StrategyWeighted:
		return b.selectWeighted(availableAccounts), nil
	default:
		return b.selectLeastUsed(availableAccounts), nil
	}
}

// filterAvailableAccounts filters accounts that have enough space
func (b *Balancer) filterAvailableAccounts(accounts []*types.StorageAccount, requiredSpace int64) []*types.StorageAccount {
	var available []*types.StorageAccount
	for _, account := range accounts {
		if account.Status == "active" {
			availableSpace := account.TotalSpace - account.UsedSpace
			if availableSpace >= requiredSpace {
				available = append(available, account)
			}
		}
	}
	return available
}

// selectLeastUsed selects account with lowest usage percentage
func (b *Balancer) selectLeastUsed(accounts []*types.StorageAccount) *types.StorageAccount {
	if len(accounts) == 0 {
		return nil
	}

	selected := accounts[0]
	minUsage := float64(selected.UsedSpace) / float64(selected.TotalSpace)

	for _, account := range accounts[1:] {
		usage := float64(account.UsedSpace) / float64(account.TotalSpace)
		if usage < minUsage {
			minUsage = usage
			selected = account
		}
	}

	return selected
}

// selectRoundRobin selects next account in rotation
func (b *Balancer) selectRoundRobin(accounts []*types.StorageAccount) *types.StorageAccount {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(accounts) == 0 {
		return nil
	}

	selected := accounts[b.currentIndex%len(accounts)]
	b.currentIndex++

	return selected
}

// selectWeighted selects account based on priority weights
func (b *Balancer) selectWeighted(accounts []*types.StorageAccount) *types.StorageAccount {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(accounts) == 0 {
		return nil
	}

	// Calculate total weight
	totalWeight := 0
	for _, account := range accounts {
		totalWeight += account.Priority
	}

	if totalWeight == 0 {
		// If all priorities are 0, use random selection
		return accounts[b.rand.Intn(len(accounts))]
	}

	// Weighted random selection
	randomValue := b.rand.Intn(totalWeight)
	currentWeight := 0

	for _, account := range accounts {
		currentWeight += account.Priority
		if randomValue < currentWeight {
			return account
		}
	}

	// Fallback (should not reach here)
	return accounts[0]
}

// GetUsageStats returns usage statistics for accounts
func (b *Balancer) GetUsageStats(accounts []*types.StorageAccount) map[string]interface{} {
	if len(accounts) == 0 {
		return map[string]interface{}{
			"total_accounts": 0,
			"active_accounts": 0,
			"total_space": 0,
			"used_space": 0,
			"available_space": 0,
		}
	}

	var totalSpace, usedSpace int64
	activeCount := 0

	for _, account := range accounts {
		totalSpace += account.TotalSpace
		usedSpace += account.UsedSpace
		if account.Status == "active" {
			activeCount++
		}
	}

	return map[string]interface{}{
		"total_accounts":   len(accounts),
		"active_accounts":  activeCount,
		"total_space":      totalSpace,
		"used_space":       usedSpace,
		"available_space":  totalSpace - usedSpace,
		"usage_percent":    float64(usedSpace) / float64(totalSpace) * 100,
	}
}
