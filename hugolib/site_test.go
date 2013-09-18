package hugolib

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"
	"testing"
)

const (
	TEMPLATE_TITLE    = "{{ .Title }}"
	PAGE_SIMPLE_TITLE = `---
title: simple template
---
content`

	TEMPLATE_MISSING_FUNC        = "{{ .Title | funcdoesnotexists }}"
	TEMPLATE_FUNC                = "{{ .Title | urlize }}"
	TEMPLATE_CONTENT             = "{{ .Content }}"
	TEMPLATE_DATE                = "{{ .Date }}"
	INVALID_TEMPLATE_FORMAT_DATE = "{{ .Date.Format time.RFC3339 }}"
	TEMPLATE_WITH_URL            = "<a href=\"foobar.jpg\">Going</a>"
	PAGE_URL_SPECIFIED           = `---
title: simple template
url: "mycategory/my-whatever-content/"
---
content`

	PAGE_WITH_MD = `---
title: page with md
---
# heading 1
text
## heading 2
more text
`
)

func pageMust(p *Page, err error) *Page {
	if err != nil {
		panic(err)
	}
	return p
}

func TestDegenerateRenderThingMissingTemplate(t *testing.T) {
	p, _ := ReadFrom(strings.NewReader(PAGE_SIMPLE_TITLE), "content/a/file.md")
	s := new(Site)
	s.prepTemplates()
	_, err := s.RenderThing(p, "foobar")
	if err == nil {
		t.Errorf("Expected err to be returned when missing the template.")
	}
}

func TestAddInvalidTemplate(t *testing.T) {
	s := new(Site)
	s.prepTemplates()
	err := s.addTemplate("missing", TEMPLATE_MISSING_FUNC)
	if err == nil {
		t.Fatalf("Expecting the template to return an error")
	}
}

func matchRender(t *testing.T, s *Site, p *Page, tmplName string, expected string) {
	content, err := s.RenderThing(p, tmplName)
	if err != nil {
		t.Fatalf("Unable to render template.")
	}

	if string(content.Bytes()) != expected {
		t.Fatalf("Content did not match expected: %s. got: %s", expected, content)
	}
}

func _TestAddSameTemplateTwice(t *testing.T) {
	p := pageMust(ReadFrom(strings.NewReader(PAGE_SIMPLE_TITLE), "content/a/file.md"))
	s := new(Site)
	s.prepTemplates()
	err := s.addTemplate("foo", TEMPLATE_TITLE)
	if err != nil {
		t.Fatalf("Unable to add template foo")
	}

	matchRender(t, s, p, "foo", "simple template")

	err = s.addTemplate("foo", "NEW {{ .Title }}")
	if err != nil {
		t.Fatalf("Unable to add template foo: %s", err)
	}

	matchRender(t, s, p, "foo", "NEW simple template")
}

func TestRenderThing(t *testing.T) {
	tests := []struct {
		content  string
		template string
		expected string
	}{
		{PAGE_SIMPLE_TITLE, TEMPLATE_TITLE, "simple template"},
		{PAGE_SIMPLE_TITLE, TEMPLATE_FUNC, "simple-template"},
		{PAGE_WITH_MD, TEMPLATE_CONTENT, "<h1>heading 1</h1>\n\n<p>text</p>\n\n<h2>heading 2</h2>\n\n<p>more text</p>\n"},
		{SIMPLE_PAGE_RFC3339_DATE, TEMPLATE_DATE, "2013-05-17 16:59:30 &#43;0000 UTC"},
	}

	s := new(Site)
	s.prepTemplates()

	for i, test := range tests {
		p, err := ReadFrom(strings.NewReader(test.content), "content/a/file.md")
		if err != nil {
			t.Fatalf("Error parsing buffer: %s", err)
		}
		templateName := fmt.Sprintf("foobar%d", i)
		err = s.addTemplate(templateName, test.template)
		if err != nil {
			t.Fatalf("Unable to add template")
		}

		p.Content = template.HTML(p.Content)
		html, err2 := s.RenderThing(p, templateName)
		if err2 != nil {
			t.Errorf("Unable to render html: %s", err)
		}

		if string(html.Bytes()) != test.expected {
			t.Errorf("Content does not match.\nExpected\n\t'%q'\ngot\n\t'%q'", test.expected, html)
		}
	}
}

func TestRenderThingOrDefault(t *testing.T) {
	tests := []struct {
		content  string
		missing  bool
		template string
		expected string
	}{
		{PAGE_SIMPLE_TITLE, true, TEMPLATE_TITLE, "simple template"},
		{PAGE_SIMPLE_TITLE, true, TEMPLATE_FUNC, "simple-template"},
		{PAGE_SIMPLE_TITLE, false, TEMPLATE_TITLE, "simple template"},
		{PAGE_SIMPLE_TITLE, false, TEMPLATE_FUNC, "simple-template"},
	}

	s := new(Site)
	s.prepTemplates()

	for i, test := range tests {
		p, err := ReadFrom(strings.NewReader(PAGE_SIMPLE_TITLE), "content/a/file.md")
		if err != nil {
			t.Fatalf("Error parsing buffer: %s", err)
		}
		templateName := fmt.Sprintf("default%d", i)
		err = s.addTemplate(templateName, test.template)
		if err != nil {
			t.Fatalf("Unable to add template")
		}

		var html *bytes.Buffer
		var err2 error
		if test.missing {
			html, err2 = s.RenderThingOrDefault(p, "missing", templateName)
		} else {
			html, err2 = s.RenderThingOrDefault(p, templateName, "missing_default")
		}

		if err2 != nil {
			t.Errorf("Unable to render html: %s", err)
		}

		if string(html.Bytes()) != test.expected {
			t.Errorf("Content does not match.  Expected '%s', got '%s'", test.expected, html)
		}
	}
}

func TestSetOutFile(t *testing.T) {
	s := new(Site)
	p := pageMust(ReadFrom(strings.NewReader(PAGE_URL_SPECIFIED), "content/a/file.md"))
	s.setOutFile(p)

	expected := "mycategory/my-whatever-content/index.html"

	if p.OutFile != "mycategory/my-whatever-content/index.html" {
		t.Errorf("Outfile does not match.  Expected '%s', got '%s'", expected, p.OutFile)
	}
}

func TestAbsUrlify(t *testing.T) {
	files := make(map[string][]byte)
	target := &InMemoryTarget{files: files}
	s := &Site{
		Target: target,
		Config: Config{BaseUrl: "http://auth/bub/"},
		Source: &inMemorySource{urlFakeSource},
	}
	s.initializeSiteInfo()
	s.prepTemplates()
	must(s.addTemplate("blue/single.html", TEMPLATE_WITH_URL))

	if err := s.CreatePages(); err != nil {
		t.Fatalf("Unable to create pages: %s", err)
	}

	if err := s.BuildSiteMeta(); err != nil {
		t.Fatalf("Unable to build site metadata: %s", err)
	}

	if err := s.RenderPages(); err != nil {
		t.Fatalf("Unable to render pages. %s", err)
	}

	content, ok := target.files["content/blue/slug-doc-1.html"]
	if !ok {
		t.Fatalf("Unable to locate rendered content")
	}

	expected := "<html><head></head><body><a href=\"http://auth/bub/foobar.jpg\">Going</a></body></html>"
	if string(content) != expected {
		t.Errorf("Expected: %q, got: %q", expected, string(content))
	}
}