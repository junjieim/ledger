package tokenizer

import (
	"strings"
	"testing"
)

func TestTokenizeDocumentAndQuery(t *testing.T) {
	document, err := TokenizeDocument("和同事一起吃火锅")
	if err != nil {
		t.Fatalf("tokenize document: %v", err)
	}
	if strings.TrimSpace(document) == "" {
		t.Fatal("expected tokenized document to be non-empty")
	}

	query, err := TokenizeQuery("火锅 火锅 聚餐")
	if err != nil {
		t.Fatalf("tokenize query: %v", err)
	}
	if strings.TrimSpace(query) == "" {
		t.Fatal("expected tokenized query to be non-empty")
	}
	if strings.Contains(query, "火锅 OR 火锅") {
		t.Fatalf("expected duplicate tokens to be removed, got %q", query)
	}
}

func TestTokenizeQueryEmptyInput(t *testing.T) {
	query, err := TokenizeQuery("   ")
	if err != nil {
		t.Fatalf("tokenize empty query: %v", err)
	}
	if query != "" {
		t.Fatalf("expected empty query, got %q", query)
	}
}
