package templateloader

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"
)

type state struct {
	l      *Loader
	ctx    context.Context
	files  map[string]string
	active map[string]bool
}

// template loads one template, mirroring Python get_template_contents.
func (s *state) template(source Source) (map[string]any, error) {
	isObject := source.Object != ""
	var location string
	switch {
	case source.File != "":
		var err error
		if location, err = normaliseFilePath(source.File); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrTemplate, err)
		}
	case source.URL != "":
		location = source.URL
	default:
		location = source.Object
	}
	if s.active[location] {
		return nil, fmt.Errorf("%w: template %s references itself", ErrTemplate, location)
	}
	s.active[location] = true
	defer delete(s.active, location)

	var data []byte
	var err error
	if isObject {
		data, err = s.l.object(s.ctx, location)
	} else {
		data, err = s.l.fetch(s.ctx, location)
	}
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("%w: Could not fetch template from %s", ErrTemplate, location)
	}
	if !utf8.Valid(data) {
		return nil, fmt.Errorf("%w: Error parsing template %s: content is not valid UTF-8", ErrTemplate, location)
	}
	template, err := parseTemplate(string(data))
	if err != nil {
		return nil, fmt.Errorf("%w: Error parsing template %s %v", ErrTemplate, location, err)
	}
	base, err := baseURL(location)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTemplate, err)
	}
	if err := s.resolveGetFiles(template, base, isObject); err != nil {
		return nil, err
	}
	return template, nil
}

func (s *state) resolveGetFiles(template map[string]any, base string, isObject bool) error {
	ignore := func(key string, value any) bool {
		text, ok := value.(string)
		if key != "get_file" && key != "type" || !ok {
			return true
		}
		return key == "type" && !strings.HasSuffix(text, ".yaml") && !strings.HasSuffix(text, ".template")
	}
	return s.getFileContents(template, base, ignore, true, isObject)
}

// getFileContents mirrors Python get_file_contents: nested containers are
// visited first (when recurse is set), then the mapping's own entries.
func (s *state) getFileContents(data any, base string, ignore func(string, any) bool, recurse, isObject bool) error {
	if err := s.ctx.Err(); err != nil {
		return err
	}
	if recurse {
		switch typed := data.(type) {
		case map[string]any:
			for _, child := range typed {
				if err := s.getFileContents(child, base, ignore, recurse, isObject); err != nil {
					return err
				}
			}
		case []any:
			for _, child := range typed {
				if err := s.getFileContents(child, base, ignore, recurse, isObject); err != nil {
					return err
				}
			}
		}
	}
	mapping, ok := data.(map[string]any)
	if !ok {
		return nil
	}
	for key, value := range mapping {
		if ignore != nil && ignore(key, value) {
			continue
		}
		reference, ok := value.(string)
		if !ok {
			return fmt.Errorf("%w: %q must reference a file by string, not %T", ErrTemplate, key, value)
		}
		if base != "" && !strings.HasSuffix(base, "/") {
			base += "/"
		}
		target, err := join(base, reference)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrTemplate, err)
		}
		if _, known := s.files[target]; !known {
			if err := s.loadFile(target, isObject); err != nil {
				return err
			}
		}
		mapping[key] = target
	}
	return nil
}

func (s *state) loadFile(target string, isObject bool) error {
	var data []byte
	var err error
	if isObject && s.l.objects != nil {
		data, err = s.l.object(s.ctx, target)
	} else {
		data, err = s.l.fetch(s.ctx, target)
		if err != nil && s.ctx.Err() == nil {
			err = fmt.Errorf("%w: Could not fetch contents for %s: %w", ErrTemplate, target, err)
		}
	}
	if err != nil {
		return err
	}
	content := fileContent(data)
	if isTemplate(content) {
		source := Source{URL: target}
		if isObject {
			source = Source{Object: target}
		}
		nested, err := s.template(source)
		if err != nil {
			return err
		}
		if content, err = dumpJSON(nested); err != nil {
			return fmt.Errorf("%w: %v", ErrTemplate, err)
		}
	}
	s.files[target] = content
	return nil
}

// environment loads one environment file and its resource registry files.
func (s *state) environment(envPath string) (map[string]any, error) {
	if envPath == "" {
		return map[string]any{}, nil
	}
	location, err := normaliseFilePath(envPath)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTemplate, err)
	}
	base, err := baseURL(location)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTemplate, err)
	}
	data, err := s.l.fetch(s.ctx, location)
	if err != nil {
		return nil, err
	}
	env, err := parseEnvironment(data)
	if err != nil {
		return nil, fmt.Errorf("%w: Error parsing environment %s: %v", ErrTemplate, location, err)
	}
	if err := s.resolveEnvironmentURLs(env["resource_registry"], base); err != nil {
		return nil, err
	}
	return env, nil
}

func (s *state) resolveEnvironmentURLs(value any, envBase string) error {
	if value == nil {
		return nil
	}
	registry, ok := value.(map[string]any)
	if !ok {
		return fmt.Errorf("%w: resource_registry must be a mapping, not %T", ErrTemplate, value)
	}
	base, err := registryBase(registry, envBase)
	if err != nil {
		return err
	}
	ignore := func(key string, value any) bool {
		if key == "base_url" || key == "hooks" || key == "restricted_actions" {
			return true
		}
		if _, nested := value.(map[string]any); nested {
			return true
		}
		text, isString := value.(string)
		return isString && strings.Contains(text, "::")
	}
	if err := s.getFileContents(registry, base, ignore, false, false); err != nil {
		return err
	}
	resources, _ := registry["resources"].(map[string]any)
	for name, entry := range resources {
		mapping, ok := entry.(map[string]any)
		if !ok {
			return fmt.Errorf("%w: resource_registry resource %q must be a mapping, not %T", ErrTemplate, name, entry)
		}
		resourceBase, err := registryBase(mapping, base)
		if err != nil {
			return err
		}
		if err := s.getFileContents(mapping, resourceBase, ignore, false, false); err != nil {
			return err
		}
	}
	return nil
}

func registryBase(mapping map[string]any, fallback string) (string, error) {
	value, present := mapping["base_url"]
	if !present {
		return fallback, nil
	}
	text, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("%w: base_url must be a string, not %T", ErrTemplate, value)
	}
	return text, nil
}
