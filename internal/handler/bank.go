package handler

import (
	"errors"
	"net/http"
	"strings"
	"unicode"

	"business-pulse/internal/auth"
	"business-pulse/internal/repository"
	"business-pulse/internal/service"
)

// GET /api/my/bank-account — подключённый расчётный счёт.
func (d Deps) bankAccount(w http.ResponseWriter, r *http.Request) {
	p, err := d.Repo.GetParticipantByID(r.Context(), auth.ParticipantID(r.Context()))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "не удалось загрузить счёт")
		return
	}
	accountID := p.AccountID
	if p.BankName == "" {
		accountID = ""
	}
	writeJSON(w, http.StatusOK, map[string]string{"bank_name": p.BankName, "account_id": accountID, "mode": "manual_csv"})
}

// PUT /api/my/bank-account — сохранение реквизитов расчётного счёта.
// Это не подтверждение владения счётом банком и не выдача доступа к банковскому API.
func (d Deps) connectBankAccount(w http.ResponseWriter, r *http.Request) {
	var req struct {
		BankName  string `json:"bank_name"`
		AccountID string `json:"account_id"`
	}
	if !decodeBody(w, r, &req) {
		return
	}
	req.BankName = strings.TrimSpace(req.BankName)
	req.AccountID = strings.TrimSpace(req.AccountID)
	if len([]rune(req.BankName)) < 2 || len([]rune(req.BankName)) > 120 {
		writeErr(w, http.StatusBadRequest, "название банка: от 2 до 120 символов")
		return
	}
	if len(req.AccountID) < 5 || len(req.AccountID) > 50 || strings.IndexFunc(req.AccountID, func(c rune) bool { return !unicode.IsLetter(c) && !unicode.IsDigit(c) && c != '-' }) >= 0 {
		writeErr(w, http.StatusBadRequest, "идентификатор счёта: 5–50 букв, цифр или дефисов")
		return
	}
	err := d.Repo.ConnectBankAccount(r.Context(), auth.ParticipantID(r.Context()), req.BankName, req.AccountID)
	switch {
	case errors.Is(err, repository.ErrAccountHasData), errors.Is(err, repository.ErrAccountInUse):
		writeErr(w, http.StatusConflict, err.Error())
	case err != nil:
		d.Log.Error("подключение счёта", "err", err)
		writeErr(w, http.StatusInternalServerError, "не удалось подключить счёт")
	default:
		d.bankAccount(w, r)
	}
}

// POST /api/my/bank-statement — выписка date,amount,type; профиль определяется сессией.
func (d Deps) importBankStatement(w http.ResponseWriter, r *http.Request) {
	pid := auth.ParticipantID(r.Context())
	p, err := d.Repo.GetParticipantByID(r.Context(), pid)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "не удалось загрузить счёт")
		return
	}
	body, ok := csvBody(w, r)
	if !ok {
		return
	}
	defer body.Close()
	report, err := d.Svc.ImportBankStatement(r.Context(), pid, p.AccountID, body)
	switch {
	case errors.Is(err, service.ErrDuplicateImport):
		writeErr(w, http.StatusConflict, "эта выписка уже загружена")
		return
	case err != nil:
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if p.GroupType == "B" {
		if res, err := d.Svc.Recalculate(r.Context(), pid); err == nil {
			d.notifyAfterRecalc(r, pid, res)
		} else {
			d.Log.Warn("пересчёт после выписки", "err", err)
		}
	}
	writeJSON(w, http.StatusOK, report)
}
