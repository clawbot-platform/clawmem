package ops

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"clawmem/internal/domain/memory"
)

type MemoryService interface {
	ListAll(context.Context) ([]memory.MemoryRecord, error)
	UpdateRecord(context.Context, memory.MemoryRecord) (memory.MemoryRecord, error)
	Delete(context.Context, string) error
}

type Service struct {
	memory MemoryService
	now    func() time.Time

	mu   sync.RWMutex
	jobs map[memory.MaintenanceJobType]memory.MaintenanceJobStatus
}

const (
	replayTag         = "replay"
	replayLinkedTag   = "replay-linked"
	jobResultComplete = "completed"
	jobResultFailed   = "failed"
)

func NewService(memoryService MemoryService) *Service {
	return &Service{
		memory: memoryService,
		now:    func() time.Time { return time.Now().UTC() },
		jobs: map[memory.MaintenanceJobType]memory.MaintenanceJobStatus{
			memory.MaintenanceJobDecayUpdate:        {JobType: memory.MaintenanceJobDecayUpdate},
			memory.MaintenanceJobExpiredCleanup:     {JobType: memory.MaintenanceJobExpiredCleanup},
			memory.MaintenanceJobStaleCompaction:    {JobType: memory.MaintenanceJobStaleCompaction},
			memory.MaintenanceJobReplayPreservation: {JobType: memory.MaintenanceJobReplayPreservation},
		},
	}
}

func (s *Service) NamespaceSummaries(ctx context.Context, query memory.MemoryQuery) ([]memory.NamespaceSummary, error) {
	records, err := s.memory.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	query = memory.NormalizeQuery(query)
	summaries := map[string]*memory.NamespaceSummary{}
	now := s.now()

	for _, record := range records {
		if !matchesQuery(record, query) {
			continue
		}
		summary, ok := summaries[record.Namespace]
		if !ok {
			summary = &memory.NamespaceSummary{
				Namespace:     record.Namespace,
				ProjectID:     record.ProjectID,
				Environment:   record.Environment,
				ClawbotID:     record.ClawbotID,
				SessionID:     record.SessionID,
				RecordsByType: map[string]int{},
			}
			summaries[record.Namespace] = summary
		}
		applyRecordToNamespaceSummary(summary, record, now)
	}

	return sortNamespaceSummaries(summaries), nil
}

func (s *Service) ClawbotSummaries(ctx context.Context, query memory.MemoryQuery) ([]memory.ClawbotSummary, error) {
	records, err := s.memory.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	query = memory.NormalizeQuery(query)
	summaries := map[string]*memory.ClawbotSummary{}
	now := s.now()

	for _, record := range records {
		if !matchesQuery(record, query) {
			continue
		}
		key := strings.Join([]string{record.ProjectID, record.Environment, record.ClawbotID}, "/")
		summary, ok := summaries[key]
		if !ok {
			summary = &memory.ClawbotSummary{
				ProjectID:     record.ProjectID,
				Environment:   record.Environment,
				ClawbotID:     record.ClawbotID,
				RecordsByType: map[string]int{},
			}
			summaries[key] = summary
		}
		applyRecordToClawbotSummary(summary, record, now)
	}

	return sortClawbotSummaries(summaries), nil
}

func (s *Service) MaintenanceOverview(ctx context.Context) (memory.MaintenanceOverview, error) {
	records, err := s.memory.ListAll(ctx)
	if err != nil {
		return memory.MaintenanceOverview{}, err
	}

	now := s.now()
	var overview memory.MaintenanceOverview
	overview.LastUpdatedAt = now
	overview.DecayQueueCount = countDecayEligible(records, now)
	overview.ExpiredCount = countExpired(records, now)
	overview.ReplayPreservedCount = countReplayPreserved(records)
	overview.StaleSummaryCandidates = countStaleSummaryCandidates(records)

	s.mu.RLock()
	defer s.mu.RUnlock()
	overview.Jobs = make([]memory.MaintenanceJobStatus, 0, len(s.jobs))
	for _, job := range s.jobs {
		overview.Jobs = append(overview.Jobs, cloneJobStatus(job))
	}
	sort.Slice(overview.Jobs, func(i, j int) bool {
		return overview.Jobs[i].JobType < overview.Jobs[j].JobType
	})
	return overview, nil
}

func (s *Service) RunJob(ctx context.Context, jobType memory.MaintenanceJobType) (memory.MaintenanceJobStatus, error) {
	startedAt := s.now()
	records, err := s.memory.ListAll(ctx)
	if err != nil {
		return s.recordJobFailure(jobType, startedAt, err), err
	}

	var summary map[string]int
	switch jobType {
	case memory.MaintenanceJobDecayUpdate:
		summary, err = s.runDecayUpdate(ctx, records, startedAt)
	case memory.MaintenanceJobExpiredCleanup:
		summary, err = s.runExpiredCleanup(ctx, records, startedAt)
	case memory.MaintenanceJobStaleCompaction:
		summary, err = s.runStaleCompaction(ctx, records, startedAt)
	case memory.MaintenanceJobReplayPreservation:
		summary, err = s.runReplayPreservation(ctx, records, startedAt)
	default:
		err = fmt.Errorf("%w %q", errUnsupportedJob, jobType)
	}
	if err != nil {
		return s.recordJobFailure(jobType, startedAt, err), err
	}
	return s.recordJobSuccess(jobType, startedAt, summary), nil
}

func (s *Service) runDecayUpdate(ctx context.Context, records []memory.MemoryRecord, now time.Time) (map[string]int, error) {
	updated := 0
	for _, record := range records {
		if !memory.IsDecayEligible(record, now) || memory.IsExpired(record, now) {
			continue
		}
		decayed := memory.DecayRecord(record, now)
		if _, err := s.memory.UpdateRecord(ctx, decayed); err != nil {
			return nil, err
		}
		updated++
	}
	return map[string]int{"updated": updated}, nil
}

func (s *Service) runExpiredCleanup(ctx context.Context, records []memory.MemoryRecord, now time.Time) (map[string]int, error) {
	deleted := 0
	for _, record := range records {
		if !memory.IsExpired(record, now) {
			continue
		}
		if record.Pinned || record.ReplayLinked || record.RetentionPolicy == memory.RetentionPolicyPreserved || record.RetentionPolicy == memory.RetentionPolicyReplayPreserve {
			continue
		}
		if err := s.memory.Delete(ctx, record.ID); err != nil {
			return nil, err
		}
		deleted++
	}
	return map[string]int{"deleted": deleted}, nil
}

func (s *Service) runStaleCompaction(ctx context.Context, records []memory.MemoryRecord, _ time.Time) (map[string]int, error) {
	type groupKey struct {
		Namespace  string
		MemoryType memory.MemoryType
		SourceRef  string
	}
	grouped := make(map[groupKey][]memory.MemoryRecord)
	for _, record := range records {
		grouped[groupKey{Namespace: record.Namespace, MemoryType: record.MemoryType, SourceRef: record.SourceRef}] = append(grouped[groupKey{Namespace: record.Namespace, MemoryType: record.MemoryType, SourceRef: record.SourceRef}], record)
	}

	deleted := 0
	for _, group := range grouped {
		if len(group) < 2 {
			continue
		}
		sort.Slice(group, func(i, j int) bool {
			if group[i].UpdatedAt.Equal(group[j].UpdatedAt) {
				return group[i].ID < group[j].ID
			}
			return group[i].UpdatedAt.After(group[j].UpdatedAt)
		})
		for _, record := range group[1:] {
			if record.Pinned || record.ReplayLinked {
				continue
			}
			if err := s.memory.Delete(ctx, record.ID); err != nil {
				return nil, err
			}
			deleted++
		}
	}
	return map[string]int{"deleted": deleted}, nil
}

func (s *Service) runReplayPreservation(ctx context.Context, records []memory.MemoryRecord, now time.Time) (map[string]int, error) {
	updated := 0
	for _, record := range records {
		if !record.ReplayLinked && record.MemoryType != memory.MemoryTypeReplayCase && !isReplayTagged(record) {
			continue
		}
		changed := !record.ReplayLinked || record.RetentionPolicy != memory.RetentionPolicyReplayPreserve || record.StabilityScore < 70
		if !changed {
			continue
		}
		record.ReplayLinked = true
		record.RetentionPolicy = memory.RetentionPolicyReplayPreserve
		if record.StabilityScore < 70 {
			record.StabilityScore = 70
		}
		record.UpdatedAt = now
		if _, err := s.memory.UpdateRecord(ctx, record); err != nil {
			return nil, err
		}
		updated++
	}
	return map[string]int{"updated": updated}, nil
}

func (s *Service) recordJobSuccess(jobType memory.MaintenanceJobType, startedAt time.Time, summary map[string]int) memory.MaintenanceJobStatus {
	status := memory.MaintenanceJobStatus{
		JobType:        jobType,
		LastRunAt:      timePointer(startedAt),
		LastDurationMS: s.now().Sub(startedAt).Milliseconds(),
		LastResult:     jobResultComplete,
		LastSummary:    summary,
	}

	s.mu.Lock()
	s.jobs[jobType] = status
	s.mu.Unlock()
	return cloneJobStatus(status)
}

func (s *Service) recordJobFailure(jobType memory.MaintenanceJobType, startedAt time.Time, err error) memory.MaintenanceJobStatus {
	status := memory.MaintenanceJobStatus{
		JobType:        jobType,
		LastRunAt:      timePointer(startedAt),
		LastDurationMS: s.now().Sub(startedAt).Milliseconds(),
		LastResult:     jobResultFailed,
		LastError:      err.Error(),
	}

	s.mu.Lock()
	s.jobs[jobType] = status
	s.mu.Unlock()
	return cloneJobStatus(status)
}

func applyRecordToNamespaceSummary(summary *memory.NamespaceSummary, record memory.MemoryRecord, now time.Time) {
	summary.TotalRecords++
	summary.RecordsByType[string(record.MemoryType)]++
	summary.PinnedCount += boolInt(record.Pinned)
	summary.ReplayLinkedCount += boolInt(record.ReplayLinked || record.MemoryType == memory.MemoryTypeReplayCase)
	summary.DecayEligible += boolInt(memory.IsDecayEligible(record, now))
	summary.ApproximateBytes += approximateRecordBytes(record)
	summary.AverageStability += record.StabilityScore
	updateLastActivity(&summary.LastActivityAt, record)
}

func applyRecordToClawbotSummary(summary *memory.ClawbotSummary, record memory.MemoryRecord, now time.Time) {
	summary.TotalRecords++
	summary.RecordsByType[string(record.MemoryType)]++
	summary.PinnedCount += boolInt(record.Pinned)
	summary.ReplayLinkedCount += boolInt(record.ReplayLinked || record.MemoryType == memory.MemoryTypeReplayCase)
	summary.DecayEligible += boolInt(memory.IsDecayEligible(record, now))
	summary.ApproximateBytes += approximateRecordBytes(record)
	summary.AverageStability += record.StabilityScore
	updateLastActivity(&summary.LastActivityAt, record)
}

func sortNamespaceSummaries(source map[string]*memory.NamespaceSummary) []memory.NamespaceSummary {
	result := make([]memory.NamespaceSummary, 0, len(source))
	for _, summary := range source {
		if summary.TotalRecords > 0 {
			summary.AverageStability = summary.AverageStability / summary.TotalRecords
		}
		result = append(result, *summary)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Namespace < result[j].Namespace })
	return result
}

func sortClawbotSummaries(source map[string]*memory.ClawbotSummary) []memory.ClawbotSummary {
	result := make([]memory.ClawbotSummary, 0, len(source))
	for _, summary := range source {
		if summary.TotalRecords > 0 {
			summary.AverageStability = summary.AverageStability / summary.TotalRecords
		}
		result = append(result, *summary)
	}
	sort.Slice(result, func(i, j int) bool {
		left := strings.Join([]string{result[i].ProjectID, result[i].Environment, result[i].ClawbotID}, "/")
		right := strings.Join([]string{result[j].ProjectID, result[j].Environment, result[j].ClawbotID}, "/")
		return left < right
	})
	return result
}

func updateLastActivity(last **time.Time, record memory.MemoryRecord) {
	activity := record.UpdatedAt
	if record.LastAccessedAt != nil && record.LastAccessedAt.After(activity) {
		activity = *record.LastAccessedAt
	}
	if *last == nil || activity.After(**last) {
		copy := activity
		*last = &copy
	}
}

func approximateRecordBytes(record memory.MemoryRecord) int64 {
	payload, err := json.Marshal(record)
	if err != nil {
		return int64(len(record.Summary))
	}
	return int64(len(payload))
}

func matchesQuery(record memory.MemoryRecord, query memory.MemoryQuery) bool {
	if query.Namespace != "" && record.Namespace != query.Namespace {
		return false
	}
	if query.ProjectID != "" && record.ProjectID != query.ProjectID {
		return false
	}
	if query.Environment != "" && record.Environment != query.Environment {
		return false
	}
	if query.ClawbotID != "" && record.ClawbotID != query.ClawbotID {
		return false
	}
	if query.SessionID != "" && record.SessionID != query.SessionID {
		return false
	}
	if query.MemoryType != "" && record.MemoryType != query.MemoryType {
		return false
	}
	if query.ScenarioID != "" && record.ScenarioID != query.ScenarioID {
		return false
	}
	if query.SourceRef != "" && record.SourceRef != query.SourceRef {
		return false
	}
	return true
}

func countDecayEligible(records []memory.MemoryRecord, now time.Time) int {
	total := 0
	for _, record := range records {
		if memory.IsDecayEligible(record, now) {
			total++
		}
	}
	return total
}

func countExpired(records []memory.MemoryRecord, now time.Time) int {
	total := 0
	for _, record := range records {
		if memory.IsExpired(record, now) {
			total++
		}
	}
	return total
}

func countReplayPreserved(records []memory.MemoryRecord) int {
	total := 0
	for _, record := range records {
		if record.ReplayLinked || record.RetentionPolicy == memory.RetentionPolicyReplayPreserve || record.MemoryType == memory.MemoryTypeReplayCase {
			total++
		}
	}
	return total
}

func countStaleSummaryCandidates(records []memory.MemoryRecord) int {
	counts := make(map[string]int)
	for _, record := range records {
		key := strings.Join([]string{record.Namespace, string(record.MemoryType), record.SourceRef}, "|")
		counts[key]++
	}
	total := 0
	for _, count := range counts {
		if count > 1 {
			total += count - 1
		}
	}
	return total
}

func isReplayTagged(record memory.MemoryRecord) bool {
	for _, tag := range record.Tags {
		if tag == replayTag || tag == replayLinkedTag {
			return true
		}
	}
	return false
}

func cloneJobStatus(status memory.MaintenanceJobStatus) memory.MaintenanceJobStatus {
	if status.LastSummary == nil {
		status.LastSummary = map[string]int{}
	} else {
		copied := make(map[string]int, len(status.LastSummary))
		for key, value := range status.LastSummary {
			copied[key] = value
		}
		status.LastSummary = copied
	}
	if status.LastRunAt != nil {
		copy := *status.LastRunAt
		status.LastRunAt = &copy
	}
	return status
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func timePointer(value time.Time) *time.Time {
	return &value
}

var errUnsupportedJob = errors.New("unsupported maintenance job")

func ErrUnsupportedJob() error {
	return errUnsupportedJob
}
