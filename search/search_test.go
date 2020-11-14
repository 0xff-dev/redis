package search

import "testing"

var content = "In order to construct out SET's of documents, we must first examine our documents for words. The process of " +
	"extraction words from documents is known as parsing and tokenization; we are producing a set of tokens (or words) " +
	"that identify the document"

func TestSearch(t *testing.T) {
	if err := indexDoc("try-set", content); err != nil {
		t.Fatal(err)
		return
	}
}

func TestReadStopWords(t *testing.T) {
	s, err := readStopWords("./stop_words")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(s)
}

func TestTokenize(t *testing.T) {
	s, err := tokenize(content)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(s)
}

func TestParse(t *testing.T) {
	query := `connect +connection +disconnect +disconnection chat -proxy -proxies`
	allSets, array := parse(query)
	t.Log("allSets: ", allSets, "\narray: ", array)
}

func TestParseAndSearch(t *testing.T) {
	query := "process tokens"
	id := parseAndSearch(query, 60)
	t.Log(id)
}
