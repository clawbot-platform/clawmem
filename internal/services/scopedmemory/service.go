package scopedmemory

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	domain "clawmem/internal/domain/scopedmemory"
	storepkg "clawmem/internal/platform/store"
)

type Store interface {
	CreateScopedRecord(context.Context, domain.Record) (domain.Record, error)
	UpdateScopedRecord(context.Context, domain.Record) (domain.Record, error)
	GetScopedRecord(context.Context, string) (domain.Record, error)
	ListScopedRecords(context.Context, domain.Query) (domain.QueryResult, error)

	CreateScopedSnapshot(context.Context, domain.Snapshot) (domain.Snapshot, error)
	GetScopedSnapshot(context.Context, string) (domain.Snapshot, error)
	ListScopedSnapshots(context.Context, domain.SnapshotQuery) (domain.SnapshotQueryResult, error)
}

type Service struct {
	store         Store
	now           func() time.Time
	recordIDGen   func() string
	snapshotIDGen func() string
}

type PersistNotesInput struct {
	Note                       string         `json:"note"`
	PriorCycleSummaries        []string       `json:"prior_cycle_summaries"`
	CarryForwardRisks          []string       `json:"carry_forward_risks"`
	UnresolvedGaps             []string       `json:"unresolved_gaps"`
	BacklogItems               []string       `json:"backlog_items"`
	ReviewerNotes              []string       `json:"reviewer_notes"`
	PolicyExceptions           []string       `json:"policy_exceptions"`
	WorkingContext             []string       `json:"working_context"`
	CycleSummaries             []string       `json:"cycle_summaries"`
	ResolveUnresolvedGaps      []string       `json:"resolve_unresolved_gaps"`
	ResolvedGapIDs             []string       `json:"resolved_gap_ids"`
	ResolvedRiskIDs            []string       `json:"resolved_risk_ids"`
	ResolvedBacklogItemIDs     []string       `json:"resolved_backlog_item_ids"`
	ResolvedReviewerNoteIDs    []string       `json:"resolved_reviewer_note_ids"`
	ResolvedPolicyExceptionIDs []string       `json:"resolved_policy_exception_ids"`
	CreatedBy                  string         `json:"created_by"`
	Status                     domain.Status  `json:"status"`
	ContentJSON                map[string]any `json:"content_json"`
	MetadataJSON               map[string]any `json:"metadata_json"`
	Provenance                 map[string]any `json:"provenance"`
	SnapshotSummary            string         `json:"snapshot_summary"`
	SnapshotManifestRef        string         `json:"snapshot_manifest_ref"`
	ExpiresAt                  *time.Time     `json:"expires_at"`
	SourceRunID                string         `json:"source_run_id"`
	SourceCycleID              string         `json:"source_cycle_id"`
	SourceArtifactID           string         `json:"source_artifact_id"`
	SourcePolicyDecisionID     string         `json:"source_policy_decision_id"`
	SourceModelProfileID       string         `json:"source_model_profile_id"`
}

type PersistNotesResult struct {
	SnapshotRef       string          `json:"snapshot_ref"`
	Snapshot          domain.Snapshot `json:"snapshot"`
	RecordsWritten    int             `json:"records_written"`
	RecordIDs         []string        `json:"record_ids"`
	ResolvedRecordIDs []string        `json:"resolved_record_ids"`
}

type CreateSnapshotInput struct {
	Namespace     domain.Namespace `json:"namespace"`
	CreatedBy     string           `json:"created_by"`
	Summary       string           `json:"summary"`
	RecordRefs    []string         `json:"record_refs"`
	QueryCriteria *domain.Query    `json:"query_criteria,omitempty"`
	ManifestRef   string           `json:"manifest_ref,omitempty"`
	MetadataJSON  map[string]any   `json:"metadata_json,omitempty"`
}

type UpdateRecordStatusInput struct {
	Status    domain.Status `json:"status"`
	UpdatedBy string        `json:"updated_by"`
	Reason    string        `json:"reason"`
}

type recordProvenance struct {
	SourceRunID            string
	SourceCycleID          string
	SourceArtifactID       string
	SourcePolicyDecisionID string
	SourceModelProfileID   string
}

const (
	defaultContextMaxPerClass = 5
	defaultPriorSummaryMax    = 7
)

func NewService(store Store) *Service {
	return &Service{
		store: store,
		now:   func() time.Time { return time.Now().UTC() },
		recordIDGen: func() string {
			return fmt.Sprintf("smr-%d", time.Now().UTC().UnixNano())
		},
		snapshotIDGen: func() string {
			return fmt.Sprintf("sms-%d", time.Now().UTC().UnixNano())
		},
	}
}

func (s *Service) ListRecords(ctx context.Context, query domain.Query) (domain.QueryResult, error) {
	return s.store.ListScopedRecords(ctx, domain.NormalizeQuery(query))
}

func (s *Service) GetRecord(ctx context.Context, recordID string) (domain.Record, error) {
	return s.store.GetScopedRecord(ctx, strings.TrimSpace(recordID))
}

func (s *Service) UpdateRecordStatus(ctx context.Context, recordID string, input UpdateRecordStatusInput) (domain.Record, error) {
	recordID = strings.TrimSpace(recordID)
	if recordID == "" {
		return domain.Record{}, errors.New("record id is required")
	}

	record, err := s.store.GetScopedRecord(ctx, recordID)
	if err != nil {
		return domain.Record{}, err
	}
	if !isActionableMemoryClass(record.MemoryClass) {
		return domain.Record{}, fmt.Errorf("status transitions are only supported for actionable classes")
	}

	target := domain.NormalizeStatus(input.Status)
	if target == "" {
		return domain.Record{}, errors.New("status is required")
	}
	if err := domain.ValidateStatus(target); err != nil {
		return domain.Record{}, err
	}
	current := domain.NormalizeStatus(record.Status)
	if !isAllowedStatusTransition(current, target) {
		return domain.Record{}, fmt.Errorf("invalid status transition from %s to %s", current, target)
	}

	now := s.now()
	record.Status = target
	record.UpdatedAt = now
	if target == domain.StatusResolved {
		record.ResolvedAt = &now
	} else if target == domain.StatusOpen {
		record.ResolvedAt = nil
	}
	record.MetadataJSON = mergeMap(record.MetadataJSON, map[string]any{
		"status_updated_by": strings.TrimSpace(input.UpdatedBy),
		"status_reason":     strings.TrimSpace(input.Reason),
		"status_updated_at": now.Format(time.RFC3339),
	})
	return s.store.UpdateScopedRecord(ctx, record)
}

func (s *Service) FetchPriorCycleSummaries(ctx context.Context, ns domain.Namespace) ([]string, error) {
	records, err := s.listAllByClass(ctx, ns, domain.MemoryClassPriorCycleSummaries)
	if err != nil {
		return nil, err
	}
	return compactTexts(records, compactRule{
		Namespace:           ns,
		MaxItems:            defaultPriorSummaryMax,
		ExcludeCurrentCycle: true,
		IncludeResolved:     true,
	}), nil
}

func (s *Service) FetchUnresolvedGaps(ctx context.Context, ns domain.Namespace) ([]string, error) {
	return s.fetchCarryForwardOpenClass(ctx, ns, domain.MemoryClassUnresolvedGaps)
}

func (s *Service) FetchCarryForwardRisks(ctx context.Context, ns domain.Namespace) ([]string, error) {
	return s.fetchCarryForwardOpenClass(ctx, ns, domain.MemoryClassCarryForwardRisks)
}

func (s *Service) FetchBacklogItems(ctx context.Context, ns domain.Namespace) ([]string, error) {
	return s.fetchCarryForwardOpenClass(ctx, ns, domain.MemoryClassBacklogItems)
}

func (s *Service) FetchReviewerNotes(ctx context.Context, ns domain.Namespace) ([]string, error) {
	records, err := s.listAllByClass(ctx, ns, domain.MemoryClassReviewerNotes)
	if err != nil {
		return nil, err
	}
	return compactTexts(records, compactRule{
		Namespace:       ns,
		MaxItems:        defaultContextMaxPerClass,
		IncludeResolved: true,
	}), nil
}

func (s *Service) FetchCompactContext(ctx context.Context, ns domain.Namespace) (domain.CompactContext, error) {
	ns = domain.NormalizeNamespace(ns)
	if err := ns.Validate(); err != nil {
		return domain.CompactContext{}, err
	}

	prior, err := s.FetchPriorCycleSummaries(ctx, ns)
	if err != nil {
		return domain.CompactContext{}, err
	}
	risks, err := s.FetchCarryForwardRisks(ctx, ns)
	if err != nil {
		return domain.CompactContext{}, err
	}
	gaps, err := s.FetchUnresolvedGaps(ctx, ns)
	if err != nil {
		return domain.CompactContext{}, err
	}
	backlog, err := s.FetchBacklogItems(ctx, ns)
	if err != nil {
		return domain.CompactContext{}, err
	}
	notes, err := s.FetchReviewerNotes(ctx, ns)
	if err != nil {
		return domain.CompactContext{}, err
	}

	return domain.CompactContext{
		PriorCycleSummaries: prior,
		CarryForwardRisks:   risks,
		UnresolvedGaps:      gaps,
		BacklogItems:        backlog,
		ReviewerNotes:       notes,
	}, nil
}

func (s *Service) PersistNotes(ctx context.Context, ns domain.Namespace, input PersistNotesInput) (PersistNotesResult, error) {
	ns = domain.NormalizeNamespace(ns)
	if err := ns.Validate(); err != nil {
		return PersistNotesResult{}, err
	}

	status := domain.NormalizeStatus(input.Status)
	if status == "" {
		status = domain.StatusOpen
	}
	if err := domain.ValidateStatus(status); err != nil {
		return PersistNotesResult{}, err
	}

	createdBy := strings.TrimSpace(input.CreatedBy)
	if createdBy == "" {
		createdBy = "control-plane"
	}
	metadata := cloneMap(input.MetadataJSON)
	if len(input.Provenance) > 0 {
		metadata["provenance"] = cloneMap(input.Provenance)
	}
	provenance := extractProvenance(input)

	recordIDs := make([]string, 0)
	recordIDs, err := s.appendTextRecords(ctx, recordIDs, ns, domain.MemoryClassWorkingContext, appendIfPresent(input.WorkingContext, input.Note), status, createdBy, input.ContentJSON, metadata, input.ExpiresAt, provenance)
	if err != nil {
		return PersistNotesResult{}, err
	}
	recordIDs, err = s.appendTextRecords(ctx, recordIDs, ns, domain.MemoryClassPriorCycleSummaries, append(input.PriorCycleSummaries, input.CycleSummaries...), status, createdBy, nil, metadata, input.ExpiresAt, provenance)
	if err != nil {
		return PersistNotesResult{}, err
	}
	recordIDs, err = s.appendTextRecords(ctx, recordIDs, ns, domain.MemoryClassCarryForwardRisks, input.CarryForwardRisks, status, createdBy, nil, metadata, input.ExpiresAt, provenance)
	if err != nil {
		return PersistNotesResult{}, err
	}

	recordIDs, err = s.upsertUnresolvedGaps(ctx, recordIDs, ns, input.UnresolvedGaps, createdBy, metadata, input.ExpiresAt, provenance)
	if err != nil {
		return PersistNotesResult{}, err
	}
	recordIDs, err = s.appendTextRecords(ctx, recordIDs, ns, domain.MemoryClassBacklogItems, input.BacklogItems, status, createdBy, nil, metadata, input.ExpiresAt, provenance)
	if err != nil {
		return PersistNotesResult{}, err
	}
	recordIDs, err = s.appendTextRecords(ctx, recordIDs, ns, domain.MemoryClassReviewerNotes, input.ReviewerNotes, status, createdBy, nil, metadata, input.ExpiresAt, provenance)
	if err != nil {
		return PersistNotesResult{}, err
	}
	recordIDs, err = s.appendTextRecords(ctx, recordIDs, ns, domain.MemoryClassPolicyExceptions, input.PolicyExceptions, status, createdBy, nil, metadata, input.ExpiresAt, provenance)
	if err != nil {
		return PersistNotesResult{}, err
	}

	resolved := make([]string, 0)
	resolved, err = s.resolveByIDs(ctx, resolved, append([]string{}, input.ResolvedGapIDs...))
	if err != nil {
		return PersistNotesResult{}, err
	}
	resolved, err = s.resolveByIDs(ctx, resolved, append([]string{}, input.ResolvedRiskIDs...))
	if err != nil {
		return PersistNotesResult{}, err
	}
	resolved, err = s.resolveByIDs(ctx, resolved, append([]string{}, input.ResolvedBacklogItemIDs...))
	if err != nil {
		return PersistNotesResult{}, err
	}
	resolved, err = s.resolveByIDs(ctx, resolved, append([]string{}, input.ResolvedReviewerNoteIDs...))
	if err != nil {
		return PersistNotesResult{}, err
	}
	resolved, err = s.resolveByIDs(ctx, resolved, append([]string{}, input.ResolvedPolicyExceptionIDs...))
	if err != nil {
		return PersistNotesResult{}, err
	}
	resolved, err = s.resolveGapsByText(ctx, resolved, ns, input.ResolveUnresolvedGaps)
	if err != nil {
		return PersistNotesResult{}, err
	}

	var snapshot domain.Snapshot
	if len(recordIDs) > 0 || len(resolved) > 0 {
		summary := strings.TrimSpace(input.SnapshotSummary)
		if summary == "" {
			summary = "scoped memory checkpoint"
		}
		snapshot, err = s.CreateSnapshot(ctx, CreateSnapshotInput{
			Namespace:   ns,
			CreatedBy:   createdBy,
			Summary:     summary,
			RecordRefs:  append(append([]string{}, recordIDs...), resolved...),
			ManifestRef: strings.TrimSpace(input.SnapshotManifestRef),
			MetadataJSON: map[string]any{
				"record_count":   len(recordIDs),
				"resolved_count": len(resolved),
			},
		})
		if err != nil {
			return PersistNotesResult{}, err
		}

		if snapshot.SnapshotID != "" {
			_, err := s.createRecord(ctx, ns, domain.MemoryClassSnapshotReference, snapshot.SnapshotID, domain.StatusOpen, createdBy, nil, map[string]any{
				"summary":      snapshot.Summary,
				"snapshot_id":  snapshot.SnapshotID,
				"record_count": len(snapshot.RecordRefs),
				"checksum":     snapshot.ManifestChecksum,
			}, nil, provenance)
			if err != nil {
				return PersistNotesResult{}, err
			}
		}
	}

	return PersistNotesResult{
		SnapshotRef:       snapshot.SnapshotID,
		Snapshot:          snapshot,
		RecordsWritten:    len(recordIDs),
		RecordIDs:         domain.CleanTexts(recordIDs),
		ResolvedRecordIDs: domain.CleanTexts(resolved),
	}, nil
}

func (s *Service) CreateSnapshot(ctx context.Context, input CreateSnapshotInput) (domain.Snapshot, error) {
	ns := domain.NormalizeNamespace(input.Namespace)
	if err := ns.Validate(); err != nil {
		return domain.Snapshot{}, err
	}
	createdBy := strings.TrimSpace(input.CreatedBy)
	if createdBy == "" {
		createdBy = "control-plane"
	}
	summary := strings.TrimSpace(input.Summary)
	if summary == "" {
		summary = "scoped memory snapshot"
	}

	criteria := domain.Query{}
	if input.QueryCriteria != nil {
		criteria = domain.NormalizeQuery(*input.QueryCriteria)
	}
	if criteria.RepoNamespace == "" {
		criteria.RepoNamespace = ns.RepoNamespace
	}
	if criteria.RunNamespace == "" {
		criteria.RunNamespace = ns.RunNamespace
	}
	if criteria.CycleNamespace == "" && ns.CycleNamespace != "" {
		criteria.CycleNamespace = ns.CycleNamespace
	}

	recordRefs := domain.CleanTexts(input.RecordRefs)
	if len(recordRefs) == 0 {
		records, err := s.listAllRecords(ctx, criteria)
		if err != nil {
			return domain.Snapshot{}, err
		}
		recordRefs = make([]string, 0, len(records))
		for _, record := range records {
			recordRefs = append(recordRefs, record.ID)
		}
		recordRefs = domain.CleanTexts(recordRefs)
	}

	snapshot := domain.Snapshot{
		SnapshotID:     s.snapshotIDGen(),
		RepoNamespace:  ns.RepoNamespace,
		RunNamespace:   ns.RunNamespace,
		CycleNamespace: ns.CycleNamespace,
		CreatedAt:      s.now(),
		CreatedBy:      createdBy,
		Summary:        summary,
		RecordRefs:     recordRefs,
		QueryCriteria:  criteria,
		ManifestRef:    strings.TrimSpace(input.ManifestRef),
		MetadataJSON:   cloneMap(input.MetadataJSON),
	}
	previousSnapshot, err := s.latestSnapshotForScope(ctx, ns)
	if err != nil {
		return domain.Snapshot{}, err
	}
	snapshot.PreviousSnapshotChecksum = previousSnapshot.ManifestChecksum
	snapshot.ManifestChecksum, err = computeSnapshotChecksum(snapshot)
	if err != nil {
		return domain.Snapshot{}, err
	}

	return s.store.CreateScopedSnapshot(ctx, snapshot)
}

func (s *Service) GetSnapshot(ctx context.Context, snapshotID string) (domain.Snapshot, error) {
	return s.store.GetScopedSnapshot(ctx, strings.TrimSpace(snapshotID))
}

func (s *Service) ListSnapshots(ctx context.Context, query domain.SnapshotQuery) (domain.SnapshotQueryResult, error) {
	return s.store.ListScopedSnapshots(ctx, domain.NormalizeSnapshotQuery(query))
}

func (s *Service) ExportSnapshot(ctx context.Context, snapshotID string) (domain.SnapshotExport, error) {
	snapshot, err := s.GetSnapshot(ctx, snapshotID)
	if err != nil {
		return domain.SnapshotExport{}, err
	}

	records := make([]domain.Record, 0, len(snapshot.RecordRefs))
	for _, recordID := range snapshot.RecordRefs {
		record, err := s.store.GetScopedRecord(ctx, recordID)
		if err != nil {
			if errors.Is(err, storepkg.ErrNotFound) {
				continue
			}
			return domain.SnapshotExport{}, err
		}
		records = append(records, record)
	}
	if len(records) == 0 {
		records, err = s.listAllRecords(ctx, snapshot.QueryCriteria)
		if err != nil {
			return domain.SnapshotExport{}, err
		}
	}

	return domain.SnapshotExport{
		Snapshot: snapshot,
		Records:  records,
	}, nil
}

func (s *Service) ExportRun(ctx context.Context, ns domain.Namespace) (domain.RunMemoryExport, error) {
	ns = domain.NormalizeNamespace(ns)
	if err := ns.Validate(); err != nil {
		return domain.RunMemoryExport{}, err
	}

	records, err := s.listAllRecords(ctx, domain.Query{
		RepoNamespace:  ns.RepoNamespace,
		RunNamespace:   ns.RunNamespace,
		CycleNamespace: ns.CycleNamespace,
		Limit:          domain.MaxPageSize,
	})
	if err != nil {
		return domain.RunMemoryExport{}, err
	}
	snapshots, err := s.listAllSnapshots(ctx, domain.SnapshotQuery{
		RepoNamespace:  ns.RepoNamespace,
		RunNamespace:   ns.RunNamespace,
		CycleNamespace: ns.CycleNamespace,
		Limit:          domain.MaxPageSize,
	})
	if err != nil {
		return domain.RunMemoryExport{}, err
	}

	classCounts := map[string]int{}
	statusCounts := map[string]int{}
	for _, record := range records {
		classCounts[string(record.MemoryClass)]++
		statusCounts[string(record.Status)]++
	}

	snapshotRefs := make([]string, 0, len(snapshots))
	snapshotChecksums := make([]string, 0, len(snapshots))
	latestSnapshotChecksum := ""
	var latestSnapshotCreatedAt time.Time
	for _, snapshot := range snapshots {
		snapshotRefs = append(snapshotRefs, snapshot.SnapshotID)
		snapshotChecksums = append(snapshotChecksums, snapshot.ManifestChecksum)
		if latestSnapshotChecksum == "" || snapshot.CreatedAt.After(latestSnapshotCreatedAt) {
			latestSnapshotChecksum = snapshot.ManifestChecksum
			latestSnapshotCreatedAt = snapshot.CreatedAt
		}
	}
	slices.Sort(snapshotRefs)
	slices.Sort(snapshotChecksums)

	return domain.RunMemoryExport{
		RepoNamespace:  ns.RepoNamespace,
		RunNamespace:   ns.RunNamespace,
		CycleNamespace: ns.CycleNamespace,
		GeneratedAt:    s.now(),
		Records:        records,
		Snapshots:      snapshots,
		ClassCounts:    classCounts,
		StatusCounts:   statusCounts,
		Manifest: map[string]any{
			"snapshot_refs":            snapshotRefs,
			"snapshot_checksums":       snapshotChecksums,
			"latest_snapshot_checksum": latestSnapshotChecksum,
			"record_count":             len(records),
			"cycle_scope":              ns.CycleNamespace,
		},
	}, nil
}

func (s *Service) fetchCarryForwardOpenClass(ctx context.Context, ns domain.Namespace, class domain.MemoryClass) ([]string, error) {
	records, err := s.listAllByClass(ctx, ns, class)
	if err != nil {
		return nil, err
	}
	return compactTexts(records, compactRule{
		Namespace:       ns,
		MaxItems:        defaultContextMaxPerClass,
		IncludeResolved: true,
	}), nil
}

func (s *Service) listAllByClass(ctx context.Context, ns domain.Namespace, class domain.MemoryClass) ([]domain.Record, error) {
	query := domain.Query{
		RepoNamespace: ns.RepoNamespace,
		RunNamespace:  ns.RunNamespace,
		MemoryClass:   class,
		Limit:         domain.MaxPageSize,
	}
	return s.listAllRecords(ctx, query)
}

func (s *Service) listAllRecords(ctx context.Context, query domain.Query) ([]domain.Record, error) {
	query = domain.NormalizeQuery(query)
	all := make([]domain.Record, 0)
	offset := query.Offset
	for {
		page, err := s.store.ListScopedRecords(ctx, domain.Query{
			RepoNamespace:          query.RepoNamespace,
			RunNamespace:           query.RunNamespace,
			CycleNamespace:         query.CycleNamespace,
			AgentNamespace:         query.AgentNamespace,
			MemoryClass:            query.MemoryClass,
			Status:                 query.Status,
			SourceRunID:            query.SourceRunID,
			SourceCycleID:          query.SourceCycleID,
			SourceArtifactID:       query.SourceArtifactID,
			SourcePolicyDecisionID: query.SourcePolicyDecisionID,
			SourceModelProfileID:   query.SourceModelProfileID,
			Limit:                  query.Limit,
			Offset:                 offset,
		})
		if err != nil {
			return nil, err
		}
		all = append(all, page.Records...)
		if !page.HasMore {
			break
		}
		offset += query.Limit
	}
	return all, nil
}

func (s *Service) listAllSnapshots(ctx context.Context, query domain.SnapshotQuery) ([]domain.Snapshot, error) {
	query = domain.NormalizeSnapshotQuery(query)
	all := make([]domain.Snapshot, 0)
	offset := query.Offset
	for {
		page, err := s.store.ListScopedSnapshots(ctx, domain.SnapshotQuery{
			RepoNamespace:  query.RepoNamespace,
			RunNamespace:   query.RunNamespace,
			CycleNamespace: query.CycleNamespace,
			Limit:          query.Limit,
			Offset:         offset,
		})
		if err != nil {
			return nil, err
		}
		all = append(all, page.Snapshots...)
		if !page.HasMore {
			break
		}
		offset += query.Limit
	}
	return all, nil
}

func (s *Service) createRecord(
	ctx context.Context,
	ns domain.Namespace,
	class domain.MemoryClass,
	text string,
	status domain.Status,
	createdBy string,
	contentJSON map[string]any,
	metadata map[string]any,
	expiresAt *time.Time,
	provenance recordProvenance,
) (domain.Record, error) {
	now := s.now()
	record := domain.Record{
		ID:                     s.recordIDGen(),
		RepoNamespace:          ns.RepoNamespace,
		RunNamespace:           ns.RunNamespace,
		CycleNamespace:         ns.CycleNamespace,
		AgentNamespace:         ns.AgentNamespace,
		MemoryClass:            domain.NormalizeClass(class),
		Status:                 domain.NormalizeStatus(status),
		ContentText:            strings.TrimSpace(text),
		ContentJSON:            cloneMap(contentJSON),
		MetadataJSON:           cloneMap(metadata),
		CreatedBy:              strings.TrimSpace(createdBy),
		CreatedAt:              now,
		UpdatedAt:              now,
		ExpiresAt:              expiresAt,
		SourceRunID:            strings.TrimSpace(provenance.SourceRunID),
		SourceCycleID:          strings.TrimSpace(provenance.SourceCycleID),
		SourceArtifactID:       strings.TrimSpace(provenance.SourceArtifactID),
		SourcePolicyDecisionID: strings.TrimSpace(provenance.SourcePolicyDecisionID),
		SourceModelProfileID:   strings.TrimSpace(provenance.SourceModelProfileID),
	}
	if record.Status == "" {
		record.Status = domain.StatusOpen
	}
	if record.Status == domain.StatusResolved {
		resolvedAt := now
		record.ResolvedAt = &resolvedAt
	}
	return s.store.CreateScopedRecord(ctx, record)
}

func (s *Service) appendTextRecords(
	ctx context.Context,
	recordIDs []string,
	ns domain.Namespace,
	class domain.MemoryClass,
	values []string,
	status domain.Status,
	createdBy string,
	contentJSON map[string]any,
	metadata map[string]any,
	expiresAt *time.Time,
	provenance recordProvenance,
) ([]string, error) {
	for _, value := range domain.CleanTexts(values) {
		record, err := s.createRecord(ctx, ns, class, value, status, createdBy, contentJSON, metadata, expiresAt, provenance)
		if err != nil {
			return nil, err
		}
		recordIDs = append(recordIDs, record.ID)
	}
	return recordIDs, nil
}

func (s *Service) upsertUnresolvedGaps(
	ctx context.Context,
	recordIDs []string,
	ns domain.Namespace,
	values []string,
	createdBy string,
	metadata map[string]any,
	expiresAt *time.Time,
	provenance recordProvenance,
) ([]string, error) {
	for _, value := range domain.CleanTexts(values) {
		record, err := s.findOpenByClassAndText(ctx, ns, domain.MemoryClassUnresolvedGaps, value)
		if err != nil {
			return nil, err
		}
		if record.ID == "" {
			record, err = s.createRecord(ctx, ns, domain.MemoryClassUnresolvedGaps, value, domain.StatusOpen, createdBy, nil, metadata, expiresAt, provenance)
			if err != nil {
				return nil, err
			}
			recordIDs = append(recordIDs, record.ID)
			continue
		}
		record.ContentText = value
		record.MetadataJSON = mergeMap(record.MetadataJSON, metadata)
		record.UpdatedAt = s.now()
		record.SourceRunID = firstNonEmpty(record.SourceRunID, provenance.SourceRunID)
		record.SourceCycleID = firstNonEmpty(record.SourceCycleID, provenance.SourceCycleID)
		record.SourceArtifactID = firstNonEmpty(record.SourceArtifactID, provenance.SourceArtifactID)
		record.SourcePolicyDecisionID = firstNonEmpty(record.SourcePolicyDecisionID, provenance.SourcePolicyDecisionID)
		record.SourceModelProfileID = firstNonEmpty(record.SourceModelProfileID, provenance.SourceModelProfileID)
		updated, err := s.store.UpdateScopedRecord(ctx, record)
		if err != nil {
			return nil, err
		}
		recordIDs = append(recordIDs, updated.ID)
	}
	return recordIDs, nil
}

func (s *Service) findOpenByClassAndText(ctx context.Context, ns domain.Namespace, class domain.MemoryClass, text string) (domain.Record, error) {
	records, err := s.listAllRecords(ctx, domain.Query{
		RepoNamespace: ns.RepoNamespace,
		RunNamespace:  ns.RunNamespace,
		MemoryClass:   class,
		Status:        domain.StatusOpen,
		Limit:         domain.MaxPageSize,
	})
	if err != nil {
		return domain.Record{}, err
	}
	trimmedText := strings.TrimSpace(text)
	for _, record := range records {
		if strings.TrimSpace(record.ContentText) != trimmedText {
			continue
		}
		if ns.CycleNamespace != "" && record.CycleNamespace != ns.CycleNamespace {
			continue
		}
		if ns.AgentNamespace != "" && record.AgentNamespace != ns.AgentNamespace {
			continue
		}
		return record, nil
	}
	return domain.Record{}, nil
}

func (s *Service) resolveByIDs(ctx context.Context, resolved []string, ids []string) ([]string, error) {
	for _, recordID := range domain.CleanTexts(ids) {
		record, err := s.store.GetScopedRecord(ctx, recordID)
		if err != nil {
			if errors.Is(err, storepkg.ErrNotFound) {
				continue
			}
			return nil, err
		}
		if record.Status == domain.StatusResolved {
			resolved = append(resolved, record.ID)
			continue
		}
		if !isActionableMemoryClass(record.MemoryClass) {
			continue
		}
		now := s.now()
		record.Status = domain.StatusResolved
		record.ResolvedAt = &now
		record.UpdatedAt = now
		updated, err := s.store.UpdateScopedRecord(ctx, record)
		if err != nil {
			return nil, err
		}
		resolved = append(resolved, updated.ID)
	}
	return resolved, nil
}

func (s *Service) resolveGapsByText(ctx context.Context, resolved []string, ns domain.Namespace, values []string) ([]string, error) {
	values = domain.CleanTexts(values)
	if len(values) == 0 {
		return resolved, nil
	}
	records, err := s.listAllRecords(ctx, domain.Query{
		RepoNamespace: ns.RepoNamespace,
		RunNamespace:  ns.RunNamespace,
		MemoryClass:   domain.MemoryClassUnresolvedGaps,
		Status:        domain.StatusOpen,
		Limit:         domain.MaxPageSize,
	})
	if err != nil {
		return nil, err
	}
	valueSet := make(map[string]struct{}, len(values))
	for _, value := range values {
		valueSet[strings.TrimSpace(value)] = struct{}{}
	}
	for _, record := range records {
		if _, ok := valueSet[strings.TrimSpace(record.ContentText)]; !ok {
			continue
		}
		now := s.now()
		record.Status = domain.StatusResolved
		record.ResolvedAt = &now
		record.UpdatedAt = now
		updated, err := s.store.UpdateScopedRecord(ctx, record)
		if err != nil {
			return nil, err
		}
		resolved = append(resolved, updated.ID)
	}
	return resolved, nil
}

func appendIfPresent(values []string, value string) []string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return values
	}
	return append(values, trimmed)
}

type compactRule struct {
	Namespace           domain.Namespace
	MaxItems            int
	ExcludeCurrentCycle bool
	IncludeResolved     bool
}

func compactTexts(records []domain.Record, rule compactRule) []string {
	maxItems := rule.MaxItems
	if maxItems <= 0 {
		maxItems = defaultContextMaxPerClass
	}
	filtered := make([]domain.Record, 0, len(records))
	for _, record := range records {
		if record.Status == domain.StatusArchived {
			continue
		}
		if rule.ExcludeCurrentCycle && rule.Namespace.CycleNamespace != "" && strings.TrimSpace(record.CycleNamespace) == strings.TrimSpace(rule.Namespace.CycleNamespace) {
			continue
		}
		if !rule.IncludeResolved && record.Status != domain.StatusOpen {
			continue
		}
		filtered = append(filtered, record)
	}

	slices.SortFunc(filtered, func(a, b domain.Record) int {
		if score := scopePriority(a, rule.Namespace) - scopePriority(b, rule.Namespace); score != 0 {
			return score
		}
		if score := statusPriority(a.Status) - statusPriority(b.Status); score != 0 {
			return score
		}
		if !a.UpdatedAt.Equal(b.UpdatedAt) {
			if a.UpdatedAt.After(b.UpdatedAt) {
				return -1
			}
			return 1
		}
		if !a.CreatedAt.Equal(b.CreatedAt) {
			if a.CreatedAt.After(b.CreatedAt) {
				return -1
			}
			return 1
		}
		return strings.Compare(a.ID, b.ID)
	})

	unique := make([]string, 0, len(filtered))
	seen := make(map[string]struct{}, len(filtered))
	for _, record := range filtered {
		text := strings.TrimSpace(record.ContentText)
		if text == "" {
			continue
		}
		if _, ok := seen[text]; ok {
			continue
		}
		seen[text] = struct{}{}
		unique = append(unique, text)
		if len(unique) >= maxItems {
			break
		}
	}
	return unique
}

func statusPriority(status domain.Status) int {
	switch domain.NormalizeStatus(status) {
	case domain.StatusOpen:
		return 0
	case domain.StatusResolved:
		return 1
	case domain.StatusSuperseded:
		return 2
	case domain.StatusArchived:
		return 3
	default:
		return 4
	}
}

func scopePriority(record domain.Record, ns domain.Namespace) int {
	recordCycle := strings.TrimSpace(record.CycleNamespace)
	recordAgent := strings.TrimSpace(record.AgentNamespace)
	nsCycle := strings.TrimSpace(ns.CycleNamespace)
	nsAgent := strings.TrimSpace(ns.AgentNamespace)
	if nsCycle != "" && recordCycle == nsCycle && nsAgent != "" && recordAgent == nsAgent {
		return 0
	}
	if nsCycle != "" && recordCycle == nsCycle {
		return 1
	}
	if recordCycle == "" {
		return 2
	}
	return 3
}

func isActionableMemoryClass(class domain.MemoryClass) bool {
	switch domain.NormalizeClass(class) {
	case domain.MemoryClassUnresolvedGaps, domain.MemoryClassCarryForwardRisks, domain.MemoryClassBacklogItems, domain.MemoryClassReviewerNotes, domain.MemoryClassPolicyExceptions:
		return true
	default:
		return false
	}
}

func isAllowedStatusTransition(current domain.Status, next domain.Status) bool {
	current = domain.NormalizeStatus(current)
	next = domain.NormalizeStatus(next)
	if current == next {
		return true
	}
	switch current {
	case domain.StatusOpen:
		return next == domain.StatusResolved || next == domain.StatusSuperseded || next == domain.StatusArchived
	case domain.StatusResolved:
		return next == domain.StatusSuperseded || next == domain.StatusArchived
	case domain.StatusSuperseded:
		return next == domain.StatusArchived
	case domain.StatusArchived:
		return false
	default:
		return false
	}
}

func extractProvenance(input PersistNotesInput) recordProvenance {
	return recordProvenance{
		SourceRunID:            firstNonEmpty(input.SourceRunID, valueFromMap(input.Provenance, "source_run_id")),
		SourceCycleID:          firstNonEmpty(input.SourceCycleID, valueFromMap(input.Provenance, "source_cycle_id")),
		SourceArtifactID:       firstNonEmpty(input.SourceArtifactID, valueFromMap(input.Provenance, "source_artifact_id")),
		SourcePolicyDecisionID: firstNonEmpty(input.SourcePolicyDecisionID, valueFromMap(input.Provenance, "source_policy_decision_id")),
		SourceModelProfileID:   firstNonEmpty(input.SourceModelProfileID, valueFromMap(input.Provenance, "source_model_profile_id")),
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func valueFromMap(input map[string]any, key string) string {
	if input == nil {
		return ""
	}
	value, ok := input[key]
	if !ok || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", typed))
	}
}

func (s *Service) latestSnapshotForScope(ctx context.Context, ns domain.Namespace) (domain.Snapshot, error) {
	listed, err := s.store.ListScopedSnapshots(ctx, domain.SnapshotQuery{
		RepoNamespace:  ns.RepoNamespace,
		RunNamespace:   ns.RunNamespace,
		CycleNamespace: ns.CycleNamespace,
		Limit:          domain.MaxPageSize,
		Offset:         0,
	})
	if err != nil {
		return domain.Snapshot{}, err
	}
	if len(listed.Snapshots) == 0 {
		return domain.Snapshot{}, nil
	}
	latest := listed.Snapshots[0]
	for _, snapshot := range listed.Snapshots[1:] {
		if snapshot.CreatedAt.After(latest.CreatedAt) {
			latest = snapshot
		}
	}
	return latest, nil
}

func computeSnapshotChecksum(snapshot domain.Snapshot) (string, error) {
	recordRefs := append([]string{}, snapshot.RecordRefs...)
	slices.Sort(recordRefs)
	payload := struct {
		SnapshotID               string         `json:"snapshot_id"`
		RepoNamespace            string         `json:"repo_namespace"`
		RunNamespace             string         `json:"run_namespace"`
		CycleNamespace           string         `json:"cycle_namespace,omitempty"`
		CreatedBy                string         `json:"created_by"`
		CreatedAt                string         `json:"created_at"`
		Summary                  string         `json:"summary"`
		RecordRefs               []string       `json:"record_refs"`
		QueryCriteria            domain.Query   `json:"query_criteria"`
		ManifestRef              string         `json:"manifest_ref,omitempty"`
		PreviousSnapshotChecksum string         `json:"previous_snapshot_checksum,omitempty"`
		MetadataJSON             map[string]any `json:"metadata_json,omitempty"`
	}{
		SnapshotID:               snapshot.SnapshotID,
		RepoNamespace:            snapshot.RepoNamespace,
		RunNamespace:             snapshot.RunNamespace,
		CycleNamespace:           snapshot.CycleNamespace,
		CreatedBy:                snapshot.CreatedBy,
		CreatedAt:                snapshot.CreatedAt.UTC().Format(time.RFC3339Nano),
		Summary:                  snapshot.Summary,
		RecordRefs:               recordRefs,
		QueryCriteria:            snapshot.QueryCriteria,
		ManifestRef:              snapshot.ManifestRef,
		PreviousSnapshotChecksum: snapshot.PreviousSnapshotChecksum,
		MetadataJSON:             cloneMap(snapshot.MetadataJSON),
	}
	serialized, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(serialized)
	return hex.EncodeToString(sum[:]), nil
}

func cloneMap(input map[string]any) map[string]any {
	if input == nil {
		return map[string]any{}
	}
	cloned := make(map[string]any, len(input))
	for key, value := range input {
		cloned[key] = value
	}
	return cloned
}

func mergeMap(base map[string]any, extra map[string]any) map[string]any {
	merged := cloneMap(base)
	for key, value := range extra {
		merged[key] = value
	}
	return merged
}
