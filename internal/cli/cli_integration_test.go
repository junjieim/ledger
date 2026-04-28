package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runLedgerCommand(t *testing.T, dbPath string, args ...string) (string, string, error) {
	t.Helper()

	dbPath = filepath.Clean(dbPath)
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	outR, outW, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe stdout: %v", err)
	}
	errR, errW, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe stderr: %v", err)
	}

	os.Stdout = outW
	os.Stderr = errW
	defer func() {
		os.Stdout = oldStdout
		os.Stderr = oldStderr
	}()

	root := NewRootCmd()
	root.SetOut(outW)
	root.SetErr(errW)
	root.SetArgs(append([]string{"--db", dbPath}, args...))

	runErr := root.Execute()

	if database != nil {
		_ = database.Close()
		database = nil
	}
	_ = outW.Close()
	_ = errW.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if _, err := io.Copy(&stdout, outR); err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	if _, err := io.Copy(&stderr, errR); err != nil {
		t.Fatalf("read stderr: %v", err)
	}

	return stdout.String(), stderr.String(), runErr
}

func TestDefaultDBPathUsesHomeLedgerDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	want := filepath.Join(home, ".ledger", "ledger.db")
	if got := defaultDBPath(); got != want {
		t.Fatalf("default db path = %q, want %q", got, want)
	}

	root := NewRootCmd()
	root.SetArgs([]string{"init"})
	if err := root.Execute(); err != nil {
		t.Fatalf("init with default db path: %v", err)
	}
	if database != nil {
		_ = database.Close()
		database = nil
	}
	if _, err := os.Stat(want); err != nil {
		t.Fatalf("expected default db to be created at %q: %v", want, err)
	}
}

func TestSearchKeywordIntegration(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "ledger.db")

	if _, stderr, err := runLedgerCommand(t, dbPath, "init"); err != nil {
		t.Fatalf("init: %v", err)
	} else if stderr != "" {
		t.Fatalf("init should not warn about embeddings, got %q", stderr)
	}
	if _, _, err := runLedgerCommand(t, dbPath, "add",
		"--amount", "20",
		"--direction", "expense",
		"--category", "餐饮",
		"--description", "火锅",
	); err != nil {
		t.Fatalf("add: %v", err)
	}

	stdout, stderr, err := runLedgerCommand(t, dbPath, "--json", "search",
		"--keyword", "火锅",
	)
	if err != nil {
		t.Fatalf("keyword search: %v", err)
	}
	if stderr != "" {
		t.Fatalf("search should not warn, got %q", stderr)
	}
	var result struct {
		Items []struct {
			ID          string `json:"id"`
			Description string `json:"description"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("decode search result: %v", err)
	}
	if len(result.Items) != 1 || result.Items[0].Description != "火锅" || result.Items[0].ID == "" {
		t.Fatalf("unexpected search result: %#v", result.Items)
	}
}

func TestSearchKeywordRequired(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "ledger.db")
	if _, _, err := runLedgerCommand(t, dbPath, "init"); err != nil {
		t.Fatalf("init: %v", err)
	}

	_, stderr, err := runLedgerCommand(t, dbPath, "search")
	if err == nil {
		t.Fatal("expected missing keyword to fail")
	}
	if !strings.Contains(stderr, "--keyword required") {
		t.Fatalf("expected --keyword required error, got %q", stderr)
	}
}

func TestConfigCommandRemoved(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "ledger.db")

	_, stderr, err := runLedgerCommand(t, dbPath, "config", "show")
	if err == nil {
		t.Fatal("expected config command to be removed")
	}
	if !strings.Contains(stderr, "unknown command") {
		t.Fatalf("expected unknown command error, got %q", stderr)
	}
}

func TestSearchSemanticFlagRemoved(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "ledger.db")

	_, stderr, err := runLedgerCommand(t, dbPath, "search", "--semantic", "聚餐", "--keyword", "火锅")
	if err == nil {
		t.Fatal("expected semantic flag to be removed")
	}
	if !strings.Contains(stderr, "unknown flag: --semantic") {
		t.Fatalf("expected unknown semantic flag error, got %q", stderr)
	}
}

func TestRefundCLIPartialAndBalanceUsesNetAmount(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "ledger.db")
	if _, _, err := runLedgerCommand(t, dbPath, "init"); err != nil {
		t.Fatalf("init: %v", err)
	}

	stdout, _, err := runLedgerCommand(t, dbPath, "--json", "add",
		"--amount", "100",
		"--direction", "expense",
		"--currency", "CNY",
		"--category", "餐饮",
		"--description", "测试火锅",
	)
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	var added struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(stdout), &added); err != nil {
		t.Fatalf("decode add: %v", err)
	}

	stdout, _, err = runLedgerCommand(t, dbPath, "--json", "refund",
		"--id", added.ID,
		"--amount", "25",
		"--note", "店家漏单",
	)
	if err != nil {
		t.Fatalf("refund: %v", err)
	}
	var refunded struct {
		RefundAmount float64 `json:"refund_amount"`
		NetAmount    float64 `json:"net_amount"`
		Note         string  `json:"note"`
	}
	if err := json.Unmarshal([]byte(stdout), &refunded); err != nil {
		t.Fatalf("decode refund: %v", err)
	}
	if refunded.RefundAmount != 25 || refunded.NetAmount != 75 {
		t.Fatalf("unexpected refund fields: %+v", refunded)
	}
	if !strings.Contains(refunded.Note, "[退款 25 CNY] 店家漏单") {
		t.Fatalf("expected refund note, got %q", refunded.Note)
	}

	stdout, _, err = runLedgerCommand(t, dbPath, "--json", "balance", "--currency", "CNY")
	if err != nil {
		t.Fatalf("balance: %v", err)
	}
	var balance struct {
		Balances []struct {
			Currency string  `json:"currency"`
			Balance  float64 `json:"balance"`
		} `json:"balances"`
	}
	if err := json.Unmarshal([]byte(stdout), &balance); err != nil {
		t.Fatalf("decode balance: %v", err)
	}
	if len(balance.Balances) != 1 || balance.Balances[0].Balance != -75 {
		t.Fatalf("expected CNY balance -75, got %+v", balance.Balances)
	}
}

func TestRefundCLIOmitAmountRefundsRemaining(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "ledger.db")
	if _, _, err := runLedgerCommand(t, dbPath, "init"); err != nil {
		t.Fatalf("init: %v", err)
	}

	stdout, _, err := runLedgerCommand(t, dbPath, "--json", "add",
		"--amount", "100",
		"--direction", "expense",
		"--category", "餐饮",
	)
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	var added struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(stdout), &added); err != nil {
		t.Fatalf("decode add: %v", err)
	}

	stdout, _, err = runLedgerCommand(t, dbPath, "--json", "refund", "--id", added.ID)
	if err != nil {
		t.Fatalf("refund remaining: %v", err)
	}
	var refunded struct {
		RefundAmount float64 `json:"refund_amount"`
		NetAmount    float64 `json:"net_amount"`
	}
	if err := json.Unmarshal([]byte(stdout), &refunded); err != nil {
		t.Fatalf("decode refund: %v", err)
	}
	if refunded.RefundAmount != 100 || refunded.NetAmount != 0 {
		t.Fatalf("unexpected full refund fields: %+v", refunded)
	}
}

func TestRefundCLIValidation(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "ledger.db")
	if _, _, err := runLedgerCommand(t, dbPath, "init"); err != nil {
		t.Fatalf("init: %v", err)
	}

	if _, stderr, err := runLedgerCommand(t, dbPath, "refund", "--amount", "25"); err == nil {
		t.Fatal("expected missing id to fail")
	} else if !strings.Contains(stderr, "--id required") {
		t.Fatalf("expected --id required, got %q", stderr)
	}

	stdout, _, err := runLedgerCommand(t, dbPath, "--json", "add",
		"--amount", "100",
		"--direction", "expense",
		"--category", "餐饮",
	)
	if err != nil {
		t.Fatalf("add expense: %v", err)
	}
	var added struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(stdout), &added); err != nil {
		t.Fatalf("decode add: %v", err)
	}
	if _, _, err := runLedgerCommand(t, dbPath, "--json", "refund", "--id", added.ID, "--amount", "75"); err != nil {
		t.Fatalf("first refund: %v", err)
	}
	if _, stderr, err := runLedgerCommand(t, dbPath, "--json", "refund", "--id", added.ID, "--amount", "30"); err == nil {
		t.Fatal("expected over refund to fail")
	} else if !strings.Contains(stderr, "refund exceeds remaining") {
		t.Fatalf("expected over refund error, got %q", stderr)
	}

	stdout, _, err = runLedgerCommand(t, dbPath, "--json", "add",
		"--amount", "100",
		"--direction", "income",
	)
	if err != nil {
		t.Fatalf("add income: %v", err)
	}
	var income struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(stdout), &income); err != nil {
		t.Fatalf("decode income: %v", err)
	}
	if _, stderr, err := runLedgerCommand(t, dbPath, "--json", "refund", "--id", income.ID, "--amount", "1"); err == nil {
		t.Fatal("expected income refund to fail")
	} else if !strings.Contains(stderr, "cannot refund non-expense transaction") {
		t.Fatalf("expected income refund error, got %q", stderr)
	}

	stdout, _, err = runLedgerCommand(t, dbPath, "--json", "transfer",
		"--from-currency", "USD",
		"--to-currency", "CNY",
		"--from-amount", "100",
		"--to-amount", "720",
	)
	if err != nil {
		t.Fatalf("transfer: %v", err)
	}
	var transfer struct {
		Expense struct {
			ID string `json:"id"`
		} `json:"expense"`
	}
	if err := json.Unmarshal([]byte(stdout), &transfer); err != nil {
		t.Fatalf("decode transfer: %v", err)
	}
	if _, stderr, err := runLedgerCommand(t, dbPath, "--json", "refund", "--id", transfer.Expense.ID, "--amount", "1"); err == nil {
		t.Fatal("expected transfer refund to fail")
	} else if !strings.Contains(stderr, "cannot refund transfer leg") {
		t.Fatalf("expected transfer refund error, got %q", stderr)
	}
}

func TestQueryJSONIncludesRefundFields(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "ledger.db")
	if _, _, err := runLedgerCommand(t, dbPath, "init"); err != nil {
		t.Fatalf("init: %v", err)
	}

	stdout, _, err := runLedgerCommand(t, dbPath, "--json", "add",
		"--amount", "100",
		"--direction", "expense",
		"--category", "餐饮",
	)
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	var added struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(stdout), &added); err != nil {
		t.Fatalf("decode add: %v", err)
	}
	if _, _, err := runLedgerCommand(t, dbPath, "--json", "refund", "--id", added.ID, "--amount", "25"); err != nil {
		t.Fatalf("refund: %v", err)
	}

	stdout, _, err = runLedgerCommand(t, dbPath, "--json", "query")
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	var queried struct {
		Items []struct {
			RefundAmount float64 `json:"refund_amount"`
			NetAmount    float64 `json:"net_amount"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(stdout), &queried); err != nil {
		t.Fatalf("decode query: %v", err)
	}
	if len(queried.Items) != 1 || queried.Items[0].RefundAmount != 25 || queried.Items[0].NetAmount != 75 {
		t.Fatalf("unexpected query refund fields: %+v", queried.Items)
	}
}

func TestManagementCommandsAndRegressionFlows(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "ledger.db")

	if _, _, err := runLedgerCommand(t, dbPath, "init"); err != nil {
		t.Fatalf("init: %v", err)
	}

	stdout, _, err := runLedgerCommand(t, dbPath, "--json", "category", "add", "--name", "差旅", "--direction", "expense")
	if err != nil {
		t.Fatalf("category add: %v", err)
	}
	var category struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal([]byte(stdout), &category); err != nil {
		t.Fatalf("decode category add: %v", err)
	}
	if category.Name != "差旅" {
		t.Fatalf("unexpected category add result: %+v", category)
	}

	stdout, _, err = runLedgerCommand(t, dbPath, "--json", "category", "list")
	if err != nil {
		t.Fatalf("category list: %v", err)
	}
	if !strings.Contains(stdout, "差旅") {
		t.Fatalf("expected category list to include 差旅, got %q", stdout)
	}

	stdout, _, err = runLedgerCommand(t, dbPath, "--json", "tag", "add", "--name", "报销")
	if err != nil {
		t.Fatalf("tag add: %v", err)
	}
	var tag struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal([]byte(stdout), &tag); err != nil {
		t.Fatalf("decode tag add: %v", err)
	}
	if tag.Name != "报销" {
		t.Fatalf("unexpected tag add result: %+v", tag)
	}

	if _, _, err := runLedgerCommand(t, dbPath, "add",
		"--amount", "1200",
		"--direction", "expense",
		"--category", "差旅",
		"--description", "机票",
		"--tag", "报销",
	); err != nil {
		t.Fatalf("add transaction: %v", err)
	}

	stdout, _, err = runLedgerCommand(t, dbPath, "--json", "query", "--category", "差旅", "--tag", "报销")
	if err != nil {
		t.Fatalf("query with filters: %v", err)
	}
	var queried struct {
		Total int `json:"total"`
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(stdout), &queried); err != nil {
		t.Fatalf("decode filtered query: %v", err)
	}
	if queried.Total != 1 || len(queried.Items) != 1 {
		t.Fatalf("unexpected filtered query result: %+v", queried)
	}
	txID := queried.Items[0].ID

	if _, _, err := runLedgerCommand(t, dbPath, "--json", "update", "--id", txID, "--amount", "1300", "--description", "机票改签"); err != nil {
		t.Fatalf("update command: %v", err)
	}
	stdout, _, err = runLedgerCommand(t, dbPath, "--json", "update",
		"--id", txID,
		"--category", "餐饮",
		"--tag", "工作餐",
		"--tag", "出差",
	)
	if err != nil {
		t.Fatalf("update command category/tags: %v", err)
	}
	if !strings.Contains(stdout, "\"category\": \"餐饮\"") || !strings.Contains(stdout, "\"工作餐\"") || !strings.Contains(stdout, "\"出差\"") {
		t.Fatalf("unexpected category/tag update output: %q", stdout)
	}
	stdout, _, err = runLedgerCommand(t, dbPath, "--json", "update",
		"--id", txID,
		"--add-tag", "高铁",
		"--remove-tag", "出差",
	)
	if err != nil {
		t.Fatalf("update command add/remove tags: %v", err)
	}
	if !strings.Contains(stdout, "\"高铁\"") || strings.Contains(stdout, "\"出差\"") {
		t.Fatalf("unexpected incremental tag update output: %q", stdout)
	}

	stdout, _, err = runLedgerCommand(t, dbPath, "--json", "balance", "--currency", "CNY")
	if err != nil {
		t.Fatalf("balance command: %v", err)
	}
	if !strings.Contains(stdout, "\"currency\": \"CNY\"") {
		t.Fatalf("unexpected balance output: %q", stdout)
	}

	stdout, _, err = runLedgerCommand(t, dbPath, "--json", "transfer",
		"--from-currency", "USD",
		"--to-currency", "CNY",
		"--from-amount", "100",
		"--to-amount", "720",
	)
	if err != nil {
		t.Fatalf("transfer command: %v", err)
	}
	if !strings.Contains(stdout, "\"transfer_group\"") {
		t.Fatalf("unexpected transfer output: %q", stdout)
	}

	stdout, _, err = runLedgerCommand(t, dbPath, "--json", "audit", "--limit", "20")
	if err != nil {
		t.Fatalf("audit command: %v", err)
	}
	if !strings.Contains(stdout, "\"items\"") {
		t.Fatalf("unexpected audit output: %q", stdout)
	}

	if _, _, err := runLedgerCommand(t, dbPath, "--json", "delete", "--id", txID); err != nil {
		t.Fatalf("delete command: %v", err)
	}
	if _, _, err := runLedgerCommand(t, dbPath, "--json", "tag", "remove", "--name", "报销"); err != nil {
		t.Fatalf("tag remove: %v", err)
	}
	if _, _, err := runLedgerCommand(t, dbPath, "--json", "category", "remove", "--name", "差旅", "--force"); err != nil {
		t.Fatalf("category remove force: %v", err)
	}
}
