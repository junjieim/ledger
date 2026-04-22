package tokenizer

import (
	"strings"
	"sync"

	"github.com/go-ego/gse"
)

var (
	loadSegmenterOnce sync.Once
	segmenter         gse.Segmenter
	segmenterErr      error
)

func TokenizeDocument(text string) (string, error) {
	tokens, err := cutSearch(text)
	if err != nil {
		return "", err
	}
	return strings.Join(tokens, " "), nil
}

func TokenizeQuery(text string) (string, error) {
	tokens, err := cutSearch(text)
	if err != nil {
		return "", err
	}
	if len(tokens) == 0 {
		return "", nil
	}

	seen := make(map[string]struct{}, len(tokens))
	terms := make([]string, 0, len(tokens))
	for _, token := range tokens {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}
		if _, ok := seen[token]; ok {
			continue
		}
		seen[token] = struct{}{}
		terms = append(terms, token)
	}

	switch len(terms) {
	case 0:
		return "", nil
	case 1:
		return terms[0], nil
	default:
		return strings.Join(terms, " OR "), nil
	}
}

func cutSearch(text string) ([]string, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, nil
	}

	seg, err := getSegmenter()
	if err != nil {
		return nil, err
	}

	tokens := seg.CutSearch(text, true)
	out := make([]string, 0, len(tokens))
	for _, token := range tokens {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}
		out = append(out, token)
	}
	if len(out) == 0 {
		return []string{text}, nil
	}
	return out, nil
}

func getSegmenter() (*gse.Segmenter, error) {
	loadSegmenterOnce.Do(func() {
		segmenter = gse.Segmenter{}
		segmenterErr = segmenter.LoadDictEmbed()
		if segmenterErr != nil {
			segmenterErr = segmenter.LoadDict()
		}
	})
	if segmenterErr != nil {
		return nil, segmenterErr
	}
	return &segmenter, nil
}
