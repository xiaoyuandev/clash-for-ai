package extension

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var githubPathPattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+(?:\.git)?$`)

func (s *Service) Install(ctx context.Context, input InstallPluginInput) (*Plugin, error) {
	sourceType, sourceURL, err := normalizePluginSource(input)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(s.managedInstallDir) == "" {
		return nil, fmt.Errorf("%w: managed install directory is not configured", ErrInvalidPluginSource)
	}
	if _, err := s.repository.GetInstallBySource(ctx, string(sourceType), sourceURL); err == nil {
		return nil, ErrPluginAlreadyInstalled
	} else if !errors.Is(err, ErrPluginNotFound) {
		return nil, err
	}
	if err := os.MkdirAll(s.managedInstallDir, 0o755); err != nil {
		return nil, fmt.Errorf("create managed plugin directory: %w", err)
	}

	tempDir := filepath.Join(
		s.managedInstallDir,
		fmt.Sprintf(".install-%d-%s", time.Now().UTC().UnixNano(), sourceDirectorySuffix(sourceURL)),
	)
	if err := s.git(ctx, "", "clone", "--depth", "1", sourceURL, tempDir); err != nil {
		_ = os.RemoveAll(tempDir)
		return nil, err
	}
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	manifestPath, manifest, warnings, err := loadSingleManagedManifest(tempDir)
	if err != nil {
		return nil, err
	}
	if existing, err := s.repository.GetByID(ctx, manifest.ID); err == nil {
		if existing.Install != nil && existing.Install.SourceURL == sourceURL {
			return nil, ErrPluginAlreadyInstalled
		}
		return nil, fmt.Errorf("%w: plugin id %q is already present", ErrPluginAlreadyInstalled, manifest.ID)
	} else if !errors.Is(err, ErrPluginNotFound) {
		return nil, err
	}

	targetDir := filepath.Join(s.managedInstallDir, managedDirectoryName(manifest.ID, sourceURL))
	if _, err := os.Stat(targetDir); err == nil {
		return nil, fmt.Errorf("%w: install directory already exists", ErrPluginAlreadyInstalled)
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("inspect install directory: %w", err)
	}
	if err := s.prepareManagedInstall(ctx, tempDir, manifest); err != nil {
		return nil, err
	}
	if err := os.Rename(tempDir, targetDir); err != nil {
		return nil, fmt.Errorf("move managed plugin into place: %w", err)
	}

	relativeManifestPath, err := filepath.Rel(tempDir, manifestPath)
	if err != nil {
		return nil, fmt.Errorf("resolve manifest path: %w", err)
	}
	finalManifestPath := filepath.Join(targetDir, relativeManifestPath)
	commit, err := s.gitOutput(ctx, targetDir, "rev-parse", "HEAD")
	if err != nil {
		commit = ""
	}

	item := manifestToPlugin(manifest, PluginScopeManaged, finalManifestPath, warnings)
	item.Enabled = true
	item.Status = PluginStatusEnabled
	stored, err := s.repository.Upsert(ctx, item)
	if err != nil {
		_ = os.RemoveAll(targetDir)
		return nil, err
	}

	install, err := s.repository.UpsertInstall(ctx, PluginInstall{
		PluginID:   stored.ID,
		SourceType: string(sourceType),
		SourceURL:  sourceURL,
		InstallDir: targetDir,
		GitCommit:  strings.TrimSpace(commit),
	})
	if err != nil {
		_ = s.repository.DeletePlugin(ctx, stored.ID)
		_ = os.RemoveAll(targetDir)
		return nil, err
	}
	stored.Install = &install
	s.startRuntime(ctx, stored)
	decorated := s.decoratePlugin(stored)
	return &decorated, nil
}

func (s *Service) LocalInstall(ctx context.Context, input LocalInstallPluginInput) (*Plugin, error) {
	enabled, err := s.isDeveloperModeEnabled(ctx)
	if err != nil {
		return nil, err
	}
	if !enabled {
		return nil, ErrDeveloperModeDisabled
	}

	sourceURL, err := normalizeLocalDirectory(input.Path)
	if err != nil {
		return nil, err
	}
	if existingInstall, err := s.repository.GetInstallBySource(ctx, string(PluginSourceLocalDirectory), sourceURL); err == nil {
		return s.updateLocalDirectoryInstall(ctx, existingInstall.PluginID, sourceURL)
	} else if !errors.Is(err, ErrPluginNotFound) {
		return nil, err
	}

	manifestPath, manifest, warnings, err := loadSingleManagedManifest(sourceURL)
	if err != nil {
		return nil, err
	}
	if existing, err := s.repository.GetByID(ctx, manifest.ID); err == nil {
		if existing.Install != nil && existing.Install.SourceType == string(PluginSourceLocalDirectory) && existing.Install.SourceURL == sourceURL {
			return s.updateLocalDirectoryInstall(ctx, manifest.ID, sourceURL)
		}
		return nil, fmt.Errorf("%w: plugin id %q is already present", ErrPluginAlreadyInstalled, manifest.ID)
	} else if !errors.Is(err, ErrPluginNotFound) {
		return nil, err
	}

	item := manifestToPlugin(manifest, PluginScopeDevelopment, manifestPath, warnings)
	item.Enabled = true
	item.Status = PluginStatusEnabled
	stored, err := s.repository.Upsert(ctx, item)
	if err != nil {
		return nil, err
	}

	install, err := s.repository.UpsertInstall(ctx, PluginInstall{
		PluginID:   stored.ID,
		SourceType: string(PluginSourceLocalDirectory),
		SourceURL:  sourceURL,
		InstallDir: sourceURL,
		GitCommit:  "",
	})
	if err != nil {
		_ = s.repository.DeletePlugin(ctx, stored.ID)
		return nil, err
	}
	stored.Install = &install
	s.startRuntime(ctx, stored)
	decorated := s.decoratePlugin(stored)
	return &decorated, nil
}

func (s *Service) Update(ctx context.Context, pluginID string) (*Plugin, error) {
	plugin, err := s.repository.GetByID(ctx, pluginID)
	if err != nil {
		return nil, err
	}
	install, err := s.repository.GetInstallByPluginID(ctx, pluginID)
	if err != nil {
		if errors.Is(err, ErrPluginNotFound) {
			return nil, ErrPluginNotManaged
		}
		return nil, err
	}
	if strings.TrimSpace(install.InstallDir) == "" {
		return nil, ErrPluginNotManaged
	}
	if install.SourceType == string(PluginSourceLocalDirectory) {
		return s.updateLocalDirectoryInstall(ctx, pluginID, install.InstallDir)
	}
	if s.runtimeHost != nil {
		_ = s.runtimeHost.Stop(ctx, pluginID)
	}

	if err := s.git(ctx, install.InstallDir, "pull", "--ff-only"); err != nil {
		return nil, err
	}
	manifestPath, manifest, warnings, err := loadSingleManagedManifest(install.InstallDir)
	if err != nil {
		return nil, err
	}
	if manifest.ID != pluginID {
		return nil, fmt.Errorf("%w: updated manifest id changed from %q to %q", ErrInvalidPluginSource, pluginID, manifest.ID)
	}
	if err := s.prepareManagedInstall(ctx, install.InstallDir, manifest); err != nil {
		return nil, err
	}
	commit, err := s.gitOutput(ctx, install.InstallDir, "rev-parse", "HEAD")
	if err != nil {
		commit = install.GitCommit
	}

	item := manifestToPlugin(manifest, PluginScopeManaged, manifestPath, warnings)
	item.CreatedAt = plugin.CreatedAt
	if item.Status == PluginStatusInvalid {
		item.Enabled = false
	} else {
		item.Enabled = plugin.Enabled
		item.Status = pluginStatusForEnabled(plugin.Enabled, plugin.Status)
	}
	stored, err := s.repository.Upsert(ctx, item)
	if err != nil {
		return nil, err
	}

	install.GitCommit = strings.TrimSpace(commit)
	updatedInstall, err := s.repository.UpsertInstall(ctx, *install)
	if err != nil {
		return nil, err
	}
	stored.Install = &updatedInstall
	s.startRuntime(ctx, stored)
	decorated := s.decoratePlugin(stored)
	return &decorated, nil
}

func (s *Service) Uninstall(ctx context.Context, pluginID string) error {
	plugin, err := s.repository.GetByID(ctx, pluginID)
	if err != nil {
		return err
	}
	install, err := s.repository.GetInstallByPluginID(ctx, plugin.ID)
	if err != nil {
		if errors.Is(err, ErrPluginNotFound) {
			return ErrPluginNotManaged
		}
		return err
	}
	if s.runtimeHost != nil {
		_ = s.runtimeHost.Stop(ctx, plugin.ID)
	}

	if err := s.repository.DeletePlugin(ctx, plugin.ID); err != nil {
		return err
	}
	if strings.TrimSpace(install.InstallDir) != "" {
		if install.SourceType != string(PluginSourceLocalDirectory) {
			if err := os.RemoveAll(install.InstallDir); err != nil {
				return fmt.Errorf("remove plugin install directory: %w", err)
			}
		}
	}
	if strings.TrimSpace(s.pluginDataDir) != "" {
		if err := os.RemoveAll(filepath.Join(s.pluginDataDir, plugin.ID)); err != nil {
			return fmt.Errorf("remove plugin data directory: %w", err)
		}
	}
	return nil
}

func (s *Service) updateLocalDirectoryInstall(ctx context.Context, pluginID string, installDir string) (*Plugin, error) {
	enabled, err := s.isDeveloperModeEnabled(ctx)
	if err != nil {
		return nil, err
	}
	if !enabled {
		return nil, ErrDeveloperModeDisabled
	}
	sourceURL, err := normalizeLocalDirectory(installDir)
	if err != nil {
		return nil, err
	}

	plugin, err := s.repository.GetByID(ctx, pluginID)
	if err != nil {
		return nil, err
	}
	if plugin.Install == nil || plugin.Install.SourceType != string(PluginSourceLocalDirectory) {
		return nil, ErrPluginNotManaged
	}
	if plugin.Install.SourceURL != sourceURL {
		return nil, fmt.Errorf("%w: local plugin source changed", ErrInvalidPluginSource)
	}
	if s.runtimeHost != nil {
		_ = s.runtimeHost.Stop(ctx, pluginID)
	}

	manifestPath, manifest, warnings, err := loadSingleManagedManifest(sourceURL)
	if err != nil {
		return nil, err
	}
	if manifest.ID != pluginID {
		return nil, fmt.Errorf("%w: local manifest id changed from %q to %q", ErrInvalidPluginSource, pluginID, manifest.ID)
	}

	item := manifestToPlugin(manifest, PluginScopeDevelopment, manifestPath, warnings)
	item.CreatedAt = plugin.CreatedAt
	if item.Status == PluginStatusInvalid {
		item.Enabled = false
	} else {
		item.Enabled = plugin.Enabled
		item.Status = pluginStatusForEnabled(plugin.Enabled, plugin.Status)
	}
	stored, err := s.repository.Upsert(ctx, item)
	if err != nil {
		return nil, err
	}

	updatedInstall, err := s.repository.UpsertInstall(ctx, PluginInstall{
		PluginID:    stored.ID,
		SourceType:  string(PluginSourceLocalDirectory),
		SourceURL:   sourceURL,
		InstallDir:  sourceURL,
		GitCommit:   "",
		InstalledAt: plugin.Install.InstalledAt,
	})
	if err != nil {
		return nil, err
	}
	stored.Install = &updatedInstall
	s.startRuntime(ctx, stored)
	decorated := s.decoratePlugin(stored)
	return &decorated, nil
}

func normalizePluginSource(input InstallPluginInput) (PluginSourceType, string, error) {
	if strings.TrimSpace(input.Source) != string(PluginSourceGitHub) {
		return "", "", fmt.Errorf("%w: source must be github", ErrInvalidPluginSource)
	}
	parsed, err := url.Parse(strings.TrimSpace(input.URL))
	if err != nil {
		return "", "", fmt.Errorf("%w: malformed GitHub URL", ErrInvalidPluginSource)
	}
	if parsed.Scheme != "https" || !strings.EqualFold(parsed.Host, "github.com") || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", "", fmt.Errorf("%w: only public https://github.com/{owner}/{repo} URLs are supported", ErrInvalidPluginSource)
	}
	path := strings.Trim(parsed.EscapedPath(), "/")
	unescaped, err := url.PathUnescape(path)
	if err != nil {
		return "", "", fmt.Errorf("%w: malformed GitHub path", ErrInvalidPluginSource)
	}
	if !githubPathPattern.MatchString(unescaped) {
		return "", "", fmt.Errorf("%w: GitHub URL must be https://github.com/{owner}/{repo}", ErrInvalidPluginSource)
	}
	unescaped = strings.TrimSuffix(unescaped, ".git")
	parts := strings.Split(unescaped, "/")
	return PluginSourceGitHub, fmt.Sprintf("https://github.com/%s/%s", parts[0], parts[1]), nil
}

func loadSingleManagedManifest(root string) (string, Manifest, []string, error) {
	paths, err := manifestPaths(root)
	if err != nil {
		return "", Manifest{}, nil, err
	}
	if len(paths) != 1 {
		return "", Manifest{}, nil, fmt.Errorf("%w: expected exactly one %s, found %d", ErrInvalidPluginSource, ManifestFileName, len(paths))
	}
	manifest, warnings, err := LoadManifest(paths[0])
	if err != nil {
		return paths[0], manifest, warnings, err
	}
	return paths[0], manifest, warnings, nil
}

func normalizeLocalDirectory(path string) (string, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "", fmt.Errorf("%w: path is required", ErrInvalidPluginSource)
	}
	if !filepath.IsAbs(trimmed) {
		return "", fmt.Errorf("%w: local plugin path must be absolute", ErrInvalidPluginSource)
	}
	cleaned, err := filepath.EvalSymlinks(trimmed)
	if err != nil {
		return "", fmt.Errorf("%w: inspect local plugin directory: %v", ErrInvalidPluginSource, err)
	}
	info, err := os.Stat(cleaned)
	if err != nil {
		return "", fmt.Errorf("%w: inspect local plugin directory: %v", ErrInvalidPluginSource, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%w: local plugin path is not a directory", ErrInvalidPluginSource)
	}
	return filepath.Clean(cleaned), nil
}

func parseBoolSetting(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func boolSettingValue(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

type installCommand struct {
	name string
	args []string
}

func (s *Service) prepareManagedInstall(ctx context.Context, dir string, manifest Manifest) error {
	commands, err := s.prepareCommandsForInstall(dir)
	if err != nil {
		return err
	}
	for _, item := range commands {
		output, err := runInstallCommand(ctx, dir, item.name, item.args...)
		if err != nil {
			message := strings.TrimSpace(string(output))
			if message == "" {
				message = err.Error()
			}
			return fmt.Errorf("%w: prepare plugin failed: %s", ErrInvalidPluginSource, message)
		}
	}
	if strings.TrimSpace(manifest.Entry.Type) == "nodePackage" {
		if _, ok, err := localNodePackageRuntimePlan(manifest.Entry, dir); err != nil {
			return fmt.Errorf("%w: prepare plugin failed: %s", ErrInvalidPluginSource, err.Error())
		} else if !ok {
			return fmt.Errorf("%w: prepare plugin failed: nodePackage entry must match a local package.json bin", ErrInvalidPluginSource)
		}
	}
	return nil
}

func (s *Service) prepareCommandsForInstall(dir string) ([]installCommand, error) {
	if strings.TrimSpace(s.prepareExecutable) != "" {
		return []installCommand{{name: s.prepareExecutable, args: cloneStringSlice(s.prepareArgs)}}, nil
	}
	scripts, ok, err := packageScripts(dir)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}

	var commands []installCommand
	switch {
	case fileExists(filepath.Join(dir, "pnpm-lock.yaml")):
		commands = append(commands, installCommand{name: "pnpm", args: []string{"install", "--frozen-lockfile"}})
		if strings.TrimSpace(scripts["build"]) != "" {
			commands = append(commands, installCommand{name: "pnpm", args: []string{"build"}})
		}
	case fileExists(filepath.Join(dir, "package-lock.json")):
		commands = append(commands, installCommand{name: "npm", args: []string{"ci"}})
		if strings.TrimSpace(scripts["build"]) != "" {
			commands = append(commands, installCommand{name: "npm", args: []string{"run", "build"}})
		}
	case fileExists(filepath.Join(dir, "yarn.lock")):
		commands = append(commands, installCommand{name: "yarn", args: []string{"install", "--frozen-lockfile"}})
		if strings.TrimSpace(scripts["build"]) != "" {
			commands = append(commands, installCommand{name: "yarn", args: []string{"build"}})
		}
	default:
		commands = append(commands, installCommand{name: "npm", args: []string{"install"}})
		if strings.TrimSpace(scripts["build"]) != "" {
			commands = append(commands, installCommand{name: "npm", args: []string{"run", "build"}})
		}
	}
	return commands, nil
}

func runInstallCommand(ctx context.Context, dir string, name string, args ...string) ([]byte, error) {
	command := exec.CommandContext(ctx, name, args...)
	command.Dir = dir
	return command.CombinedOutput()
}

func packageScripts(dir string) (map[string]string, bool, error) {
	content, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("%w: read package.json: %v", ErrInvalidPluginSource, err)
	}
	var packageJSON struct {
		Scripts map[string]string `json:"scripts"`
	}
	if err := json.Unmarshal(content, &packageJSON); err != nil {
		return nil, false, fmt.Errorf("%w: decode package.json: %v", ErrInvalidPluginSource, err)
	}
	if packageJSON.Scripts == nil {
		packageJSON.Scripts = map[string]string{}
	}
	return packageJSON.Scripts, true, nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (s *Service) git(ctx context.Context, dir string, args ...string) error {
	output, err := s.gitCombinedOutput(ctx, dir, args...)
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = err.Error()
		}
		return fmt.Errorf("git %s failed: %s", strings.Join(args, " "), message)
	}
	return nil
}

func (s *Service) gitOutput(ctx context.Context, dir string, args ...string) (string, error) {
	output, err := s.gitCombinedOutput(ctx, dir, args...)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func (s *Service) gitCombinedOutput(ctx context.Context, dir string, args ...string) ([]byte, error) {
	command := exec.CommandContext(ctx, s.gitExecutable, args...)
	if strings.TrimSpace(dir) != "" {
		command.Dir = dir
	}
	return command.CombinedOutput()
}

func managedDirectoryName(pluginID string, sourceURL string) string {
	return sanitizePathComponent(pluginID) + "-" + sourceDirectorySuffix(sourceURL)
}

func sourceDirectorySuffix(sourceURL string) string {
	sum := sha256.Sum256([]byte(sourceURL))
	return hex.EncodeToString(sum[:])[:12]
}

func sanitizePathComponent(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	builder := strings.Builder{}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '.' || r == '_' {
			builder.WriteRune(r)
			continue
		}
		builder.WriteByte('-')
	}
	result := strings.Trim(builder.String(), ".-")
	if result == "" {
		return "plugin"
	}
	return result
}
