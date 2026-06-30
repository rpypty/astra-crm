package httpserver

import (
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"github.com/ashpak/astra-crm-backend/internal/debugimport"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

type TeamleadDebugService interface {
	StartFinAllImport(ctx context.Context, params debugimport.ImportFinAllParams) (debugimport.FinAllImportJob, error)
	GetFinAllImportJob(ctx context.Context, teamID int64, jobID int64) (debugimport.FinAllImportJob, error)
}

type TeamleadDebugHandler struct {
	service TeamleadDebugService
}

func NewTeamleadDebugHandler(service TeamleadDebugService) *TeamleadDebugHandler {
	return &TeamleadDebugHandler{service: service}
}

type debugFinAllImportJobResponse struct {
	Job debugimport.FinAllImportJob `json:"job"`
}

func (h *TeamleadDebugHandler) ImportFinAll(w http.ResponseWriter, r *http.Request) {
	actor, ok := CurrentUser(r.Context())
	if !ok {
		RespondError(w, UnauthorizedError())
		return
	}
	if h.service == nil {
		RespondError(w, ServiceUnavailableError())
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxImportFileSize)
	if err := r.ParseMultipartForm(maxImportFileSize); err != nil {
		RespondError(w, multipartFormError(err))
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		RespondError(w, ValidationError(map[string]string{
			"file": "Выберите XLSX файл",
		}))
		return
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, maxImportFileSize+1))
	if err != nil {
		RespondError(w, ValidationError(map[string]string{
			"file": "Файл слишком большой или поврежден",
		}))
		return
	}
	if int64(len(data)) > maxImportFileSize {
		RespondError(w, ValidationError(map[string]string{
			"file": "Файл слишком большой",
		}))
		return
	}
	bankCode := strings.TrimSpace(r.FormValue("bankCode"))
	if bankCode == "" {
		RespondError(w, ValidationError(map[string]string{
			"bankCode": "Выберите банк реквизитов",
		}))
		return
	}

	job, err := h.service.StartFinAllImport(r.Context(), debugimport.ImportFinAllParams{
		ActorID:  actor.ID,
		TeamID:   actor.TeamID,
		FileName: header.Filename,
		Data:     data,
		BankCode: bankCode,
		DryRun:   r.FormValue("dryRun") == "true",
	})
	if err != nil {
		RespondError(w, mapDebugImportError(err))
		return
	}

	WriteJSON(w, http.StatusAccepted, debugFinAllImportJobResponse{Job: job})
}

func (h *TeamleadDebugHandler) FinAllImportJob(w http.ResponseWriter, r *http.Request) {
	actor, ok := CurrentUser(r.Context())
	if !ok {
		RespondError(w, UnauthorizedError())
		return
	}
	if h.service == nil {
		RespondError(w, ServiceUnavailableError())
		return
	}
	jobID, err := strconv.ParseInt(chi.URLParam(r, "jobId"), 10, 64)
	if err != nil || jobID <= 0 {
		RespondError(w, NotFoundError())
		return
	}
	job, err := h.service.GetFinAllImportJob(r.Context(), actor.TeamID, jobID)
	if err != nil {
		RespondError(w, mapDebugImportError(err))
		return
	}

	WriteJSON(w, http.StatusOK, debugFinAllImportJobResponse{Job: job})
}

func multipartFormError(err error) *APIError {
	message := "Не удалось прочитать файл из запроса"
	details := []string{err.Error()}
	switch {
	case errors.Is(err, multipart.ErrMessageTooLarge), strings.Contains(err.Error(), "request body too large"):
		message = "Файл слишком большой. Максимальный размер: 32 МБ"
	case strings.Contains(err.Error(), "i/o timeout"):
		message = "Файл не успел загрузиться до таймаута сервера. Попробуйте еще раз; если повторится, проверьте скорость соединения или уменьшите файл"
	case strings.Contains(err.Error(), "request Content-Type isn't multipart/form-data"):
		message = "Запрос отправлен не как multipart/form-data. Обновите страницу и выберите файл заново"
	case errors.Is(err, io.EOF), strings.Contains(err.Error(), "unexpected EOF"):
		message = "Загрузка файла оборвалась. Попробуйте отправить файл еще раз"
	}
	return &APIError{
		Status:  http.StatusBadRequest,
		Code:    CodeValidation,
		Message: "Некоторые поля заполнены неверно",
		Fields: map[string]string{
			"file": message,
		},
		Details: details,
	}
}

func mapDebugImportError(err error) error {
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return NotFoundError()
	case errors.Is(err, debugimport.ErrInvalidInput):
		return ValidationError(map[string]string{
			"file": "Некорректный файл импорта",
		}).WithCause(err)
	case errors.Is(err, debugimport.ErrInvalidWorkbook):
		return ValidationError(map[string]string{
			"file": "Файл не похож на XLSX книгу",
		}).WithCause(err)
	case errors.Is(err, debugimport.ErrFinAllNotFound):
		return ValidationError(map[string]string{
			"file": "В книге нет листа Fin_ALL",
		}).WithCause(err)
	default:
		return err
	}
}
