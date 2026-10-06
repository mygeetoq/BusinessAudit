package service

import "testing"

func TestParseBankStatementRussianSemicolon(t *testing.T) {
	raw := []byte("\ufeffдата;сумма;тип\n01.10.2026;\"1 500,50\";приход\n02.10.2026;200,00;возврат\n")
	txs, err := ParseBankStatement(raw, "MY-ACCOUNT")
	if err != nil { t.Fatal(err) }
	if len(txs) != 2 || txs[0].AccountID != "MY-ACCOUNT" || txs[0].Amount.String() != "1500.5" || txs[1].Type != "return" {
		t.Fatalf("unexpected transactions: %#v", txs)
	}
}

func TestParseBankStatementRejectsInvalidRow(t *testing.T) {
	for _, raw := range []string{
		"date,amount,type\n2026-10-01,-10,income\n",
		"date,amount,type\n2026-10-01,10,expense\n",
		"date,amount,type\n2026-10-01,10,income\nwrong,20,income\n",
		"date,amount\n2026-10-01,10\n",
	} {
		if _, err := ParseBankStatement([]byte(raw), "ACCOUNT"); err == nil { t.Errorf("accepted invalid CSV %q", raw) }
	}
}
