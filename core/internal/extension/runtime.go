package extension

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const runtimeRequestTimeout = 60 * time.Second

type RuntimeHost struct {
	mu        sync.Mutex
	processes map[string]*runtimeProcess
}

type runtimeProcess struct {
	mu       sync.Mutex
	plugin   Plugin
	plan     PluginRuntimePlan
	command  *exec.Cmd
	stdin    io.WriteCloser
	pending  map[int64]chan rpcResponse
	nextID   int64
	state    PluginRuntimeState
	lastErr  string
	updated  string
	stderr   cappedBuffer
	done     chan struct{}
	stopOnce sync.Once
}

type rpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int64  `json:"id,omitempty"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type cappedBuffer struct {
	mu    sync.Mutex
	limit int
	text  string
}

func NewRuntimeHost() *RuntimeHost {
	return &RuntimeHost{processes: map[string]*runtimeProcess{}}
}

func RuntimePlanForPlugin(plugin Plugin) (PluginRuntimePlan, error) {
	entry := plugin.Manifest.Entry
	entryType := strings.TrimSpace(entry.Type)
	switch entryType {
	case "", "none":
		return PluginRuntimePlan{EntryType: "none"}, nil
	case "process":
		if err := validateProcessEntryCommand(entry.Command); err != nil {
			return PluginRuntimePlan{}, err
		}
		manifestDir := filepath.Dir(plugin.ManifestPath)
		command := filepath.Clean(filepath.Join(manifestDir, strings.TrimSpace(entry.Command)))
		return PluginRuntimePlan{
			EntryType: "process",
			Command:   command,
			Args:      cloneStringSlice(entry.Args),
			Cwd:       manifestDir,
		}, nil
	case "nodePackage":
		if err := validateNodePackageEntry(entry); err != nil {
			return PluginRuntimePlan{}, err
		}
		if plugin.Install != nil && plugin.Install.SourceType == string(PluginSourceLocalDirectory) {
			plan, ok, err := localNodePackageRuntimePlan(entry, filepath.Dir(plugin.ManifestPath))
			if err != nil {
				return PluginRuntimePlan{}, err
			}
			if ok {
				return plan, nil
			}
		}
		packageSpec := fmt.Sprintf("%s@%s", strings.TrimSpace(entry.Package), strings.TrimSpace(entry.Version))
		args := []string{"--yes", "--package", packageSpec, strings.TrimSpace(entry.Bin)}
		args = append(args, entry.Args...)
		return PluginRuntimePlan{
			EntryType: "nodePackage",
			Command:   "npx",
			Args:      args,
			Cwd:       filepath.Dir(plugin.ManifestPath),
		}, nil
	default:
		return PluginRuntimePlan{}, fmt.Errorf("unsupported runtime entry type: %s", entryType)
	}
}

func localNodePackageRuntimePlan(entry ManifestEntry, manifestDir string) (PluginRuntimePlan, bool, error) {
	packagePath := filepath.Join(manifestDir, "package.json")
	content, err := os.ReadFile(packagePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return PluginRuntimePlan{}, false, nil
		}
		return PluginRuntimePlan{}, false, fmt.Errorf("read local package.json: %w", err)
	}

	var packageJSON struct {
		Name string          `json:"name"`
		Bin  json.RawMessage `json:"bin"`
	}
	if err := json.Unmarshal(content, &packageJSON); err != nil {
		return PluginRuntimePlan{}, false, fmt.Errorf("decode local package.json: %w", err)
	}
	if strings.TrimSpace(packageJSON.Name) != strings.TrimSpace(entry.Package) {
		return PluginRuntimePlan{}, false, nil
	}

	binPath, err := localPackageBinPath(packageJSON.Bin, strings.TrimSpace(entry.Bin))
	if err != nil {
		return PluginRuntimePlan{}, true, err
	}
	binPath = filepath.Clean(binPath)
	if filepath.IsAbs(binPath) || binPath == "." || binPath == ".." || strings.HasPrefix(binPath, ".."+string(filepath.Separator)) {
		return PluginRuntimePlan{}, true, fmt.Errorf("local package bin %q must stay inside the plugin directory", entry.Bin)
	}

	command := filepath.Join(manifestDir, binPath)
	if _, err := os.Stat(command); err != nil {
		return PluginRuntimePlan{}, true, fmt.Errorf("local package bin %q is not built: %w", entry.Bin, err)
	}
	args := []string{command}
	args = append(args, entry.Args...)
	return PluginRuntimePlan{
		EntryType: "nodePackage",
		Command:   "node",
		Args:      args,
		Cwd:       manifestDir,
	}, true, nil
}

func localPackageBinPath(raw json.RawMessage, binName string) (string, error) {
	var binMap map[string]string
	if err := json.Unmarshal(raw, &binMap); err == nil && len(binMap) > 0 {
		if value := strings.TrimSpace(binMap[binName]); value != "" {
			return value, nil
		}
		return "", fmt.Errorf("local package.json does not declare bin %q", binName)
	}

	var singleBin string
	if err := json.Unmarshal(raw, &singleBin); err == nil && strings.TrimSpace(singleBin) != "" {
		return strings.TrimSpace(singleBin), nil
	}
	return "", fmt.Errorf("local package.json does not declare bin %q", binName)
}

func (h *RuntimeHost) Start(ctx context.Context, plugin Plugin, initialSettings map[string]any) error {
	if h == nil {
		return nil
	}
	plan, err := RuntimePlanForPlugin(plugin)
	if err != nil {
		h.setDegraded(plugin, plan, err)
		return err
	}
	if plan.EntryType == "none" || !plugin.Enabled || plugin.Status != PluginStatusEnabled {
		return nil
	}

	h.mu.Lock()
	if existing, ok := h.processes[plugin.ID]; ok && existing.isRunning() {
		h.mu.Unlock()
		return nil
	}
	process := newRuntimeProcess(plugin, plan)
	h.processes[plugin.ID] = process
	h.mu.Unlock()

	if err := process.start(ctx); err != nil {
		process.setState(PluginRuntimeStateDegraded, err.Error())
		return err
	}
	initCtx, cancel := context.WithTimeout(ctx, runtimeRequestTimeout)
	defer cancel()
	initializeParams := map[string]any{
		"plugin": map[string]any{
			"id":      plugin.ID,
			"name":    plugin.Name,
			"version": plugin.Version,
		},
		"manifest": plugin.Manifest,
	}
	if initialSettings != nil {
		initializeParams["settings"] = initialSettings
	}
	if _, err := process.call(initCtx, "initialize", initializeParams); err != nil {
		process.setState(PluginRuntimeStateDegraded, err.Error())
		return err
	}
	process.setState(PluginRuntimeStateRunning, "")
	return nil
}

func (h *RuntimeHost) Stop(ctx context.Context, pluginID string) error {
	if h == nil {
		return nil
	}
	h.mu.Lock()
	process := h.processes[pluginID]
	delete(h.processes, pluginID)
	h.mu.Unlock()
	if process == nil {
		return nil
	}
	return process.stop(ctx)
}

func (h *RuntimeHost) Call(ctx context.Context, plugin Plugin, method string, params any, initialSettings map[string]any) (json.RawMessage, error) {
	if h == nil {
		return nil, errors.New("runtime host unavailable")
	}
	if err := h.Start(ctx, plugin, initialSettings); err != nil {
		return nil, err
	}
	h.mu.Lock()
	process := h.processes[plugin.ID]
	h.mu.Unlock()
	if process == nil {
		return nil, errors.New("runtime process is not running")
	}
	callCtx, cancel := context.WithTimeout(ctx, runtimeRequestTimeout)
	defer cancel()
	return process.call(callCtx, method, params)
}

func (h *RuntimeHost) Notify(ctx context.Context, plugin Plugin, method string, params any, initialSettings map[string]any) error {
	if h == nil {
		return nil
	}
	if err := h.Start(ctx, plugin, initialSettings); err != nil {
		return err
	}
	h.mu.Lock()
	process := h.processes[plugin.ID]
	h.mu.Unlock()
	if process == nil {
		return nil
	}
	return process.notify(ctx, method, params)
}

func (h *RuntimeHost) Status(plugin Plugin) PluginRuntime {
	plan, err := RuntimePlanForPlugin(plugin)
	if err != nil {
		return PluginRuntime{
			State:     PluginRuntimeStateDegraded,
			EntryType: strings.TrimSpace(plugin.Manifest.Entry.Type),
			LastError: err.Error(),
			Args:      []string{},
			UpdatedAt: time.Now().UTC().Format(time.RFC3339),
		}
	}
	if plan.EntryType == "none" {
		return PluginRuntime{
			State:     PluginRuntimeStateNone,
			EntryType: "none",
			Args:      []string{},
		}
	}
	if h == nil {
		return runtimeStatusFromPlan(plan, PluginRuntimeStateStopped, "")
	}
	h.mu.Lock()
	process := h.processes[plugin.ID]
	h.mu.Unlock()
	if process == nil {
		return runtimeStatusFromPlan(plan, PluginRuntimeStateStopped, "")
	}
	return process.status()
}

func (h *RuntimeHost) setDegraded(plugin Plugin, plan PluginRuntimePlan, err error) {
	if h == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if plan.EntryType == "" {
		plan.EntryType = strings.TrimSpace(plugin.Manifest.Entry.Type)
	}
	process := newRuntimeProcess(plugin, plan)
	process.setState(PluginRuntimeStateDegraded, err.Error())
	h.processes[plugin.ID] = process
}

func newRuntimeProcess(plugin Plugin, plan PluginRuntimePlan) *runtimeProcess {
	return &runtimeProcess{
		plugin:  plugin,
		plan:    plan,
		pending: map[int64]chan rpcResponse{},
		state:   PluginRuntimeStateStopped,
		updated: time.Now().UTC().Format(time.RFC3339),
		stderr:  cappedBuffer{limit: 4096},
		done:    make(chan struct{}),
	}
}

func (p *runtimeProcess) start(_ context.Context) error {
	p.setState(PluginRuntimeStateStarting, "")
	command := exec.Command(p.plan.Command, p.plan.Args...)
	command.Dir = p.plan.Cwd

	stdin, err := command.StdinPipe()
	if err != nil {
		return fmt.Errorf("open runtime stdin: %w", err)
	}
	stdout, err := command.StdoutPipe()
	if err != nil {
		return fmt.Errorf("open runtime stdout: %w", err)
	}
	stderr, err := command.StderrPipe()
	if err != nil {
		return fmt.Errorf("open runtime stderr: %w", err)
	}
	if err := command.Start(); err != nil {
		return fmt.Errorf("start runtime: %w", err)
	}

	p.mu.Lock()
	p.command = command
	p.stdin = stdin
	p.mu.Unlock()

	go p.readStdout(stdout)
	go p.readStderr(stderr)
	go p.wait()
	return nil
}

func (p *runtimeProcess) call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	id, responseCh, err := p.enqueueRequest(method, params)
	if err != nil {
		return nil, err
	}
	select {
	case response := <-responseCh:
		if response.Error != nil {
			return nil, errors.New(response.Error.Message)
		}
		return response.Result, nil
	case <-ctx.Done():
		p.removePending(id)
		return nil, ctx.Err()
	}
}

func (p *runtimeProcess) notify(ctx context.Context, method string, params any) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.stdin == nil {
		return errors.New("runtime stdin is closed")
	}
	content, err := json.Marshal(rpcRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	})
	if err != nil {
		return err
	}
	_, err = p.stdin.Write(append(content, '\n'))
	return err
}

func (p *runtimeProcess) enqueueRequest(method string, params any) (int64, chan rpcResponse, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.stdin == nil {
		return 0, nil, errors.New("runtime stdin is closed")
	}
	p.nextID++
	id := p.nextID
	responseCh := make(chan rpcResponse, 1)
	p.pending[id] = responseCh
	content, err := json.Marshal(rpcRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	})
	if err != nil {
		delete(p.pending, id)
		return 0, nil, err
	}
	if _, err := p.stdin.Write(append(content, '\n')); err != nil {
		delete(p.pending, id)
		return 0, nil, err
	}
	return id, responseCh, nil
}

func (p *runtimeProcess) removePending(id int64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.pending, id)
}

func (p *runtimeProcess) readStdout(stdout io.Reader) {
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var response rpcResponse
		if err := json.Unmarshal([]byte(line), &response); err != nil || response.ID == 0 {
			continue
		}
		p.mu.Lock()
		responseCh := p.pending[response.ID]
		delete(p.pending, response.ID)
		p.mu.Unlock()
		if responseCh != nil {
			responseCh <- response
		}
	}
	if err := scanner.Err(); err != nil {
		p.setState(PluginRuntimeStateDegraded, err.Error())
	}
}

func (p *runtimeProcess) readStderr(stderr io.Reader) {
	scanner := bufio.NewScanner(stderr)
	scanner.Buffer(make([]byte, 0, 16*1024), 256*1024)
	for scanner.Scan() {
		p.stderr.append(scanner.Text() + "\n")
	}
}

func (p *runtimeProcess) wait() {
	err := p.command.Wait()
	close(p.done)
	if err != nil {
		message := strings.TrimSpace(err.Error() + ": " + p.stderr.String())
		p.setState(PluginRuntimeStateDegraded, message)
		return
	}
	p.setState(PluginRuntimeStateStopped, "")
}

func (p *runtimeProcess) stop(ctx context.Context) error {
	var stopErr error
	p.stopOnce.Do(func() {
		shutdownCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		_, _ = p.call(shutdownCtx, "shutdown", map[string]any{})

		p.mu.Lock()
		if p.stdin != nil {
			_ = p.stdin.Close()
		}
		command := p.command
		p.mu.Unlock()

		if command != nil && command.Process != nil {
			select {
			case <-p.done:
			case <-ctx.Done():
				stopErr = ctx.Err()
			case <-time.After(2 * time.Second):
				stopErr = command.Process.Kill()
			}
		}
		p.setState(PluginRuntimeStateStopped, "")
	})
	return stopErr
}

func (p *runtimeProcess) isRunning() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.command != nil && p.state == PluginRuntimeStateRunning
}

func (p *runtimeProcess) status() PluginRuntime {
	p.mu.Lock()
	defer p.mu.Unlock()
	return PluginRuntime{
		State:     p.state,
		EntryType: p.plan.EntryType,
		Command:   p.plan.Command,
		Args:      cloneStringSlice(p.plan.Args),
		Cwd:       p.plan.Cwd,
		LastError: p.lastErr,
		UpdatedAt: p.updated,
	}
}

func (p *runtimeProcess) setState(state PluginRuntimeState, lastErr string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.state = state
	p.lastErr = strings.TrimSpace(lastErr)
	p.updated = time.Now().UTC().Format(time.RFC3339)
}

func runtimeStatusFromPlan(plan PluginRuntimePlan, state PluginRuntimeState, lastErr string) PluginRuntime {
	return PluginRuntime{
		State:     state,
		EntryType: plan.EntryType,
		Command:   plan.Command,
		Args:      cloneStringSlice(plan.Args),
		Cwd:       plan.Cwd,
		LastError: strings.TrimSpace(lastErr),
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}
}

func (b *cappedBuffer) append(value string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.text += value
	if b.limit > 0 && len(b.text) > b.limit {
		b.text = b.text[len(b.text)-b.limit:]
	}
}

func (b *cappedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.text
}

func (s *Service) decoratePlugin(plugin Plugin) Plugin {
	if s.runtimeHost == nil {
		plugin.Runtime = runtimeStatusFromPlan(PluginRuntimePlan{EntryType: strings.TrimSpace(plugin.Manifest.Entry.Type)}, PluginRuntimeStateStopped, "")
		return plugin
	}
	plugin.Runtime = s.runtimeHost.Status(plugin)
	return plugin
}

func (s *Service) decoratePlugins(items []Plugin) []Plugin {
	decorated := make([]Plugin, 0, len(items))
	for _, item := range items {
		decorated = append(decorated, s.decoratePlugin(item))
	}
	return decorated
}

func (s *Service) startRuntime(ctx context.Context, plugin Plugin) {
	if s.runtimeHost == nil {
		return
	}
	if err := s.guardDevelopmentRuntime(ctx, plugin); err != nil {
		return
	}
	if err := s.runtimeHost.Start(ctx, plugin, s.effectiveSettingsForRuntime(ctx, plugin.ID)); err != nil {
		_, _ = s.repository.RecordAudit(ctx, AuditLogEntry{
			PluginID:      plugin.ID,
			PluginVersion: plugin.Version,
			Capability:    "runtime.lifecycle",
			Action:        "runtime.start",
			ResourceType:  "runtime",
			ResourceID:    plugin.ID,
			Status:        "failed",
			ErrorMessage:  err.Error(),
		})
	}
}

func (s *Service) effectiveSettingsForRuntime(ctx context.Context, pluginID string) map[string]any {
	settings, err := s.GetSettings(ctx, pluginID)
	if err != nil {
		return nil
	}
	return settings.EffectiveValues
}
