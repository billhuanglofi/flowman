package insomnia

import (
	"encoding/json"
	"fmt"
	"os"
)

type document struct {
	Type      string     `json:"_type"`
	Resources []resource `json:"resources"`
}

type resource struct {
	ID                 string                     `json:"_id"`
	Type               string                     `json:"_type"`
	ParentID           string                     `json:"parentId"`
	Name               string                     `json:"name"`
	Method             string                     `json:"method"`
	URL                string                     `json:"url"`
	Parameters         []parameter                `json:"parameters"`
	Headers            []header                   `json:"headers"`
	Body               body                       `json:"body"`
	Authentication     map[string]any             `json:"authentication"`
	ClientCertificates []map[string]any           `json:"clientCertificates"`
	CookieJar          map[string]any             `json:"cookieJar"`
	Extra              map[string]json.RawMessage `json:"-"`
}

type parameter struct {
	Name     string `json:"name"`
	Value    string `json:"value"`
	FileName string `json:"fileName"`
}

type header struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type body struct {
	MimeType string      `json:"mimeType"`
	Text     string      `json:"text"`
	Params   []parameter `json:"params"`
}

func loadDocument(path string) (document, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return document{}, fmt.Errorf("read insomnia document %s: %w", path, err)
	}
	var doc document
	if err := json.Unmarshal(data, &doc); err != nil {
		return document{}, fmt.Errorf("decode insomnia document %s: %w", path, err)
	}
	return doc, nil
}
