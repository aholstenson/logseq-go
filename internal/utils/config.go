package utils

import (
	"fmt"

	"olympos.io/encoding/edn"
)

type GraphConfig struct {
	JournalsDir string        `edn:"journals-directory"`
	Journal     JournalConfig `edn:"journal"`

	File FileConfig `edn:"file"`

	DefaultTemplates DefaultTemplates `edn:"default-templates"`
}

type JournalConfig struct {
	FileNameFormat string `edn:"file-name-format"`
}

type FileConfig struct {
	NameFormat FilenameFormat `edn:"name-format"`
}

type DefaultTemplates struct {
	Journals string `edn:"journals"`
}

func ParseConfig(data []byte) (*GraphConfig, error) {
	config := GraphConfig{
		JournalsDir: "journals",
		Journal: JournalConfig{
			FileNameFormat: "yyyy_MM_dd",
		},
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
