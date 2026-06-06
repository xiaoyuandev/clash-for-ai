package extension

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
)

type Service struct {
	repository Repository
	sources    []ScanSource
	bundled    []BundledPlugin
	homeDir    string
}

func NewService(repository Repository, sources []ScanSource, bundled ...BundledPlugin) *Service {
	homeDir, _ := os.UserHomeDir()
	return &Service{
		repository: repository,
		sources:    append([]ScanSource(nil), sources...),
		bundled:    append([]BundledPlugin(nil), bundled...),
		homeDir:    homeDir,
	}
}

func (s *Service) List(ctx context.Context) ([]Plugin, error) {
	return s.repository.List(ctx)
}

func (s *Service) GetByID(ctx context.Context, id string) (*Plugin, error) {
	return s.repository.GetByID(ctx, id)
}

func (s *Service) Enable(ctx context.Context, id string) (*Plugin, error) {
	return s.repository.SetEnabled(ctx, id, true)
}

func (s *Service) Disable(ctx context.Context, id string) (*Plugin, error) {
	return s.repository.SetEnabled(ctx, id, false)
}

func (s *Service) Scan(ctx context.Context) ([]Plugin, error) {
	for _, source := range s.sources {
		if err := s.scanSource(ctx, source); err != nil {
			return nil, err
		}
	}
	if err := s.scanBundled(ctx); err != nil {
		return nil, err
	}
	return s.repository.List(ctx)
}

func (s *Service) scanBundled(ctx context.Context) error {
	if len(s.bundled) == 0 {
		return nil
	}

	seenIDs := make([]string, 0, len(s.bundled))
	for _, bundled := range s.bundled {
		manifest := bundled.Manifest
		warnings, err := ValidateManifest(manifest)
		item := manifestToPlugin(manifest, PluginScopeBundled, bundled.ManifestPath, warnings)
		if err != nil {
			item = invalidPluginFromManifest(manifest, PluginScopeBundled, bundled.ManifestPath, err)
		}

		if existing, err := s.repository.GetByID(ctx, item.ID); err == nil {
			item.CreatedAt = existing.CreatedAt
			if item.Status == PluginStatusInvalid {
				item.Enabled = false
			} else {
				item.Enabled = existing.Enabled
				item.Status = pluginStatusForEnabled(existing.Enabled, existing.Status)
			}
		} else if !errors.Is(err, ErrPluginNotFound) {
			return err
		}

		if _, err := s.repository.Upsert(ctx, item); err != nil {
			return err
		}
		seenIDs = append(seenIDs, item.ID)
	}

	sort.Strings(seenIDs)
	return s.repository.MarkMissing(ctx, PluginScopeBundled, seenIDs)
}

func (s *Service) scanSource(ctx context.Context, source ScanSource) error {
	paths, err := manifestPaths(source.Dir)
	if err != nil {
		return err
	}

	loaded := make(map[string]Plugin)
	duplicates := make(map[string][]string)

	for _, manifestPath := range paths {
		manifest, warnings, err := LoadManifest(manifestPath)
		if err != nil {
			if manifest.ID == "" {
				log.Printf("[extensions] skipping invalid manifest %s: %v", manifestPath, err)
				continue
			}
			loaded[manifest.ID] = invalidPluginFromManifest(manifest, source.Scope, manifestPath, err)
			continue
		}

		if existing, ok := loaded[manifest.ID]; ok {
			duplicates[manifest.ID] = append(duplicates[manifest.ID], existing.ManifestPath, manifestPath)
			loaded[manifest.ID] = invalidPluginFromManifest(
				manifest,
				source.Scope,
				existing.ManifestPath,
				fmt.Errorf("duplicate plugin id %q in %s and %s", manifest.ID, existing.ManifestPath, manifestPath),
			)
			continue
		}

		loaded[manifest.ID] = manifestToPlugin(manifest, source.Scope, manifestPath, warnings)
	}

	seenIDs := make([]string, 0, len(loaded))
	for _, item := range loaded {
		if duplicatePaths, ok := duplicates[item.ID]; ok && len(duplicatePaths) > 1 {
			item.LastError = fmt.Sprintf("duplicate plugin id %q in %s", item.ID, duplicatePaths[0])
			item.Status = PluginStatusInvalid
			item.Enabled = false
		}

		if existing, err := s.repository.GetByID(ctx, item.ID); err == nil {
			item.CreatedAt = existing.CreatedAt
			if item.Status == PluginStatusInvalid {
				item.Enabled = false
			} else {
				item.Enabled = existing.Enabled
				item.Status = pluginStatusForEnabled(existing.Enabled, existing.Status)
			}
		} else if !errors.Is(err, ErrPluginNotFound) {
			return err
		}

		if _, err := s.repository.Upsert(ctx, item); err != nil {
			return err
		}
		seenIDs = append(seenIDs, item.ID)
	}

	sort.Strings(seenIDs)
	return s.repository.MarkMissing(ctx, source.Scope, seenIDs)
}

func manifestPaths(root string) ([]string, error) {
	if root == "" {
		return nil, nil
	}
	info, err := os.Stat(root)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("inspect extension directory: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("extension path is not a directory: %s", root)
	}

	paths := []string{}
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if entry.Name() == ManifestFileName {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan extension directory: %w", err)
	}
	sort.Strings(paths)
	return paths, nil
}

func invalidPluginFromManifest(manifest Manifest, scope PluginScope, manifestPath string, err error) Plugin {
	item := manifestToPlugin(manifest, scope, manifestPath, ManifestWarnings(manifest))
	item.Status = PluginStatusInvalid
	item.Enabled = false
	item.LastError = err.Error()
	return item
}
