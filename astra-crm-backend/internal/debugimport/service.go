package debugimport

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrInvalidInput = errors.New("invalid debug import input")

const (
	defaultMethodType = "C2C"
	importComment     = "Исторический импорт данных из Fin_ALL: смена восстановлена по агрегированным оборотам Excel без исходной CSV-выписки. Расхождение подтверждено для переноса архивных данных в CRM."
)

type PasswordHasher func(password string) (string, error)

type Service struct {
	db           *pgxpool.Pool
	hashPassword PasswordHasher
	now          func() time.Time
	location     *time.Location
}

func NewService(db *pgxpool.Pool, hashPassword PasswordHasher) *Service {
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		loc = time.UTC
	}
	return &Service{
		db:           db,
		hashPassword: hashPassword,
		now:          time.Now,
		location:     loc,
	}
}

type ImportFinAllParams struct {
	ActorID  int64
	TeamID   int64
	FileName string
	Data     []byte
	BankCode string
	DryRun   bool
}

type ImportFinAllResult struct {
	DryRun                 bool           `json:"dryRun"`
	FileName               string         `json:"fileName"`
	SourceHash             string         `json:"sourceHash"`
	ParsedRows             int            `json:"parsedRows"`
	ParsedCircles          int            `json:"parsedCircles"`
	ImportedCircles        int            `json:"importedCircles"`
	SkippedExistingCircles int            `json:"skippedExistingCircles"`
	CreatedTraders         int            `json:"createdTraders"`
	CreatedRequisites      int            `json:"createdRequisites"`
	CreatedAssignments     int            `json:"createdAssignments"`
	CreatedShifts          int            `json:"createdShifts"`
	CreatedShiftRequisites int            `json:"createdShiftRequisites"`
	BlockedRequisites      int            `json:"blockedRequisites"`
	InboundTurnoverMinor   int64          `json:"inboundTurnoverMinor"`
	OutboundTurnoverMinor  int64          `json:"outboundTurnoverMinor"`
	ClosingBalanceMinor    int64          `json:"closingBalanceMinor"`
	Warnings               []parseWarning `json:"warnings"`
}

type FinAllImportJob struct {
	ID           int64               `json:"id"`
	TeamID       int64               `json:"teamId"`
	ActorID      int64               `json:"actorId"`
	FileName     string              `json:"fileName"`
	SourceHash   string              `json:"sourceHash"`
	DryRun       bool                `json:"dryRun"`
	Status       string              `json:"status"`
	Result       *ImportFinAllResult `json:"result,omitempty"`
	ErrorMessage *string             `json:"errorMessage,omitempty"`
	CreatedAt    time.Time           `json:"createdAt"`
	StartedAt    *time.Time          `json:"startedAt,omitempty"`
	FinishedAt   *time.Time          `json:"finishedAt,omitempty"`
}

type applyState struct {
	tradersByWorker     map[string]int64
	requisitesByPhone   map[string]int64
	shiftsByTraderDate  map[string]int64
	shiftTotals         map[int64]*shiftTotals
	existingSourceItems map[string]struct{}
}

type shiftTotals struct {
	TraderID      int64
	InboundMinor  int64
	OutboundMinor int64
	InboundCount  int64
	OutboundCount int64
}

func (s *Service) StartFinAllImport(ctx context.Context, params ImportFinAllParams) (FinAllImportJob, error) {
	if s == nil || s.db == nil || s.hashPassword == nil {
		return FinAllImportJob{}, errors.New("debug import service is not configured")
	}
	params.BankCode = strings.TrimSpace(params.BankCode)
	if params.ActorID <= 0 || params.TeamID <= 0 || len(params.Data) == 0 || params.BankCode == "" {
		return FinAllImportJob{}, ErrInvalidInput
	}
	sourceHash := hashBytes(params.Data)
	job, err := s.createJob(ctx, params, sourceHash)
	if err != nil {
		return FinAllImportJob{}, err
	}

	data := append([]byte(nil), params.Data...)
	go s.runFinAllImportJob(job.ID, ImportFinAllParams{
		ActorID:  params.ActorID,
		TeamID:   params.TeamID,
		FileName: params.FileName,
		Data:     data,
		BankCode: params.BankCode,
		DryRun:   params.DryRun,
	})

	return job, nil
}

func (s *Service) GetFinAllImportJob(ctx context.Context, teamID int64, jobID int64) (FinAllImportJob, error) {
	if s == nil || s.db == nil {
		return FinAllImportJob{}, errors.New("debug import service is not configured")
	}
	if teamID <= 0 || jobID <= 0 {
		return FinAllImportJob{}, ErrInvalidInput
	}
	return s.getJob(ctx, teamID, jobID)
}

func (s *Service) ImportFinAll(ctx context.Context, params ImportFinAllParams) (ImportFinAllResult, error) {
	if s == nil || s.db == nil || s.hashPassword == nil {
		return ImportFinAllResult{}, errors.New("debug import service is not configured")
	}
	if params.ActorID <= 0 || params.TeamID <= 0 || len(params.Data) == 0 {
		return ImportFinAllResult{}, ErrInvalidInput
	}
	bankCode := strings.TrimSpace(params.BankCode)
	if bankCode == "" {
		return ImportFinAllResult{}, ErrInvalidInput
	}

	rows, warnings, err := parseFinAllWorkbook(params.Data, s.location)
	if err != nil {
		return ImportFinAllResult{}, err
	}
	sourceHash := hashBytes(params.Data)
	result := ImportFinAllResult{
		DryRun:     params.DryRun,
		FileName:   params.FileName,
		SourceHash: sourceHash,
		ParsedRows: len(rows),
		Warnings:   warnings,
	}
	for _, row := range rows {
		result.ParsedCircles += len(row.Circles)
	}

	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return ImportFinAllResult{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	state, err := s.loadState(ctx, tx, params.TeamID, sourceHash, bankCode)
	if err != nil {
		return ImportFinAllResult{}, err
	}

	for _, row := range rows {
		traderID, created, err := s.ensureTrader(ctx, tx, params.TeamID, params.ActorID, row.Operator, state)
		if err != nil {
			return ImportFinAllResult{}, fmt.Errorf("ensure trader row %d: %w", row.SourceRow, err)
		}
		if created {
			result.CreatedTraders++
		}

		requisiteID, created, err := s.ensureRequisite(ctx, tx, params.TeamID, params.ActorID, bankCode, row, state)
		if err != nil {
			return ImportFinAllResult{}, fmt.Errorf("ensure requisite row %d phone %s card %s: %w", row.SourceRow, row.Phone, row.Card, err)
		}
		if created {
			result.CreatedRequisites++
		}

		for _, circle := range row.Circles {
			sourceKey := sourceItemKey(row.SourceRow, circle.Number)
			if _, exists := state.existingSourceItems[sourceKey]; exists {
				result.SkippedExistingCircles++
				continue
			}

			shiftID, created, err := s.ensureHistoricalShift(ctx, tx, params.TeamID, traderID, circle.Date, state)
			if err != nil {
				return ImportFinAllResult{}, fmt.Errorf("ensure shift row %d circle %d: %w", row.SourceRow, circle.Number, err)
			}
			if created {
				result.CreatedShifts++
			}

			assignmentID, err := s.createClosedAssignment(ctx, tx, params.TeamID, params.ActorID, traderID, requisiteID, circle)
			if err != nil {
				return ImportFinAllResult{}, fmt.Errorf("create assignment row %d circle %d: %w", row.SourceRow, circle.Number, err)
			}
			result.CreatedAssignments++

			shiftRequisiteID, err := s.createClosedShiftRequisite(ctx, tx, params.TeamID, traderID, requisiteID, assignmentID, shiftID, row, circle)
			if err != nil {
				return ImportFinAllResult{}, fmt.Errorf("create shift requisite row %d circle %d: %w", row.SourceRow, circle.Number, err)
			}
			result.CreatedShiftRequisites++

			if err := s.linkAssignmentToShiftRequisite(ctx, tx, params.TeamID, assignmentID, shiftRequisiteID, circle); err != nil {
				return ImportFinAllResult{}, fmt.Errorf("link assignment row %d circle %d: %w", row.SourceRow, circle.Number, err)
			}
			s.addShiftTotals(state, shiftID, traderID, circle)
			if circle.Blocked {
				if err := s.markRequisiteBlocked(ctx, tx, params.TeamID, requisiteID); err != nil {
					return ImportFinAllResult{}, fmt.Errorf("mark requisite blocked row %d circle %d: %w", row.SourceRow, circle.Number, err)
				}
				result.BlockedRequisites++
			}
			if err := s.rememberSourceItem(ctx, tx, params.TeamID, sourceHash, row.SourceRow, circle.Number, params.ActorID, traderID, requisiteID, assignmentID, shiftID, shiftRequisiteID); err != nil {
				return ImportFinAllResult{}, fmt.Errorf("remember source item row %d circle %d: %w", row.SourceRow, circle.Number, err)
			}
			state.existingSourceItems[sourceKey] = struct{}{}
			result.ImportedCircles++
			result.InboundTurnoverMinor += circle.InboundTurnoverMinor
			result.OutboundTurnoverMinor += circle.OutboundTurnoverMinor
			result.ClosingBalanceMinor += circle.ClosingBalanceMinor
		}
	}

	for shiftID, totals := range state.shiftTotals {
		if err := s.insertAcceptedReconciliations(ctx, tx, params.TeamID, params.ActorID, shiftID, *totals); err != nil {
			return ImportFinAllResult{}, fmt.Errorf("insert accepted reconciliations shift %d: %w", shiftID, err)
		}
	}

	if err := s.writeSummaryAudit(ctx, tx, params.TeamID, params.ActorID, result); err != nil {
		return ImportFinAllResult{}, fmt.Errorf("write import audit: %w", err)
	}

	if params.DryRun {
		return result, nil
	}
	if err := tx.Commit(ctx); err != nil {
		return ImportFinAllResult{}, err
	}
	return result, nil
}

func (s *Service) runFinAllImportJob(jobID int64, params ImportFinAllParams) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	if err := s.markJobRunning(ctx, params.TeamID, jobID); err != nil {
		return
	}
	result, err := s.ImportFinAll(ctx, params)
	if err != nil {
		_ = s.markJobFailed(ctx, params.TeamID, jobID, err)
		return
	}
	_ = s.markJobSucceeded(ctx, params.TeamID, jobID, result)
}

func (s *Service) createJob(ctx context.Context, params ImportFinAllParams, sourceHash string) (FinAllImportJob, error) {
	var job FinAllImportJob
	var resultJSON []byte
	err := s.db.QueryRow(ctx, `
		INSERT INTO debug_fin_all_import_jobs (team_id, actor_id, file_name, source_hash, dry_run, status)
		VALUES ($1, $2, $3, $4, $5, 'queued')
		RETURNING id, team_id, actor_id, file_name, source_hash, dry_run, status, result_json, error_message, created_at, started_at, finished_at
	`, params.TeamID, params.ActorID, params.FileName, sourceHash, params.DryRun).Scan(
		&job.ID,
		&job.TeamID,
		&job.ActorID,
		&job.FileName,
		&job.SourceHash,
		&job.DryRun,
		&job.Status,
		&resultJSON,
		&job.ErrorMessage,
		&job.CreatedAt,
		&job.StartedAt,
		&job.FinishedAt,
	)
	if err != nil {
		return FinAllImportJob{}, err
	}
	if err := decodeJobResult(&job, resultJSON); err != nil {
		return FinAllImportJob{}, err
	}
	return job, nil
}

func (s *Service) getJob(ctx context.Context, teamID int64, jobID int64) (FinAllImportJob, error) {
	var job FinAllImportJob
	var resultJSON []byte
	err := s.db.QueryRow(ctx, `
		SELECT id, team_id, actor_id, file_name, source_hash, dry_run, status, result_json, error_message, created_at, started_at, finished_at
		FROM debug_fin_all_import_jobs
		WHERE team_id = $1 AND id = $2
	`, teamID, jobID).Scan(
		&job.ID,
		&job.TeamID,
		&job.ActorID,
		&job.FileName,
		&job.SourceHash,
		&job.DryRun,
		&job.Status,
		&resultJSON,
		&job.ErrorMessage,
		&job.CreatedAt,
		&job.StartedAt,
		&job.FinishedAt,
	)
	if err != nil {
		return FinAllImportJob{}, err
	}
	if err := decodeJobResult(&job, resultJSON); err != nil {
		return FinAllImportJob{}, err
	}
	return job, nil
}

func (s *Service) markJobRunning(ctx context.Context, teamID int64, jobID int64) error {
	_, err := s.db.Exec(ctx, `
		UPDATE debug_fin_all_import_jobs
		SET status = 'running', started_at = COALESCE(started_at, now())
		WHERE team_id = $1 AND id = $2 AND status = 'queued'
	`, teamID, jobID)
	return err
}

func (s *Service) markJobSucceeded(ctx context.Context, teamID int64, jobID int64, result ImportFinAllResult) error {
	payload, err := json.Marshal(result)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(ctx, `
		UPDATE debug_fin_all_import_jobs
		SET status = 'succeeded', result_json = $3, error_message = NULL, finished_at = now()
		WHERE team_id = $1 AND id = $2
	`, teamID, jobID, payload)
	return err
}

func (s *Service) markJobFailed(ctx context.Context, teamID int64, jobID int64, jobErr error) error {
	_, err := s.db.Exec(ctx, `
		UPDATE debug_fin_all_import_jobs
		SET status = 'failed', error_message = $3, finished_at = now()
		WHERE team_id = $1 AND id = $2
	`, teamID, jobID, userFacingJobError(jobErr))
	return err
}

func decodeJobResult(job *FinAllImportJob, resultJSON []byte) error {
	if len(resultJSON) == 0 {
		return nil
	}
	var result ImportFinAllResult
	if err := json.Unmarshal(resultJSON, &result); err != nil {
		return err
	}
	job.Result = &result
	return nil
}

func userFacingJobError(err error) string {
	switch {
	case errors.Is(err, ErrInvalidWorkbook):
		return "Файл не похож на XLSX книгу"
	case errors.Is(err, ErrFinAllNotFound):
		return "В книге нет листа Fin_ALL"
	case errors.Is(err, ErrInvalidInput):
		return "Некорректные параметры импорта"
	default:
		return err.Error()
	}
}

func (s *Service) loadState(ctx context.Context, tx pgx.Tx, teamID int64, sourceHash string, bankCode string) (applyState, error) {
	state := applyState{
		tradersByWorker:     map[string]int64{},
		requisitesByPhone:   map[string]int64{},
		shiftsByTraderDate:  map[string]int64{},
		shiftTotals:         map[int64]*shiftTotals{},
		existingSourceItems: map[string]struct{}{},
	}

	traderRows, err := tx.Query(ctx, `
		SELECT u.id, p.external_worker_name
		FROM users u
		JOIN trader_profiles p ON p.user_id = u.id
		WHERE u.team_id = $1 AND u.role = 'trader' AND u.deleted_at IS NULL
	`, teamID)
	if err != nil {
		return state, err
	}
	defer traderRows.Close()
	for traderRows.Next() {
		var id int64
		var worker string
		if err := traderRows.Scan(&id, &worker); err != nil {
			return state, err
		}
		state.tradersByWorker[worker] = id
	}
	if err := traderRows.Err(); err != nil {
		return state, err
	}

	requisiteRows, err := tx.Query(ctx, `
		SELECT id, phone
		FROM requisites
		WHERE team_id = $1 AND bank_code = $2 AND deleted_at IS NULL
	`, teamID, bankCode)
	if err != nil {
		return state, err
	}
	defer requisiteRows.Close()
	for requisiteRows.Next() {
		var id int64
		var phone string
		if err := requisiteRows.Scan(&id, &phone); err != nil {
			return state, err
		}
		state.requisitesByPhone[phone] = id
	}
	if err := requisiteRows.Err(); err != nil {
		return state, err
	}

	itemRows, err := tx.Query(ctx, `
		SELECT source_row, source_circle
		FROM debug_fin_all_import_items
		WHERE team_id = $1 AND source_hash = $2 AND source_sheet = $3
	`, teamID, sourceHash, finAllSheetName)
	if err != nil {
		return state, err
	}
	defer itemRows.Close()
	for itemRows.Next() {
		var row int64
		var circle int
		if err := itemRows.Scan(&row, &circle); err != nil {
			return state, err
		}
		state.existingSourceItems[sourceItemKey(row, circle)] = struct{}{}
	}
	return state, itemRows.Err()
}

func (s *Service) ensureTrader(ctx context.Context, tx pgx.Tx, teamID int64, actorID int64, workerName string, state applyState) (int64, bool, error) {
	if id, ok := state.tradersByWorker[workerName]; ok {
		return id, false, nil
	}

	password, err := randomPassword()
	if err != nil {
		return 0, false, err
	}
	passwordHash, err := s.hashPassword(password)
	if err != nil {
		return 0, false, err
	}
	loginBase := normalizeLogin(workerName)
	if loginBase == "" {
		loginBase = "trader"
	}

	var userID int64
	for attempt := 0; attempt < 100; attempt++ {
		login := loginBase
		if attempt > 0 {
			login = fmt.Sprintf("%s_%d", loginBase, attempt+1)
		}
		err = tx.QueryRow(ctx, `
			INSERT INTO users (team_id, role, login, password_hash, status)
			VALUES ($1, 'trader', $2, $3, 'active')
			ON CONFLICT (login) DO NOTHING
			RETURNING id
		`, teamID, login, passwordHash).Scan(&userID)
		if err == nil {
			break
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return 0, false, err
		}
	}
	if userID == 0 {
		return 0, false, fmt.Errorf("cannot generate unique trader login for %s", workerName)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO trader_profiles (user_id, salary_rate_bps, external_worker_name)
		VALUES ($1, 0, $2)
	`, userID, workerName); err != nil {
		return 0, false, err
	}
	state.tradersByWorker[workerName] = userID
	if err := s.writeAudit(ctx, tx, teamID, actorID, "user.created", "user", userID, map[string]any{
		"id":                 userID,
		"externalWorkerName": workerName,
		"source":             "fin_all_debug_import",
	}, nil); err != nil {
		return 0, false, err
	}
	return userID, true, nil
}

func (s *Service) ensureRequisite(ctx context.Context, tx pgx.Tx, teamID int64, actorID int64, bankCode string, row finAllRow, state applyState) (int64, bool, error) {
	if id, ok := state.requisitesByPhone[row.Phone]; ok {
		if err := s.updateRequisiteDetails(ctx, tx, teamID, actorID, id, row); err != nil {
			return 0, false, err
		}
		return id, false, nil
	}

	var requisiteID int64
	err := tx.QueryRow(ctx, `
		INSERT INTO requisites (
			team_id, phone, method_type, bank_code, status,
			holder_name, card_number, details_filled_at, details_filled_by,
			employee_comment, created_by
		)
		VALUES ($1, $2, $3, $4, 'active', $5, $6, now(), $7, $8, $7)
		RETURNING id
	`, teamID, row.Phone, defaultMethodType, bankCode, row.Holder, row.Card, actorID, "Создано debug-импортом Fin_ALL").Scan(&requisiteID)
	if err != nil {
		return 0, false, err
	}
	state.requisitesByPhone[row.Phone] = requisiteID
	if err := s.writeAudit(ctx, tx, teamID, actorID, "requisite.created", "requisite", requisiteID, map[string]any{
		"id":         requisiteID,
		"phone":      row.Phone,
		"bankCode":   bankCode,
		"cardNumber": row.Card,
		"holderName": row.Holder,
		"source":     "fin_all_debug_import",
	}, nil); err != nil {
		return 0, false, err
	}
	return requisiteID, true, nil
}

func (s *Service) updateRequisiteDetails(ctx context.Context, tx pgx.Tx, teamID int64, actorID int64, requisiteID int64, row finAllRow) error {
	tag, err := tx.Exec(ctx, `
		UPDATE requisites
		SET holder_name = $4,
			card_number = $5,
			details_filled_at = COALESCE(details_filled_at, now()),
			details_filled_by = COALESCE(details_filled_by, $3),
			updated_at = now()
		WHERE team_id = $1
			AND id = $2
			AND (
				holder_name IS DISTINCT FROM $4
				OR card_number IS DISTINCT FROM $5
				OR details_filled_at IS NULL
				OR details_filled_by IS NULL
			)
	`, teamID, requisiteID, actorID, row.Holder, row.Card)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return nil
	}
	return s.writeAudit(ctx, tx, teamID, actorID, "requisite.updated", "requisite", requisiteID, map[string]any{
		"id":         requisiteID,
		"cardNumber": row.Card,
		"holderName": row.Holder,
		"source":     "fin_all_debug_import",
	}, nil)
}

func (s *Service) ensureHistoricalShift(ctx context.Context, tx pgx.Tx, teamID int64, traderID int64, shiftDate time.Time, state applyState) (int64, bool, error) {
	key := fmt.Sprintf("%d:%s", traderID, shiftDate.Format("2006-01-02"))
	if id, ok := state.shiftsByTraderDate[key]; ok {
		return id, false, nil
	}

	startedAt := time.Date(shiftDate.Year(), shiftDate.Month(), shiftDate.Day(), 9, 0, 0, 0, s.location)
	closedAt := time.Date(shiftDate.Year(), shiftDate.Month(), shiftDate.Day(), 23, 59, 0, 0, s.location)
	var shiftID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO trader_shifts (
			team_id, trader_id, started_at, ended_at, status,
			inbound_reconciliation_status, outbound_reconciliation_status,
			close_comment, closed_at
		)
		VALUES ($1, $2, $3, $4, 'closed_with_discrepancy', 'accepted_with_comment', 'accepted_with_comment', $5, $4)
		RETURNING id
	`, teamID, traderID, startedAt, closedAt, importComment).Scan(&shiftID); err != nil {
		return 0, false, err
	}
	state.shiftsByTraderDate[key] = shiftID
	return shiftID, true, nil
}

func (s *Service) createClosedAssignment(ctx context.Context, tx pgx.Tx, teamID int64, actorID int64, traderID int64, requisiteID int64, circle finAllCircle) (int64, error) {
	status := "worked"
	if circle.Blocked {
		status = "blocked"
	}
	assignedAt := time.Date(circle.Date.Year(), circle.Date.Month(), circle.Date.Day(), 8, 59, 0, 0, s.location)
	completedAt := time.Date(circle.Date.Year(), circle.Date.Month(), circle.Date.Day(), 23, 58, 0, 0, s.location)
	var assignmentID int64
	err := tx.QueryRow(ctx, `
		INSERT INTO requisite_assignments (
			team_id, requisite_id, trader_id, assigned_by, assigned_at, unassigned_at, comment,
			status, assigned_for_date, target_turnover_minor, started_at, completed_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $5, $6, now())
		RETURNING id
	`, teamID, requisiteID, traderID, actorID, assignedAt, completedAt, "Debug Fin_ALL import", status, circle.Date, circle.InboundTurnoverMinor).Scan(&assignmentID)
	return assignmentID, err
}

func (s *Service) createClosedShiftRequisite(ctx context.Context, tx pgx.Tx, teamID int64, traderID int64, requisiteID int64, assignmentID int64, shiftID int64, row finAllRow, circle finAllCircle) (int64, error) {
	status := "worked_discrepancy"
	if circle.Blocked {
		status = "blocked"
	}
	takenAt := time.Date(circle.Date.Year(), circle.Date.Month(), circle.Date.Day(), 9, 0, 0, 0, s.location)
	releasedAt := time.Date(circle.Date.Year(), circle.Date.Month(), circle.Date.Day(), 23, 58, 0, 0, s.location)
	var shiftRequisiteID int64
	err := tx.QueryRow(ctx, `
		INSERT INTO shift_requisites (
			team_id, shift_id, trader_id, requisite_id, assignment_id,
			card_number, holder_name, taken_at, released_at, status,
			inbound_turnover_minor, outbound_turnover_minor, closing_balance_minor
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id
	`, teamID, shiftID, traderID, requisiteID, assignmentID, row.Card, row.Holder, takenAt, releasedAt, status, circle.InboundTurnoverMinor, circle.OutboundTurnoverMinor, circle.ClosingBalanceMinor).Scan(&shiftRequisiteID)
	return shiftRequisiteID, err
}

func (s *Service) linkAssignmentToShiftRequisite(ctx context.Context, tx pgx.Tx, teamID int64, assignmentID int64, shiftRequisiteID int64, circle finAllCircle) error {
	status := "worked"
	if circle.Blocked {
		status = "blocked"
	}
	_, err := tx.Exec(ctx, `
		UPDATE requisite_assignments
		SET shift_requisite_id = $1, status = $2, updated_at = now()
		WHERE team_id = $3 AND id = $4
	`, shiftRequisiteID, status, teamID, assignmentID)
	return err
}

func (s *Service) addShiftTotals(state applyState, shiftID int64, traderID int64, circle finAllCircle) {
	totals := state.shiftTotals[shiftID]
	if totals == nil {
		totals = &shiftTotals{TraderID: traderID}
		state.shiftTotals[shiftID] = totals
	}
	totals.InboundMinor += circle.InboundTurnoverMinor
	totals.OutboundMinor += circle.OutboundTurnoverMinor
	if circle.InboundTurnoverMinor > 0 {
		totals.InboundCount++
	}
	if circle.OutboundTurnoverMinor > 0 {
		totals.OutboundCount++
	}
}

func (s *Service) insertAcceptedReconciliations(ctx context.Context, tx pgx.Tx, teamID int64, actorID int64, shiftID int64, totals shiftTotals) error {
	for _, item := range []struct {
		reconciliationType string
		actualMinor        int64
		successCount       int64
	}{
		{reconciliationType: "trader_shift_inbound", actualMinor: totals.InboundMinor, successCount: totals.InboundCount},
		{reconciliationType: "trader_shift_outbound", actualMinor: totals.OutboundMinor, successCount: totals.OutboundCount},
	} {
		_, err := tx.Exec(ctx, `
			INSERT INTO reconciliation_runs (
				team_id, type, scope_type, shift_id, accounting_period_id, trader_id, import_batch_id,
				expected_amount_minor, actual_amount_minor, diff_amount_minor,
				success_amount_minor, success_count, failed_amount_minor, failed_count,
				total_amount_minor, total_count, status, comment, confirmed_by, confirmed_at
			)
			VALUES ($1, $2, 'trader_shift', $3, NULL, $4, NULL, 0, $5, $5, $5, $6, 0, 0, $5, $6, 'accepted_with_comment', $7, $8, now())
		`, teamID, item.reconciliationType, shiftID, totals.TraderID, item.actualMinor, item.successCount, importComment, actorID)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) markRequisiteBlocked(ctx context.Context, tx pgx.Tx, teamID int64, requisiteID int64) error {
	_, err := tx.Exec(ctx, `
		UPDATE requisites
		SET status = 'blocked', updated_at = now()
		WHERE team_id = $1 AND id = $2
	`, teamID, requisiteID)
	return err
}

func (s *Service) rememberSourceItem(ctx context.Context, tx pgx.Tx, teamID int64, sourceHash string, rowNumber int64, circleNumber int, actorID int64, traderID int64, requisiteID int64, assignmentID int64, shiftID int64, shiftRequisiteID int64) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO debug_fin_all_import_items (
			team_id, source_hash, source_sheet, source_row, source_circle,
			trader_id, requisite_id, assignment_id, shift_id, shift_requisite_id, created_by
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, teamID, sourceHash, finAllSheetName, rowNumber, circleNumber, traderID, requisiteID, assignmentID, shiftID, shiftRequisiteID, actorID)
	return err
}

func (s *Service) writeSummaryAudit(ctx context.Context, tx pgx.Tx, teamID int64, actorID int64, result ImportFinAllResult) error {
	action := "debug.fin_all_import_applied"
	if result.DryRun {
		action = "debug.fin_all_import_dry_run"
	}
	return s.writeAudit(ctx, tx, teamID, actorID, action, "debug_fin_all_import", 0, result, nil)
}

func (s *Service) writeAudit(ctx context.Context, tx pgx.Tx, teamID int64, actorID int64, action string, entityType string, entityID int64, after any, comment *string) error {
	payload, err := json.Marshal(after)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO audit_logs (team_id, actor_id, action, entity_type, entity_id, after_json, comment)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, teamID, actorID, action, entityType, fmt.Sprint(entityID), payload, comment)
	return err
}

func hashBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func sourceItemKey(row int64, circle int) string {
	return fmt.Sprintf("%d:%d", row, circle)
}

func randomPassword() (string, error) {
	var data [18]byte
	if _, err := rand.Read(data[:]); err != nil {
		return "", err
	}
	return "fin-all-" + hex.EncodeToString(data[:]), nil
}

func normalizeLogin(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	lastUnderscore := false
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			builder.WriteRune(r)
			lastUnderscore = false
			continue
		}
		if !lastUnderscore {
			builder.WriteRune('_')
			lastUnderscore = true
		}
	}
	return strings.Trim(builder.String(), "_")
}
