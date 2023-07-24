package existingfileparser

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/getsyncer/syncer/sharedapi/files"
)

const RecommendedSectionStart = "THIS SECTION IS AUTOGENERATED BY SYNCER, DO NOT EDIT"
const RecommendedSectionEnd = "END OF AUTOGENERATED SECTION BY SYNCER"

func RecommendedNewlineSeparatedConfig() ParseConfig {
	return ParseConfig{
		SplitBy:       "\n",
		StartSection:  ContainsSubstring(RecommendedSectionStart),
		EndSection:    ContainsSubstring(RecommendedSectionEnd),
		SectionTrim:   strings.TrimSpace,
		SectionSorter: sort.Strings,
	}
}

type ParseResult struct {
	State          *files.State
	PreAutogenMsg  string
	AutogenMsg     string
	PostAutogenMsg string
}

type ParseConfig struct {
	SplitBy       string
	StartSection  func(string) bool
	EndSection    func(string) bool
	SectionTrim   func(string) string
	SectionSorter func([]string)
}

func ContainsSubstring(substring string) func(string) bool {
	return func(s string) bool {
		return strings.Contains(s, substring)
	}
}

func Parse(ctx context.Context, loader files.StateLoader, path files.Path, conf ParseConfig) (*ParseResult, error) {
	currentState, err := loader.LoadState(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("failed to load state for %s: %w", path, err)
	}
	parts := strings.Split(string(currentState.Contents), conf.SplitBy)
	if conf.SectionTrim != nil {
		for idx := range parts {
			parts[idx] = conf.SectionTrim(parts[idx])
		}
	}
	startIndex, endIndex := -1, -1
	for idx, part := range parts {
		if startIndex == -1 && conf.StartSection(part) {
			startIndex = idx
			continue
		}
		if startIndex != -1 && conf.EndSection(part) {
			endIndex = idx
		}
	}
	if endIndex == -1 {
		return &ParseResult{
			State:          currentState,
			PreAutogenMsg:  "",
			AutogenMsg:     "",
			PostAutogenMsg: string(currentState.Contents),
		}, nil
	}
	if startIndex == -1 {
		panic("invalid state")
	}
	preSection := parts[:startIndex]
	autoSection := parts[startIndex : endIndex+1]
	postSection := parts[endIndex+1:]
	if conf.SectionSorter != nil {
		conf.SectionSorter(preSection)
		conf.SectionSorter(autoSection)
		conf.SectionSorter(postSection)
	}
	return &ParseResult{
		State:          currentState,
		PreAutogenMsg:  strings.Join(preSection, conf.SplitBy),
		AutogenMsg:     strings.Join(autoSection, conf.SplitBy),
		PostAutogenMsg: strings.Join(postSection, conf.SplitBy),
	}, nil
}
