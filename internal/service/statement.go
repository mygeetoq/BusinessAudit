package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ParseBankStatement принимает единый формат, который можно подготовить из
// выписки любого банка. account_id берётся из сессии, а не из файла.
// Поддерживаются CSV с запятой/точкой с запятой и русские/английские заголовки.
func ParseBankStatement(raw []byte, accountID string) ([]ParsedTx, error) {
	if len(raw) == 0 || len(raw) > maxCSVSize {
		return nil, errors.New("CSV должен быть непустым и не более 10 МБ")
	}
	r := csv.NewReader(bytes.NewReader(bytes.TrimPrefix(raw, []byte{0xef, 0xbb, 0xbf})))
	first := string(bytes.SplitN(raw, []byte{'\n'}, 2)[0])
	if strings.Count(first, ";") > strings.Count(first, ",") {
		r.Comma = ';'
	}
	r.FieldsPerRecord = -1
	r.TrimLeadingSpace = true
	head, err := r.Read()
	if err != nil {
		return nil, errors.New("не удалось прочитать заголовок CSV")
	}
	columns := map[string]int{}
	for i, h := range head {
		h = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(h, "\ufeff")))
		switch h {
		case "date", "дата", "дата операции":
			columns["date"] = i
		case "amount", "сумма", "сумма операции":
			columns["amount"] = i
		case "type", "тип", "тип операции":
			columns["type"] = i
		}
	}
	if len(columns) != 3 {
		return nil, errors.New("нужны заголовки date,amount,type (или дата,сумма,тип)")
	}
	var out []ParsedTx
	for line := 2; ; line++ {
		row, err := r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("строка %d: некорректный CSV", line)
		}
		value := func(key string) string {
			i := columns[key]
			if i >= len(row) {
				return ""
			}
			return strings.TrimSpace(row[i])
		}
		date, err := parseDate(value("date"))
		if err != nil {
			return nil, fmt.Errorf("строка %d: некорректная дата", line)
		}
		amount, err := decimal.NewFromString(strings.ReplaceAll(strings.ReplaceAll(value("amount"), " ", ""), ",", "."))
		if err != nil || !amount.IsPositive() {
			return nil, fmt.Errorf("строка %d: сумма должна быть положительной", line)
		}
		txType, err := parseTxType(value("type"))
		if err != nil {
			return nil, fmt.Errorf("строка %d: %w", line, err)
		}
		out = append(out, ParsedTx{AccountID: accountID, Date: date, Amount: amount, Type: txType})
	}
	if len(out) == 0 {
		return nil, errors.New("в CSV нет операций")
	}
	return out, nil
}

// ImportBankStatement проверяет дубль в пределах профиля и импортирует операции.
func (s *Service) ImportBankStatement(ctx context.Context, pid uuid.UUID, accountID string, body io.Reader) (*ImportReport, error) {
	raw, err := io.ReadAll(io.LimitReader(body, maxCSVSize+1))
	if err != nil {
		return nil, err
	}
	txs, err := ParseBankStatement(raw, accountID)
	if err != nil {
		return nil, err
	}
	hash := sha256.Sum256(append([]byte(pid.String()+":"), raw...))
	key := hex.EncodeToString(hash[:])
	dup, err := s.repo.HasImportBatch(ctx, key)
	if err != nil {
		return nil, err
	}
	if dup {
		return nil, ErrDuplicateImport
	}
	report := &ImportReport{Errors: []string{}}
	if err := s.ImportParsedTransactions(ctx, txs, report); err != nil {
		return nil, err
	}
	if err := s.repo.RecordImportBatch(ctx, key, report.Imported); err != nil {
		return report, err
	}
	return report, nil
}
