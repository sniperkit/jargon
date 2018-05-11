package stackexchange

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/format"
	"io/ioutil"
	"net/http"
	"os"
	"text/template"
	"time"
)

func writeDictionary() error {
	pages := 10
	pageSize := 100

	var items []item
	for page := 1; page <= pages; page++ {
		wrapper, err := fetchTags(page, pageSize)
		if err != nil {
			return err
		}
		items = append(items, wrapper.Items...)

		if !wrapper.HasMore {
			break
		}
	}

	tmpl := `
// This file is generated by github.com/clipperhouse/jargon/stackexchange/generator.go
// Best not to modify it, as it will likely be overwritten
package stackexchange

type dict struct {
	tags []string
	synonyms map[string]string
}

func (d *dict) GetTags() []string {
	return d.tags
}

func (d *dict) GetSynonyms() map[string]string {
	return d.synonyms
}

// Dictionary is the main exported Dictionary of Stack Exchange tags and synonyms, fetched via api.stackexchange.com
// It includes the most popular {{ . | len }} tags and their synonyms
var Dictionary = &dict{tags, synonyms}

var tags = []string {
	{{ range . -}}
		"{{ .Name }}",
	{{ end -}}
}

var synonyms = map[string]string {
{{ range $i, $tag := . -}}
	{{ range .Synonyms -}}
		"{{ . }}": "{{ $tag.Name }}",
	{{ end -}}
{{ end -}}
}
	
`
	t := template.Must(template.New("dict").Parse(tmpl))

	var source bytes.Buffer
	tmplErr := t.Execute(&source, items)
	if tmplErr != nil {
		return tmplErr
	}

	formatted, fmtErr := format.Source(source.Bytes())

	if fmtErr != nil {
		return fmtErr
	}

	f, createErr := os.Create("generated.go")
	if createErr != nil {
		return createErr
	}
	defer f.Close()

	_, writeErr := f.Write(formatted)

	if writeErr != nil {
		return writeErr
	}

	return nil
}

var tagsURL = "http://api.stackexchange.com/2.2/tags?page=%d&pagesize=%d&order=desc&sort=popular&site=stackoverflow&filter=!4-J-du8hXSkh2Is1a&page=%d"
var client = http.Client{
	Timeout: time.Second * 2, // Maximum of 2 secs
}
var empty = wrapper{}

func fetchTags(page int, pageSize int) (wrapper, error) {
	if page == 0 {
		page = 1
	}

	if pageSize == 0 {
		pageSize = 100
	}

	url := fmt.Sprintf(tagsURL, page, pageSize)
	r, httpErr := client.Get(url)
	if httpErr != nil {
		return empty, httpErr
	}

	defer r.Body.Close()

	body, readErr := ioutil.ReadAll(r.Body)
	if readErr != nil {
		return empty, readErr
	}

	wrapper := wrapper{}
	jsonErr := json.Unmarshal(body, &wrapper)
	if jsonErr != nil {
		return empty, jsonErr
	}

	return wrapper, nil
}

type item struct {
	Name      string   `json:"name"`
	Synonyms  []string `json:"synonyms"`
	Moderator bool     `json:"is_moderator_only"`
}

type wrapper struct {
	Items   []item `json:"items"`
	HasMore bool   `json:"has_more"`
}
