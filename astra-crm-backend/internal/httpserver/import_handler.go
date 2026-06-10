package httpserver

import (
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"github.com/ashpak/astra-crm-backend/internal/imports"
	"github.com/ashpak/astra-crm-backend/internal/shifts"
)

const maxImportFileSize = 32 << 20

type ImportService interface {
	ApplyCSV(ctx context.Context, params imports.ApplyCSVParams) (imports.ApplyResult, error)
}

type ImportShiftService interface {
	Current(ctx context.Context, teamID int64, traderID int64) (*shifts.Shift, error)
}

type ImportHandler struct {
	importService ImportService
	shiftService  ImportShiftService
}

func NewImportHandler(importService ImportService, shiftService ImportShiftService) *ImportHandler {
	return &ImportHandler{
		importService: importService,
		shiftService:  shiftService,
	}
}

type importResponse struct {
	Result publicImportResult `json:"result"`
}

type publicImportResult struct {
	ImportBatchID          int64          `json:"importBatchId"`
	Status                 string         `json:"status"`
	RowsCount              int64          `json:"rowsCount"`
	CreatedOrders          int64          `json:"createdOrders"`
	UpdatedOrders          int64          `json:"updatedOrders"`
	DeactivatedScopeItems  int64          `json:"deactivatedScopeItems"`
	ActiveScopeItems       int64          `json:"activeScopeItems"`
	SupersededBatches      int64          `json:"supersededBatches"`
	RawStatusCounts        map[string]int `json:"rawStatusCounts"`
	NormalizedStatusCounts map[string]int `json:"normalizedStatusCounts"`
	UnknownStatuses        []string       `json:"unknownStatuses"`
}

func (h *ImportHandler) TraderInbound(w http.ResponseWriter, r *http.Request) {
	h.applyTraderImport(w, r, imports.DirectionInbound)
}

func (h *ImportHandler) TraderOutbound(w http.ResponseWriter, r *http.Request) {
	h.applyTraderImport(w, r, imports.DirectionOutbound)
}

func (h *ImportHandler) TeamleadInbound(w http.ResponseWriter, r *http.Request) {
	h.applyTeamleadImport(w, r, imports.DirectionInbound)
}

func (h *ImportHandler) TeamleadOutbound(w http.ResponseWriter, r *http.Request) {
	h.applyTeamleadImport(w, r, imports.DirectionOutbound)
}

func (h *ImportHandler) applyTraderImport(w http.ResponseWriter, r *http.Request, direction string) {
	actor, ok := CurrentUser(r.Context())
	if !ok {
		RespondError(w, UnauthorizedError())
		return
	}
	if h.importService == nil || h.shiftService == nil {
		RespondError(w, ServiceUnavailableError())
		return
	}

	shift, err := h.shiftService.Current(r.Context(), actor.TeamID, actor.ID)
	if err != nil {
		RespondError(w, mapShiftError(err))
		return
	}
	if shift == nil {
		RespondError(w, DomainError("CURRENT_SHIFT_REQUIRED", "Нужна открытая смена для импорта CSV"))
		return
	}

	fileName, reader, ok := uploadFileFromRequest(w, r)
	if !ok {
		return
	}
	defer reader.Close()

	result, err := h.importService.ApplyCSV(r.Context(), imports.ApplyCSVParams{
		ActorID:  actor.ID,
		TeamID:   actor.TeamID,
		FileName: fileName,
		Scope: imports.Scope{
			Type:      imports.ScopeTypeTraderShift,
			Direction: direction,
			ShiftID:   &shift.ID,
			TraderID:  &actor.ID,
		},
		Reader: reader,
		ParseOptions: imports.ParseOptions{
			ColumnSet: imports.ColumnSetTrader,
		},
	})
	if err != nil {
		RespondError(w, mapImportError(err, result.Parse))
		return
	}

	WriteJSON(w, http.StatusCreated, importResponse{
		Result: publicImportResultFromDomain(result),
	})
}

func (h *ImportHandler) applyTeamleadImport(w http.ResponseWriter, r *http.Request, direction string) {
	actor, ok := CurrentUser(r.Context())
	if !ok {
		RespondError(w, UnauthorizedError())
		return
	}
	if h.importService == nil {
		RespondError(w, ServiceUnavailableError())
		return
	}

	fileName, reader, ok := uploadFileFromRequest(w, r)
	if !ok {
		return
	}
	defer reader.Close()

	accountingPeriodID, ok := accountingPeriodIDFromRequest(w, r)
	if !ok {
		return
	}

	result, err := h.importService.ApplyCSV(r.Context(), imports.ApplyCSVParams{
		ActorID:  actor.ID,
		TeamID:   actor.TeamID,
		FileName: fileName,
		Scope: imports.Scope{
			Type:               imports.ScopeTypeTeamleadPeriod,
			Direction:          direction,
			AccountingPeriodID: &accountingPeriodID,
		},
		Reader: reader,
		ParseOptions: imports.ParseOptions{
			ColumnSet: imports.ColumnSetTeamlead,
		},
	})
	if err != nil {
		RespondError(w, mapImportError(err, result.Parse))
		return
	}

	WriteJSON(w, http.StatusCreated, importResponse{
		Result: publicImportResultFromDomain(result),
	})
}

func uploadFileFromRequest(w http.ResponseWriter, r *http.Request) (string, multipart.File, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, maxImportFileSize)
	if err := r.ParseMultipartForm(maxImportFileSize); err != nil {
		RespondError(w, ValidationError(map[string]string{
			"file": "CSV файл обязателен и должен быть multipart/form-data",
		}))
		return "", nil, false
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		RespondError(w, ValidationError(map[string]string{
			"file": "CSV файл обязателен",
		}))
		return "", nil, false
	}

	fileName := strings.TrimSpace(header.Filename)
	if fileName == "" {
		fileName = "import.csv"
	}

	return fileName, file, true
}

func accountingPeriodIDFromRequest(w http.ResponseWriter, r *http.Request) (int64, bool) {
	raw := strings.TrimSpace(r.URL.Query().Get("accountingPeriodId"))
	if raw == "" {
		if err := r.ParseMultipartForm(maxImportFileSize); err == nil {
			raw = strings.TrimSpace(r.FormValue("accountingPeriodId"))
		}
	}

	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		RespondError(w, ValidationError(map[string]string{
			"accountingPeriodId": "Некорректный ID периода",
		}))
		return 0, false
	}

	return id, true
}

func publicImportResultFromDomain(result imports.ApplyResult) publicImportResult {
	return publicImportResult{
		ImportBatchID:          result.Batch.ID,
		Status:                 result.Batch.Status,
		RowsCount:              result.RowsCount,
		CreatedOrders:          result.CreatedOrders,
		UpdatedOrders:          result.UpdatedOrders,
		DeactivatedScopeItems:  result.DeactivatedScopeItems,
		ActiveScopeItems:       result.ActiveScopeItems,
		SupersededBatches:      result.SupersededBatches,
		RawStatusCounts:        result.Parse.RawStatusCounts,
		NormalizedStatusCounts: result.Parse.NormalizedStatusCounts,
		UnknownStatuses:        unknownStatuses(result.Parse),
	}
}

func unknownStatuses(parse imports.ParseResult) []string {
	seen := map[string]bool{}
	statuses := make([]string, 0)
	for _, row := range parse.Rows {
		if row.NormalizedStatus != imports.NormalizedStatusUnknown || seen[row.RawStatus] {
			continue
		}
		seen[row.RawStatus] = true
		statuses = append(statuses, row.RawStatus)
	}

	return statuses
}

func mapImportError(err error, parse imports.ParseResult) error {
	switch {
	case errors.Is(err, imports.ErrInvalidInput):
		return ValidationError(map[string]string{
			"body": "Некоторые поля заполнены неверно",
		})
	case errors.Is(err, imports.ErrRepositoryNotConfigured):
		return ServiceUnavailableError()
	case errors.Is(err, imports.ErrValidation):
		return csvValidationError(parse)
	default:
		return err
	}
}

func csvValidationError(parse imports.ParseResult) *APIError {
	code := "CSV_VALIDATION_ERROR"
	if len(parse.Errors) > 0 && parse.Errors[0].Code != "" {
		code = parse.Errors[0].Code
	}

	return &APIError{
		Status:  http.StatusBadRequest,
		Code:    code,
		Message: "CSV файл содержит ошибки",
		Details: parseErrorDetails(parse.Errors),
	}
}

func parseErrorDetails(errorsList []imports.ParseError) []string {
	details := make([]string, 0, len(errorsList))
	for _, parseError := range errorsList {
		details = append(details, formatParseError(parseError))
	}

	return details
}

func formatParseError(parseError imports.ParseError) string {
	parts := []string{parseError.Code}
	if parseError.Row > 0 {
		parts = append(parts, fmt.Sprintf("row=%d", parseError.Row))
	}
	if parseError.Field != "" {
		parts = append(parts, "field="+parseError.Field)
	}
	if parseError.InnerID != "" {
		parts = append(parts, "innerId="+parseError.InnerID)
	}
	if len(parseError.Rows) > 0 {
		parts = append(parts, fmt.Sprintf("rows=%v", parseError.Rows))
	}
	if parseError.Message != "" {
		parts = append(parts, parseError.Message)
	}

	return strings.Join(parts, " ")
}
