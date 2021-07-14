package template

import (
	"errors"
	"fmt"
	htmltemplate "html/template"
	"os"
	"path"
	"regexp"
	texttemplate "text/template"

	"github.com/authgear/authgear-server/pkg/util/intl"
	"github.com/authgear/authgear-server/pkg/util/intlresource"
	"github.com/authgear/authgear-server/pkg/util/resource"
)

type Resource interface {
	templateResource()
}

// HTML defines a HTML template
type HTML struct {
	// Name is the name of template
	Name string
	// ComponentDependencies is the HTML component templates this template depends on.
	ComponentDependencies []*HTML
}

var _ resource.Descriptor = &HTML{}

func (t *HTML) templateResource() {}

func (t *HTML) MatchResource(path string) (*resource.Match, bool) {
	return matchTemplatePath(path, t.Name)
}

func (t *HTML) FindResources(fs resource.Fs) ([]resource.Location, error) {
	return readTemplates(fs, t.Name)
}

func (t *HTML) ViewResources(resources []resource.ResourceFile, view resource.View) (interface{}, error) {
	return viewHTMLTemplates(t.Name, resources, view)
}

func (t *HTML) UpdateResource(ctx resource.Context, resrc *resource.ResourceFile, data []byte, view resource.View) (*resource.ResourceFile, error) {
	return &resource.ResourceFile{
		Location: resrc.Location,
		Data:     data,
	}, nil
}

// PlainText defines a plain text template
type PlainText struct {
	// Name is the name of template
	Name string
	// ComponentDependencies is the plain text component templates this template depends on.
	ComponentDependencies []*PlainText
}

var _ resource.Descriptor = &PlainText{}

func (t *PlainText) templateResource() {}

func (t *PlainText) MatchResource(path string) (*resource.Match, bool) {
	return matchTemplatePath(path, t.Name)
}

func (t *PlainText) FindResources(fs resource.Fs) ([]resource.Location, error) {
	return readTemplates(fs, t.Name)
}

func (t *PlainText) ViewResources(resources []resource.ResourceFile, view resource.View) (interface{}, error) {
	return viewTextTemplates(t.Name, resources, view)
}

func (t *PlainText) UpdateResource(ctx resource.Context, resrc *resource.ResourceFile, data []byte, view resource.View) (*resource.ResourceFile, error) {
	return &resource.ResourceFile{
		Location: resrc.Location,
		Data:     data,
	}, nil
}

func RegisterHTML(name string, dependencies ...*HTML) *HTML {
	desc := &HTML{Name: name, ComponentDependencies: dependencies}
	resource.RegisterResource(desc)
	return desc
}

func RegisterPlainText(name string, dependencies ...*PlainText) *PlainText {
	desc := &PlainText{Name: name, ComponentDependencies: dependencies}
	resource.RegisterResource(desc)
	return desc
}

func matchTemplatePath(path string, templateName string) (*resource.Match, bool) {
	r := fmt.Sprintf("^templates/([a-zA-Z0-9-]+)/%s$", regexp.QuoteMeta(templateName))
	matches := regexp.MustCompile(r).FindStringSubmatch(path)
	if len(matches) != 2 {
		return nil, false
	}

	languageTag := matches[1]

	isLanguageTagValid := false
	for _, localeKey := range intl.AvailableLanguages {
		if languageTag == localeKey {
			isLanguageTagValid = true
			break
		}
	}
	if !isLanguageTagValid {
		return nil, false
	}

	return &resource.Match{LanguageTag: languageTag}, true
}

func readTemplates(fs resource.Fs, templateName string) ([]resource.Location, error) {
	templatesDir, err := fs.Open("templates")
	if os.IsNotExist(err) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	defer templatesDir.Close()

	langTagDirs, err := templatesDir.Readdirnames(0)
	if err != nil {
		return nil, err
	}

	var locations []resource.Location
	for _, langTag := range langTagDirs {
		p := path.Join("templates", langTag, templateName)
		location := resource.Location{
			Fs:   fs,
			Path: p,
		}
		_, err := resource.ReadLocation(location)
		if os.IsNotExist(err) {
			continue
		} else if err != nil {
			return nil, err
		}
		locations = append(locations, location)
	}

	return locations, nil
}

type languageTemplate struct {
	languageTag string
	data        []byte
}

func (t languageTemplate) GetLanguageTag() string {
	return t.languageTag
}

var templateLanguageTagRegex = regexp.MustCompile("^templates/([a-zA-Z0-9-_]+)/")

func viewTemplates(resources []resource.ResourceFile, rawView resource.View) (interface{}, error) {
	switch view := rawView.(type) {
	case resource.AppFileView:
		return viewTemplatesAppFile(resources, view)
	case resource.EffectiveFileView:
		return viewTemplatesEffectiveFile(resources, view)
	case resource.EffectiveResourceView:
		return viewTemplatesEffectiveResource(resources, view)
	default:
		return nil, fmt.Errorf("unsupported view: %T", rawView)
	}
}

func viewTemplatesAppFile(resources []resource.ResourceFile, view resource.AppFileView) (interface{}, error) {
	// When template is viewed as AppFile,
	// the exact file is returned.
	path := view.AppFilePath()

	var found bool
	var bytes []byte
	for _, resrc := range resources {
		if resrc.Location.Fs.GetFsLevel() == resource.FsLevelApp && resrc.Location.Path == path {
			found = true
			bytes = resrc.Data
		}
	}

	if !found {
		return nil, resource.ErrResourceNotFound
	}

	return bytes, nil
}

func viewTemplatesEffectiveFile(resources []resource.ResourceFile, view resource.EffectiveFileView) (interface{}, error) {
	// When template is viewed as EffectiveFile, the most specific template is returned.
	path := view.EffectiveFilePath()

	// Compute requestedLangTag
	matches := templateLanguageTagRegex.FindStringSubmatch(path)
	if len(matches) < 2 {
		return nil, resource.ErrResourceNotFound
	}
	requestedLangTag := matches[1]

	var found bool
	var bytes []byte
	for _, resrc := range resources {
		langTag := templateLanguageTagRegex.FindStringSubmatch(resrc.Location.Path)[1]
		if langTag == requestedLangTag {
			found = true
			bytes = resrc.Data
		}
	}

	if !found {
		return nil, resource.ErrResourceNotFound
	}

	return bytes, nil
}

func viewTemplatesEffectiveResource(resources []resource.ResourceFile, view resource.EffectiveResourceView) (interface{}, error) {
	preferredLanguageTags := view.PreferredLanguageTags()
	defaultLanguageTag := view.DefaultLanguageTag()

	languageTemplates := make(map[string]languageTemplate)
	add := func(langTag string, resrc resource.ResourceFile) error {
		t := languageTemplate{
			languageTag: langTag,
			data:        resrc.Data,
		}
		languageTemplates[langTag] = t
		return nil
	}
	extractLanguageTag := func(resrc resource.ResourceFile) string {
		langTag := templateLanguageTagRegex.FindStringSubmatch(resrc.Location.Path)[1]
		return langTag
	}

	err := intlresource.Prepare(resources, view, extractLanguageTag, add)
	if err != nil {
		return nil, err
	}

	var items []intlresource.LanguageItem
	for _, i := range languageTemplates {
		items = append(items, i)
	}

	matched, err := intlresource.Match(preferredLanguageTags, defaultLanguageTag, items)
	if errors.Is(err, intlresource.ErrNoLanguageMatch) {
		if len(items) > 0 {
			// Use first item in case of no match, to ensure resolution always succeed
			matched = items[0]
		} else {
			// If no configured translation for a template, fail the resolution process
			return nil, ErrNotFound
		}
	} else if err != nil {
		return nil, err
	}

	tagger := matched.(languageTemplate)
	return tagger.data, nil
}

func viewHTMLTemplates(name string, resources []resource.ResourceFile, view resource.View) (interface{}, error) {

	switch view.(type) {
	case resource.AppFileView:
		bytes, err := viewTemplates(resources, view)
		if err != nil {
			return nil, err
		}
		return bytes, nil
	case resource.EffectiveFileView:
		bytes, err := viewTemplates(resources, view)
		if err != nil {
			return nil, err
		}
		return bytes, nil
	case resource.EffectiveResourceView:
		bytes, err := viewTemplates(resources, view)
		if err != nil {
			return nil, err
		}
		tpl := htmltemplate.New(name)
		tpl.Funcs(DefaultFuncMap)
		_, err = tpl.Parse(string(bytes.([]byte)))
		if err != nil {
			return nil, fmt.Errorf("invalid HTML template: %w", err)
		}
		return tpl, nil
	case resource.ValidateResourceView:
		for _, resrc := range resources {
			tpl := htmltemplate.New(name)
			tpl.Funcs(DefaultFuncMap)
			_, err := tpl.Parse(string(resrc.Data))
			if err != nil {
				return nil, fmt.Errorf("invalid HTML template: %w", err)
			}
		}
		return nil, nil
	default:
		return nil, fmt.Errorf("unsupported view: %T", view)
	}

}

func viewTextTemplates(name string, resources []resource.ResourceFile, view resource.View) (interface{}, error) {

	switch view.(type) {
	case resource.AppFileView:
		bytes, err := viewTemplates(resources, view)
		if err != nil {
			return nil, err
		}
		return bytes, nil
	case resource.EffectiveFileView:
		bytes, err := viewTemplates(resources, view)
		if err != nil {
			return nil, err
		}
		return bytes, nil
	case resource.EffectiveResourceView:
		bytes, err := viewTemplates(resources, view)
		if err != nil {
			return nil, err
		}
		tpl := texttemplate.New(name)
		tpl.Funcs(DefaultFuncMap)
		_, err = tpl.Parse(string(bytes.([]byte)))
		if err != nil {
			return nil, fmt.Errorf("invalid text template: %w", err)
		}
		return tpl, nil
	case resource.ValidateResourceView:
		for _, resrc := range resources {
			tpl := texttemplate.New(name)
			tpl.Funcs(DefaultFuncMap)
			_, err := tpl.Parse(string(resrc.Data))
			if err != nil {
				return nil, fmt.Errorf("invalid text template: %w", err)
			}
		}
		return nil, nil
	default:
		return nil, fmt.Errorf("unsupported view: %T", view)
	}

}
