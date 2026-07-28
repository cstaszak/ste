// Package config loads the optional .ste.yml file and applies it to the
// default lint options.
package config

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/cstaszak/ste/internal/lint"
)

// Name is the file the tool looks for.
const Name = ".ste.yml"

// WordList changes one embedded word list.
type WordList struct {
	// Add maps a phrase to its suggested replacement.
	Add map[string]string `yaml:"add"`
	// Remove takes phrases out of the list.
	Remove []string `yaml:"remove"`
	// Replace drops the embedded list and uses only the Add entries.
	Replace bool `yaml:"replace"`
}

// Config is the parsed .ste.yml file. Every field is optional.
type Config struct {
	Mode                  string              `yaml:"mode"`
	MaxPer100W            *float64            `yaml:"max_per_100w"`
	MaxSentenceWords      *int                `yaml:"max_sentence_words"`
	MaxParagraphSentences *int                `yaml:"max_paragraph_sentences"`
	Rules                 map[string]bool     `yaml:"rules"`
	Words                 map[string]WordList `yaml:"words"`
	// Allow lists technical names that the dictionary check accepts.
	Allow []string `yaml:"allow"`

	path string
}

// Path reports the file the configuration came from. It is empty for defaults.
func (c *Config) Path() string { return c.path }

// Find looks for .ste.yml in dir and every parent directory.
func Find(dir string) string {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return ""
	}
	for {
		p := filepath.Join(abs, Name)
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
		parent := filepath.Dir(abs)
		if parent == abs {
			return ""
		}
		abs = parent
	}
}

// Load reads a configuration file. An empty path returns the defaults.
func Load(path string) (*Config, error) {
	c := &Config{}
	if path == "" {
		return c, nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	dec := yaml.NewDecoder(bytes.NewReader(b))
	dec.KnownFields(true)
	if err := dec.Decode(c); err != nil && err != io.EOF {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	c.path = path
	if c.Mode != "" {
		if _, err := lint.ParseMode(c.Mode); err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
	}
	for id := range c.Rules {
		if _, ok := lint.Lookup(id); !ok {
			return nil, fmt.Errorf("%s: unknown rule %q", path, id)
		}
	}
	for id := range c.Words {
		if _, ok := lint.Lookup(id); !ok {
			return nil, fmt.Errorf("%s: unknown rule %q in words", path, id)
		}
	}
	return c, nil
}

// Mode returns the mode to use, given the flag value the user passed.
// The flag wins over the file.
func (c *Config) ResolveMode(flag string) (lint.Mode, error) {
	s := flag
	if s == "" {
		s = c.Mode
	}
	if s == "" {
		return lint.ModeFlavored, nil
	}
	return lint.ParseMode(s)
}

// Apply changes the options to match the configuration.
func (c *Config) Apply(opt lint.Options) lint.Options {
	if c.MaxSentenceWords != nil {
		opt.MaxSentenceWords = *c.MaxSentenceWords
	}
	if c.MaxParagraphSentences != nil {
		opt.MaxParagraphSentences = *c.MaxParagraphSentences
	}
	for id, on := range c.Rules {
		opt.Enabled[id] = on
	}
	for id, w := range c.Words {
		list := opt.Lists[id]
		if list == nil || w.Replace {
			list = lint.NewPhraseList()
		}
		for _, phrase := range w.Remove {
			list.Remove(phrase)
		}
		for phrase, suggest := range w.Add {
			list.Add(phrase, suggest)
		}
		opt.Lists[id] = list
	}
	return opt
}

// Threshold returns the score gate, and whether the file set one.
func (c *Config) Threshold() (float64, bool) {
	if c.MaxPer100W == nil {
		return 0, false
	}
	return *c.MaxPer100W, true
}
