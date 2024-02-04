package utils

import (
	"fmt"

	"olympos.io/encoding/edn"
)

type GraphConfig struct {
	JournalsDir string        `edn:"journals-directory"`
	Journal     JournalConfig `edn:"journal"`

	PagesDir string `edn:"pages-directory"`

	File FileConfig `edn:"file"`

	DefaultTemplates DefaultTemplates `edn:"default-templates"`

	Property PropertyConfig `edn:"property"`

	IgnoredPageReferencesKeywords []string `edn:"ignored-page-references-keywords"`
}

type JournalConfig struct {
	PageTitleFormat string `edn:"page-title-format"`

	FileNameFormat string `edn:"file-name-format"`
}

type FileConfig struct {
	NameFormat FilenameFormat `edn:"name-format"`
}

type DefaultTemplates struct {
	Journals string `edn:"journals"`
}

type PropertyConfig struct {
	SeparatedByCommas []string `edn:"separated-by-commas"`
}

func ParseConfig(data []byte) (*GraphConfig, error) {
	config := GraphConfig{
		JournalsDir: "journals",
		Journal: JournalConfig{
			PageTitleFormat: "EEE do, MMM yyyy",
			FileNameFormat:  "yyyy_MM_dd",
		},
		PagesDir: "pages",
		File: FileConfig{
			NameFormat: FilenameFormatTripleLowbar,
		},
	}

	err := edn.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to read EDN: %w", err)
	}

	return &config, nil
}
