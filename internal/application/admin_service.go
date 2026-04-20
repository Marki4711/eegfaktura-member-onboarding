package application

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/your-org/eegfaktura-member-onboarding/internal/shared"
)

// ApplicationListFilters holds optional filter parameters for the admin list endpoint.
type ApplicationListFilters struct {
	Status          *string
	ReferenceNumber *string
	Lastname        *string
	Email           *string
	MeteringPoint   *string
	SubmittedFrom   *time.Time
	SubmittedTo     *time.Time
	// RCNumbers restricts results to a specific set of RC numbers (tenant-admin scope).
	// When nil, no restriction is applied (superuser scope).
	RCNumbers *[]string
}

// adminTransitions defines which status changes the admin endpoint may perform.
// Import-related transitions (approved→imported etc.) are handled by the
// dedicated import endpoint (PROJ-3) and must not appear here.
var adminTransitions = map[shared.ApplicationStatus][]shared.ApplicationStatus{
	shared.StatusSubmitted:   {shared.StatusUnderReview},
	shared.StatusUnderReview: {shared.StatusNeedsInfo, shared.StatusApproved, shared.StatusRejected},
	shared.StatusNeedsInfo:   {shared.StatusSubmitted},
}

// AdminApplicationService implements admin review business logic.
type AdminApplicationService struct {
	db            *sql.DB
	appRepo       *ApplicationRepository
	meteringRepo  *MeteringPointRepository
	statusLogRepo *StatusLogRepository
}

// NewAdminApplicationService creates an AdminApplicationService.
func NewAdminApplicationService(
	db *sql.DB,
	appRepo *ApplicationRepository,
	meteringRepo *MeteringPointRepository,
	statusLogRepo *StatusLogRepository,
) *AdminApplicationService {
	return &AdminApplicationService{
		db:            db,
		appRepo:       appRepo,
		meteringRepo:  meteringRepo,
		statusLogRepo: statusLogRepo,
	}
}

// ListApplications returns a paginated, filtered list of applications for admin review.
func (s *AdminApplicationService) ListApplications(filters ApplicationListFilters, page, pageSize int) (*shared.ApplicationListResponse, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	} else if pageSize > 100 {
		pageSize = 100
	}

	items, total, err := s.appRepo.List(filters, page, pageSize)
	if err != nil {
		return nil, fmt.Errorf("failed to list applications: %w", err)
	}

	// Enrich each item with its metering point numbers.
	if len(items) > 0 {
		ids := make([]uuid.UUID, len(items))
		for i, item := range items {
			ids[i] = item.ID
		}
		mpMap, err := s.meteringRepo.GetNumbersByApplicationIDs(ids)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch metering points: %w", err)
		}
		for i := range items {
			if nums, ok := mpMap[items[i].ID]; ok {
				items[i].MeteringPoints = nums
			} else {
				items[i].MeteringPoints = []string{}
			}
		}
	}

	return &shared.ApplicationListResponse{
		Items:    items,
		Page:     page,
		PageSize: pageSize,
		Total:    total,
	}, nil
}

// GetApplicationDetail returns the full detail view for a single application.
func (s *AdminApplicationService) GetApplicationDetail(id uuid.UUID) (*shared.AdminApplicationDetailResponse, error) {
	app, err := s.appRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	meteringPoints, err := s.meteringRepo.GetByApplicationID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch metering points: %w", err)
	}
	if meteringPoints == nil {
		meteringPoints = []shared.MeteringPoint{}
	}

	statusLog, err := s.statusLogRepo.GetByApplicationID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch status log: %w", err)
	}
	if statusLog == nil {
		statusLog = []shared.StatusLogEntry{}
	}

	return &shared.AdminApplicationDetailResponse{
		Application:    *app,
		MeteringPoints: meteringPoints,
		StatusLog:      statusLog,
	}, nil
}

// AdminUpdateApplication applies a partial admin update to a draft or needs_info application.
func (s *AdminApplicationService) AdminUpdateApplication(id uuid.UUID, req shared.AdminUpdateApplicationRequest) (*shared.ApplicationResponse, error) {
	app, err := s.appRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Admin may edit applications in pre-import active states.
	// draft, rejected, and imported are not editable by admin.
	allowed := map[shared.ApplicationStatus]bool{
		shared.StatusSubmitted:    true,
		shared.StatusUnderReview:  true,
		shared.StatusNeedsInfo:    true,
		shared.StatusApproved:     true,
		shared.StatusImportFailed: true,
	}
	if !allowed[app.Status] {
		return nil, shared.NewConflictError("application cannot be edited in its current status")
	}

	// Apply partial updates.
	if req.MemberType != nil {
		app.MemberType = shared.MemberType(*req.MemberType)
	}
	if req.Firstname != nil {
		app.Firstname = trimStringPtr(req.Firstname)
	}
	if req.Lastname != nil {
		app.Lastname = trimStringPtr(req.Lastname)
	}
	if req.CompanyName != nil {
		app.CompanyName = trimStringPtr(req.CompanyName)
	}
	if req.UIDNumber != nil {
		app.UIDNumber = trimStringPtr(req.UIDNumber)
	}
	if req.RegisterNumber != nil {
		app.RegisterNumber = trimStringPtr(req.RegisterNumber)
	}
	if req.BirthDate != nil {
		parsed, err := parseDateString(req.BirthDate)
		if err != nil {
			return nil, shared.NewValidationError("Validation failed", map[string]string{"birthDate": err.Error()})
		}
		app.BirthDate = parsed
	}
	if req.Email != nil {
		app.Email = *req.Email
	}
	if req.Phone != nil {
		app.Phone = req.Phone
	}
	if req.ResidentStreet != nil {
		app.ResidentStreet = *req.ResidentStreet
	}
	if req.ResidentStreetNumber != nil {
		app.ResidentStreetNumber = *req.ResidentStreetNumber
	}
	if req.ResidentZip != nil {
		app.ResidentZip = *req.ResidentZip
	}
	if req.ResidentCity != nil {
		app.ResidentCity = *req.ResidentCity
	}
	if req.AdminNote != nil {
		app.AdminNote = req.AdminNote
	}
	clearMemberTypeFields(app)
	if err = validateMemberTypeFields(app); err != nil {
		return nil, err
	}

	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if req.MeteringPoints != nil {
		now := time.Now().UTC()
		points := make([]shared.MeteringPoint, len(req.MeteringPoints))
		for i, mp := range req.MeteringPoints {
			points[i] = shared.MeteringPoint{
				MeteringPoint: mp.MeteringPoint,
				Direction:     shared.MeterDirection(mp.Direction),
				CreatedAt:     now,
				UpdatedAt:     now,
			}
		}
		if err := s.meteringRepo.CreateBulkTx(tx, id, points); err != nil {
			return nil, fmt.Errorf("failed to update metering points: %w", err)
		}
	}

	if err := s.appRepo.UpdateAdminTx(tx, app); err != nil {
		return nil, fmt.Errorf("failed to update application: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &shared.ApplicationResponse{
		ID:              app.ID,
		ReferenceNumber: app.ReferenceNumber,
		Status:          string(app.Status),
		CreatedAt:       app.CreatedAt,
		UpdatedAt:       app.UpdatedAt,
	}, nil
}

// ChangeStatus performs an admin status transition and writes a status_log entry.
// actorID is the Keycloak user ID of the reviewer; pass "" until PROJ-4 adds auth.
func (s *AdminApplicationService) ChangeStatus(id uuid.UUID, toStatus shared.ApplicationStatus, reason, actorID string) (*shared.ChangeStatusResponse, error) {
	app, err := s.appRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	if !isAdminTransitionAllowed(app.Status, toStatus) {
		return nil, shared.NewConflictError(
			fmt.Sprintf("transition from %s to %s is not allowed", app.Status, toStatus),
		)
	}

	if requiresReason(toStatus) && reason == "" {
		return nil, shared.NewValidationError("Validation failed", map[string]string{
			"reason": "a reason is required for this status transition",
		})
	}

	now := time.Now().UTC()

	// Timestamp columns that vary by target status.
	var submittedAt, approvedAt, rejectedAt *time.Time
	var needsInfoReason *string

	switch toStatus {
	case shared.StatusSubmitted:
		submittedAt = &now
	case shared.StatusApproved:
		approvedAt = &now
	case shared.StatusRejected:
		rejectedAt = &now
	case shared.StatusNeedsInfo:
		r := reason
		needsInfoReason = &r
	}

	var actorPtr *string
	if actorID != "" {
		actorPtr = &actorID
	}

	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if err := s.appRepo.UpdateStatusAdminTx(tx, id, toStatus, submittedAt, approvedAt, rejectedAt, needsInfoReason, actorPtr); err != nil {
		return nil, fmt.Errorf("failed to update status: %w", err)
	}

	fromStatus := string(app.Status)
	toStatusStr := string(toStatus)
	var reasonPtr *string
	if reason != "" {
		reasonPtr = &reason
	}
	logEntry := &shared.StatusLogEntry{
		ApplicationID:   id,
		FromStatus:      &fromStatus,
		ToStatus:        toStatusStr,
		ChangedByUserID: actorPtr,
		Reason:          reasonPtr,
		CreatedAt:       now,
	}
	if err := s.statusLogRepo.CreateTx(tx, logEntry); err != nil {
		return nil, fmt.Errorf("failed to write status log: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &shared.ChangeStatusResponse{
		ID:     id,
		Status: string(toStatus),
	}, nil
}

func isAdminTransitionAllowed(from, to shared.ApplicationStatus) bool {
	allowed, ok := adminTransitions[from]
	if !ok {
		return false
	}
	for _, s := range allowed {
		if s == to {
			return true
		}
	}
	return false
}

func requiresReason(status shared.ApplicationStatus) bool {
	return status == shared.StatusNeedsInfo || status == shared.StatusRejected
}
