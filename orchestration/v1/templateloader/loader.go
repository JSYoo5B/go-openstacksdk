// Package templateloader reads Heat templates and environment files and
// resolves the files they reference. It reimplements the behavior of the
// pinned Python openstacksdk template_utils helpers (Apache-2.0, originally
// from python-heatclient) that back Proxy.get_template_contents and
// Proxy.read_env_and_templates; see README.md for attribution and differences.
//
// A Loader performs no authenticated OpenStack request. Like the Python
// helpers it reads local files and anonymous HTTP(S) URLs only.
package templateloader

import (
	"context"
	"errors"
	"fmt"
	"net/http"
)

// ErrTemplate is wrapped by every template, environment and file processing
// failure.
var ErrTemplate = errors.New("heat template processing failed")

// ObjectReader fetches the content of a template or file stored in an object
// store. Python's object_request callable is the equivalent. The loader never
// creates one: without a reader, a template object cannot be fetched.
type ObjectReader func(ctx context.Context, url string) ([]byte, error)

// Loader reads templates and environments. The zero value is not usable; use New.
type Loader struct {
	client   *http.Client
	objects  ObjectReader
	maxBytes int64
}

// Option configures a Loader.
type Option func(*Loader) error

// WithHTTPClient selects the client used for http and https URLs. The default
// is a copy of http.DefaultClient. The client never receives OpenStack tokens.
func WithHTTPClient(client *http.Client) Option {
	return func(l *Loader) error {
		if client == nil {
			return fmt.Errorf("%w: HTTP client is required", ErrTemplate)
		}
		l.client = client
		return nil
	}
}

// WithObjectReader supplies the reader used for template objects and for files
// referenced from them.
func WithObjectReader(reader ObjectReader) Option {
	return func(l *Loader) error {
		if reader == nil {
			return fmt.Errorf("%w: object reader is required", ErrTemplate)
		}
		l.objects = reader
		return nil
	}
}

// WithMaxFileBytes rejects a template, environment or referenced file larger
// than the limit. Python has no limit; zero (the default) means unlimited.
func WithMaxFileBytes(limit int64) Option {
	return func(l *Loader) error {
		if limit < 0 {
			return fmt.Errorf("%w: file size limit must not be negative", ErrTemplate)
		}
		l.maxBytes = limit
		return nil
	}
}

// New creates a Loader.
func New(options ...Option) (*Loader, error) {
	l := &Loader{client: &http.Client{}}
	for _, option := range options {
		if option == nil {
			return nil, fmt.Errorf("%w: nil option", ErrTemplate)
		}
		if err := option(l); err != nil {
			return nil, err
		}
	}
	return l, nil
}

// Source selects exactly one template location. File is a local path (or a
// URL, which is used as is), URL is an http, https or file URL and Object is
// fetched through the Loader's ObjectReader.
type Source struct {
	File   string
	URL    string
	Object string
}

func (s Source) count() int {
	n := 0
	for _, v := range []string{s.File, s.URL, s.Object} {
		if v != "" {
			n++
		}
	}
	return n
}

// Template is the result of GetTemplateContents. Files maps the absolute URL
// of every referenced file to its content; Content is the parsed template
// whose get_file and type references were rewritten to those URLs.
type Template struct {
	Files   map[string]string
	Content map[string]any
}

// StackInput selects the inputs of ReadEnvAndTemplates.
type StackInput struct {
	Template         Source
	Files            map[string]string
	EnvironmentFiles []string
}

// StackAttributes are the stack attributes produced by ReadEnvAndTemplates. A
// nil field means that Python would not set the attribute. Environment is
// non-nil (possibly empty) when environment files were requested, Template is
// non-nil when a template source was requested, and Files is non-nil only when
// a template source was requested and some file was collected.
type StackAttributes struct {
	Environment map[string]any
	Template    map[string]any
	Files       map[string]string
}

// GetTemplateContents loads a template and resolves its file references. The
// supplied files map is copied, never modified. Every failure is wrapped as
// "Error in processing template files", like the Python proxy method.
func (l *Loader) GetTemplateContents(ctx context.Context, source Source, files map[string]string) (*Template, error) {
	result, err := l.getTemplateContents(ctx, source, files)
	if err != nil {
		return nil, fmt.Errorf("Error in processing template files: %w", err)
	}
	return result, nil
}

func (l *Loader) getTemplateContents(ctx context.Context, source Source, files map[string]string) (*Template, error) {
	if l == nil || l.client == nil {
		return nil, fmt.Errorf("%w: use New to create a Loader", ErrTemplate)
	}
	if ctx == nil {
		return nil, fmt.Errorf("%w: context is required", ErrTemplate)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if source.count() == 0 {
		return nil, fmt.Errorf("%w: Must provide one of template_file, template_url or template_object", ErrTemplate)
	}
	if source.count() > 1 {
		return nil, fmt.Errorf("%w: only one of template file, URL or object may be provided", ErrTemplate)
	}
	st := &state{l: l, ctx: ctx, files: map[string]string{}, active: map[string]bool{}}
	for key, value := range files {
		st.files[key] = value
	}
	content, err := st.template(source)
	if err != nil {
		return nil, err
	}
	return &Template{Files: st.files, Content: content}, nil
}

// ReadEnvAndTemplates reads the environment files, then the template, and
// returns the stack attributes Python Proxy.read_env_and_templates builds.
// As in Python, environment files collected from resource registries are
// returned only together with a template, and a file collected from the
// environments replaces a template file with the same URL.
func (l *Loader) ReadEnvAndTemplates(ctx context.Context, input StackInput) (*StackAttributes, error) {
	if l == nil || l.client == nil {
		return nil, fmt.Errorf("%w: use New to create a Loader", ErrTemplate)
	}
	if ctx == nil {
		return nil, fmt.Errorf("%w: context is required", ErrTemplate)
	}
	attrs := &StackAttributes{}
	var envFiles map[string]string
	if len(input.EnvironmentFiles) > 0 {
		var err error
		envFiles, attrs.Environment, err = l.ProcessEnvironments(ctx, input.EnvironmentFiles)
		if err != nil {
			return nil, err
		}
	}
	if input.Template.count() > 0 {
		template, err := l.getTemplateContents(ctx, input.Template, input.Files)
		if err != nil {
			return nil, err
		}
		attrs.Template = template.Content
		if len(template.Files) > 0 || len(envFiles) > 0 {
			attrs.Files = template.Files
			for key, value := range envFiles {
				attrs.Files[key] = value
			}
		}
	}
	return attrs, nil
}

// ProcessEnvironments reads one or more environment files, merges them in
// order with deep mapping updates and returns the files referenced from their
// resource registries together with the merged environment.
func (l *Loader) ProcessEnvironments(ctx context.Context, paths []string) (map[string]string, map[string]any, error) {
	if l == nil || l.client == nil {
		return nil, nil, fmt.Errorf("%w: use New to create a Loader", ErrTemplate)
	}
	if ctx == nil {
		return nil, nil, fmt.Errorf("%w: context is required", ErrTemplate)
	}
	merged, mergedFiles := map[string]any{}, map[string]string{}
	for _, path := range paths {
		if err := ctx.Err(); err != nil {
			return nil, nil, err
		}
		st := &state{l: l, ctx: ctx, files: map[string]string{}, active: map[string]bool{}}
		env, err := st.environment(path)
		if err != nil {
			return nil, nil, err
		}
		for key, value := range st.files {
			mergedFiles[key] = value
		}
		deepUpdate(merged, env)
	}
	return mergedFiles, merged, nil
}
