package cli

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	ledgerdb "github.com/ledger-ai/ledger/internal/db"
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

func openTestDB(t *testing.T, dbPath string) *sql.DB {
	t.Helper()
	db, err := ledgerdb.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
	return db
}

func TestConfigContractAndWarnings(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "ledger.db")

	_, stderr, err := runLedgerCommand(t, dbPath, "init")
	if err != nil {
		t.Fatalf("init command: %v", err)
	}
	if !strings.Contains(stderr, "embedding configuration is incomplete (missing: api_key)") {
		t.Fatalf("expected init warning, got %q", stderr)
	}

	stdout, stderr, err := runLedgerCommand(t, dbPath, "--json", "config", "show")
	if err != nil {
		t.Fatalf("config show: %v", err)
	}
	if stderr != "" {
		t.Fatalf("config show should not warn, got %q", stderr)
	}
	var shown map[string]any
	if err := json.Unmarshal([]byte(stdout), &shown); err != nil {
		t.Fatalf("decode config show: %v", err)
	}
	if shown["api_key"] != "(not set)" {
		t.Fatalf("expected masked empty api_key, got %#v", shown["api_key"])
	}

	stdout, stderr, err = runLedgerCommand(t, dbPath, "--json", "config", "set",
		"--api-key", "dummy-key-123456",
		"--model-name", "embedding-3",
		"--model-url", "https://example.com/embed",
		"--dimensions", "4",
	)
	if err != nil {
		t.Fatalf("config set: %v", err)
	}
	if stderr != "" {
		t.Fatalf("config set should not warn, got %q", stderr)
	}
	var setResult map[string]any
	if err := json.Unmarshal([]byte(stdout), &setResult); err != nil {
		t.Fatalf("decode config set: %v", err)
	}
	if setResult["api_key"] != "dum******456" {
		t.Fatalf("expected masked api_key, got %#v", setResult["api_key"])
	}

	_, stderr, err = runLedgerCommand(t, dbPath, "query", "--limit", "1")
	if err != nil {
		t.Fatalf("query command: %v", err)
	}
	if strings.Contains(stderr, "embedding configuration is incomplete") {
		t.Fatalf("did not expect warning after config set, got %q", stderr)
	}
}

func TestSearchWithoutEmbeddingConfigMatchesDocs(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "ledger.db")

	if _, _, err := runLedgerCommand(t, dbPath, "init"); err != nil {
		t.Fatalf("init: %v", err)
	}
	if _, _, err := runLedgerCommand(t, dbPath, "add",
		"--amount", "20",
		"--direction", "expense",
		"--category", "餐饮",
		"--description", "火锅",
	); err != nil {
		t.Fatalf("add: %v", err)
	}

	stdout, stderr, err := runLedgerCommand(t, dbPath, "--json", "search", "--semantic", "火锅", "--mode", "semantic")
	if err != nil {
		t.Fatalf("semantic search should not error when config is missing: %v", err)
	}
	if !strings.Contains(stderr, "embedding configuration is incomplete") || !strings.Contains(stderr, "semantic search returned no vector results") {
		t.Fatalf("unexpected semantic warning output: %q", stderr)
	}
	var semanticResult struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal([]byte(stdout), &semanticResult); err != nil {
		t.Fatalf("decode semantic result: %v", err)
	}
	if len(semanticResult.Items) != 0 {
		t.Fatalf("expected empty semantic result, got %#v", semanticResult.Items)
	}

	stdout, stderr, err = runLedgerCommand(t, dbPath, "--json", "search",
		"--keyword", "火锅",
		"--semantic", "聚餐",
		"--mode", "hybrid",
	)
	if err != nil {
		t.Fatalf("hybrid search: %v", err)
	}
	if !strings.Contains(stderr, "hybrid search is returning keyword results only") {
		t.Fatalf("expected hybrid fallback warning, got %q", stderr)
	}
	var hybridResult struct {
		Items []struct {
			MatchType string `json:"match_type"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(stdout), &hybridResult); err != nil {
		t.Fatalf("decode hybrid result: %v", err)
	}
	if len(hybridResult.Items) != 1 || hybridResult.Items[0].MatchType != "keyword" {
		t.Fatalf("expected keyword-only hybrid fallback, got %#v", hybridResult.Items)
	}
}

func TestSemanticSearchReembedsWhenDimensionsChange(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			Input      []string `json:"input"`
			Dimensions int      `json:"dimensions"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		type item struct {
			Index     int       `json:"index"`
			Embedding []float64 `json:"embedding"`
		}
		resp := struct {
			Data []item `json:"data"`
		}{}
		for i := range payload.Input {
			vec := make([]float64, payload.Dimensions)
			for j := range vec {
				vec[j] = float64(j + 1)
			}
			resp.Data = append(resp.Data, item{Index: i, Embedding: vec})
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	dbPath := filepath.Join(t.TempDir(), "ledger.db")
	if _, _, err := runLedgerCommand(t, dbPath, "init"); err != nil {
		t.Fatalf("init: %v", err)
	}
	if _, _, err := runLedgerCommand(t, dbPath, "add",
		"--amount", "28.5",
		"--direction", "expense",
		"--category", "餐饮",
		"--description", "午餐牛肉面",
	); err != nil {
		t.Fatalf("add: %v", err)
	}
	if _, _, err := runLedgerCommand(t, dbPath, "--json", "config", "set",
		"--api-key", "dummy-key-123456",
		"--model-name", "embedding-3",
		"--model-url", server.URL,
		"--dimensions", "4",
	); err != nil {
		t.Fatalf("config set 4d: %v", err)
	}
	if _, _, err := runLedgerCommand(t, dbPath, "--json", "search", "--semantic", "午餐", "--mode", "semantic"); err != nil {
		t.Fatalf("semantic search with config: %v", err)
	}

	db := openTestDB(t, dbPath)
	var dim1 int
	var sig1 string
	if err := db.QueryRow(`SELECT dimensions, config_signature FROM transaction_embeddings LIMIT 1`).Scan(&dim1, &sig1); err != nil {
		t.Fatalf("load embedding row after first sync: %v", err)
	}
	if dim1 != 4 {
		t.Fatalf("expected dimension 4, got %d", dim1)
	}

	if _, _, err := runLedgerCommand(t, dbPath, "--json", "config", "set",
		"--api-key", "dummy-key-123456",
		"--model-name", "embedding-3",
		"--model-url", server.URL,
		"--dimensions", "6",
	); err != nil {
		t.Fatalf("config set 6d: %v", err)
	}
	if _, _, err := runLedgerCommand(t, dbPath, "--json", "search", "--semantic", "午餐", "--mode", "semantic"); err != nil {
		t.Fatalf("semantic search after dimension change: %v", err)
	}

	var dim2 int
	var sig2 string
	if err := db.QueryRow(`SELECT dimensions, config_signature FROM transaction_embeddings LIMIT 1`).Scan(&dim2, &sig2); err != nil {
		t.Fatalf("load embedding row after second sync: %v", err)
	}
	if dim2 != 6 {
		t.Fatalf("expected dimension 6 after re-embed, got %d", dim2)
	}
	if sig1 == sig2 {
		t.Fatalf("expected config signature to change after dimension update, got %q", sig1)
	}
}
