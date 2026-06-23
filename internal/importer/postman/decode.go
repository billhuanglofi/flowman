package postman

import (
	"encoding/json"
	"fmt"
	"os"
)

type collectionEnvelope struct {
	Info     infoObject                 `json:"info"`
	Items    []itemObject               `json:"item"`
	Variable []map[string]any           `json:"variable"`
	Extra    map[string]json.RawMessage `json:"-"`
}

type infoObject struct {
	Name string `json:"name"`
}

type itemObject struct {
	Name    string                     `json:"name"`
	Items   []itemObject               `json:"item"`
	Request requestObject              `json:"request"`
	Event   []eventObject              `json:"event"`
	Extra   map[string]json.RawMessage `json:"-"`
}

type eventObject struct {
	Listen string       `json:"listen"`
	Script scriptObject `json:"script"`
}

type scriptObject struct {
	Exec []string `json:"exec"`
}

type requestObject struct {
	Method      string                     `json:"method"`
	Header      []headerObject             `json:"header"`
	URL         json.RawMessage            `json:"url"`
	Body        bodyObject                 `json:"body"`
	Auth        authObject                 `json:"auth"`
	Certificate map[string]any             `json:"certificate"`
	Extra       map[string]json.RawMessage `json:"-"`
}

type headerObject struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Type  string `json:"type"`
}

type bodyObject struct {
	Mode     string                     `json:"mode"`
	Raw      string                     `json:"raw"`
	Options  rawOptionsObject           `json:"options"`
	URLEnc   []headerObject             `json:"urlencoded"`
	FormData []formDataObject           `json:"formdata"`
	GraphQL  map[string]any             `json:"graphql"`
	Extra    map[string]json.RawMessage `json:"-"`
}

type rawOptionsObject struct {
	Raw map[string]any `json:"raw"`
}

type formDataObject struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Type  string `json:"type"`
	Src   any    `json:"src"`
}

type authObject struct {
	Type   string                     `json:"type"`
	Basic  []authEntry                `json:"basic"`
	Bearer []authEntry                `json:"bearer"`
	APIKey []authEntry                `json:"apikey"`
	Extra  map[string]json.RawMessage `json:"-"`
}

type authEntry struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type urlObject struct {
	Raw      string        `json:"raw"`
	Protocol string        `json:"protocol"`
	Host     []string      `json:"host"`
	Path     []string      `json:"path"`
	Query    []queryObject `json:"query"`
}

type queryObject struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func (item *itemObject) UnmarshalJSON(data []byte) error {
	type alias itemObject
	aux := alias{}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	*item = itemObject(aux)
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	deleteKnownKeys(raw, item)
	item.Extra = raw
	return nil
}

func (request *requestObject) UnmarshalJSON(data []byte) error {
	type alias requestObject
	aux := alias{}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	*request = requestObject(aux)
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	deleteKnownKeys(raw, request)
	request.Extra = raw
	return nil
}

func (body *bodyObject) UnmarshalJSON(data []byte) error {
	type alias bodyObject
	aux := alias{}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	*body = bodyObject(aux)
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	deleteKnownKeys(raw, body)
	body.Extra = raw
	return nil
}

func (auth *authObject) UnmarshalJSON(data []byte) error {
	type alias authObject
	aux := alias{}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	*auth = authObject(aux)
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	deleteKnownKeys(raw, auth)
	auth.Extra = raw
	return nil
}

func loadCollection(path string) (collectionEnvelope, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return collectionEnvelope{}, fmt.Errorf("read postman collection %s: %w", path, err)
	}
	var collection collectionEnvelope
	if err := unmarshalWithExtra(data, &collection, &collection.Extra); err != nil {
		return collectionEnvelope{}, fmt.Errorf("decode postman collection %s: %w", path, err)
	}
	return collection, nil
}

func unmarshalWithExtra[T any](data []byte, target *T, extra *map[string]json.RawMessage) error {
	if err := json.Unmarshal(data, target); err != nil {
		return err
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	deleteKnownKeys(raw, target)
	*extra = raw
	return nil
}

func deleteKnownKeys(raw map[string]json.RawMessage, target any) {
	switch target.(type) {
	case *collectionEnvelope:
		delete(raw, "info")
		delete(raw, "item")
		delete(raw, "variable")
	case *itemObject:
		delete(raw, "name")
		delete(raw, "item")
		delete(raw, "request")
		delete(raw, "event")
	case *requestObject:
		delete(raw, "method")
		delete(raw, "header")
		delete(raw, "url")
		delete(raw, "body")
		delete(raw, "auth")
		delete(raw, "certificate")
	case *bodyObject:
		delete(raw, "mode")
		delete(raw, "raw")
		delete(raw, "options")
		delete(raw, "urlencoded")
		delete(raw, "formdata")
		delete(raw, "graphql")
	case *authObject:
		delete(raw, "type")
		delete(raw, "basic")
		delete(raw, "bearer")
		delete(raw, "apikey")
	}
}
