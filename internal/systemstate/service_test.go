package systemstate

import (
	"errors"
	"testing"

	"web-share/internal/logx"
)

type fakeLogger struct{}

func (fakeLogger) Info(string, ...logx.Field)  {}
func (fakeLogger) Warn(string, ...logx.Field)  {}
func (fakeLogger) Error(string, ...logx.Field) {}
func (fakeLogger) Audit(string, ...logx.Field) {}

type fakeContextMenuPort struct {
	inspectResults []InspectResult
	inspectErrs    []error
	installErr     error
	removeErr      error
	installCalls   int
	removeCalls    int
	inspectCalls   int
}

type fakeAutostartPort struct {
	inspectResults []InspectResult
	inspectErrs    []error
	enableErr      error
	disableErr     error
	enableCalls    int
	disableCalls   int
	inspectCalls   int
}

type fakeTrayPort struct {
	inspectResults []InspectResult
	inspectErrs    []error
	startErr       error
	stopErr        error
	startCalls     int
	stopCalls      int
	inspectCalls   int
}

type fakeProgramPort struct {
	inspectResults []InspectResult
	inspectErrs    []error
	stopErr        error
	stopCalls      int
	inspectCalls   int
}

type fakeManagerPort struct {
	inspectResults []InspectResult
	inspectErrs    []error
	startErr       error
	stopErr        error
	startCalls     int
	stopCalls      int
	inspectCalls   int
}

func (f *fakeContextMenuPort) Inspect(string) (InspectResult, error) {
	idx := f.inspectCalls
	f.inspectCalls++
	if idx < len(f.inspectErrs) && f.inspectErrs[idx] != nil {
		return InspectResult{}, f.inspectErrs[idx]
	}
	if idx < len(f.inspectResults) {
		return f.inspectResults[idx], nil
	}
	if len(f.inspectResults) == 0 {
		return InspectResult{}, nil
	}
	return f.inspectResults[len(f.inspectResults)-1], nil
}

func (f *fakeContextMenuPort) Install(string, string) error {
	f.installCalls++
	return f.installErr
}

func (f *fakeContextMenuPort) Remove() error {
	f.removeCalls++
	return f.removeErr
}

func (f *fakeAutostartPort) Inspect(string, string) (InspectResult, error) {
	idx := f.inspectCalls
	f.inspectCalls++
	if idx < len(f.inspectErrs) && f.inspectErrs[idx] != nil {
		return InspectResult{}, f.inspectErrs[idx]
	}
	if idx < len(f.inspectResults) {
		return f.inspectResults[idx], nil
	}
	if len(f.inspectResults) == 0 {
		return InspectResult{}, nil
	}
	return f.inspectResults[len(f.inspectResults)-1], nil
}

func (f *fakeAutostartPort) Enable(string, string) error {
	f.enableCalls++
	return f.enableErr
}

func (f *fakeAutostartPort) Disable(string) error {
	f.disableCalls++
	return f.disableErr
}

func (f *fakeTrayPort) Inspect() (InspectResult, error) {
	idx := f.inspectCalls
	f.inspectCalls++
	if idx < len(f.inspectErrs) && f.inspectErrs[idx] != nil {
		return InspectResult{}, f.inspectErrs[idx]
	}
	if idx < len(f.inspectResults) {
		return f.inspectResults[idx], nil
	}
	if len(f.inspectResults) == 0 {
		return InspectResult{}, nil
	}
	return f.inspectResults[len(f.inspectResults)-1], nil
}

func (f *fakeTrayPort) Start(string) error {
	f.startCalls++
	return f.startErr
}

func (f *fakeTrayPort) Stop() error {
	f.stopCalls++
	return f.stopErr
}

func (f *fakeProgramPort) Inspect() (InspectResult, error) {
	idx := f.inspectCalls
	f.inspectCalls++
	if idx < len(f.inspectErrs) && f.inspectErrs[idx] != nil {
		return InspectResult{}, f.inspectErrs[idx]
	}
	if idx < len(f.inspectResults) {
		return f.inspectResults[idx], nil
	}
	if len(f.inspectResults) == 0 {
		return InspectResult{}, nil
	}
	return f.inspectResults[len(f.inspectResults)-1], nil
}

func (f *fakeProgramPort) Stop() error {
	f.stopCalls++
	return f.stopErr
}

func (f *fakeManagerPort) Inspect() (InspectResult, error) {
	idx := f.inspectCalls
	f.inspectCalls++
	if idx < len(f.inspectErrs) && f.inspectErrs[idx] != nil {
		return InspectResult{}, f.inspectErrs[idx]
	}
	if idx < len(f.inspectResults) {
		return f.inspectResults[idx], nil
	}
	if len(f.inspectResults) == 0 {
		return InspectResult{}, nil
	}
	return f.inspectResults[len(f.inspectResults)-1], nil
}

func (f *fakeManagerPort) Start(string) error {
	f.startCalls++
	return f.startErr
}

func (f *fakeManagerPort) Stop() error {
	f.stopCalls++
	return f.stopErr
}

func TestEnsureContextMenuInstalledAlreadySatisfied(t *testing.T) {
	port := &fakeContextMenuPort{
		inspectResults: []InspectResult{{Installed: true}},
	}
	service := NewService(fakeLogger{})
	service.ContextMenu = port

	result := service.EnsureContextMenuInstalled("C:\\app.exe", "en-US")
	if !result.OK {
		t.Fatalf("expected OK result, got %+v", result)
	}
	if result.Changed {
		t.Fatalf("expected unchanged result, got %+v", result)
	}
	if port.installCalls != 0 || port.removeCalls != 0 {
		t.Fatalf("expected no install/remove calls, got install=%d remove=%d", port.installCalls, port.removeCalls)
	}
}

func TestEnsureContextMenuInstalledCleansDirtyState(t *testing.T) {
	port := &fakeContextMenuPort{
		inspectResults: []InspectResult{
			{Installed: true, Dirty: true, Warnings: []string{"legacy command detected"}},
			{Installed: true},
		},
	}
	service := NewService(fakeLogger{})
	service.ContextMenu = port

	result := service.EnsureContextMenuInstalled("C:\\app.exe", "en-US")
	if !result.OK || !result.Changed {
		t.Fatalf("expected changed OK result, got %+v", result)
	}
	if port.removeCalls != 1 || port.installCalls != 1 {
		t.Fatalf("expected one cleanup and one install, got install=%d remove=%d", port.installCalls, port.removeCalls)
	}
	if len(result.Warnings) == 0 {
		t.Fatalf("expected warning passthrough, got %+v", result)
	}
}

func TestEnsureContextMenuRemovedWarnsWhenAlreadyMissing(t *testing.T) {
	port := &fakeContextMenuPort{
		inspectResults: []InspectResult{{Installed: false}},
	}
	service := NewService(fakeLogger{})
	service.ContextMenu = port

	result := service.EnsureContextMenuRemoved("C:\\app.exe")
	if !result.OK || result.Changed {
		t.Fatalf("expected unchanged OK result, got %+v", result)
	}
	if len(result.Warnings) == 0 {
		t.Fatalf("expected warning when already missing, got %+v", result)
	}
}

func TestEnsureContextMenuInstalledFailsWhenCleanupFails(t *testing.T) {
	port := &fakeContextMenuPort{
		inspectResults: []InspectResult{{Installed: true, Dirty: true}},
		removeErr:      errors.New("cleanup failed"),
	}
	service := NewService(fakeLogger{})
	service.ContextMenu = port

	result := service.EnsureContextMenuInstalled("C:\\app.exe", "en-US")
	if result.OK {
		t.Fatalf("expected failure result, got %+v", result)
	}
}

func TestEnsureAutostartEnabledAlreadySatisfied(t *testing.T) {
	port := &fakeAutostartPort{
		inspectResults: []InspectResult{{Installed: true}},
	}
	service := NewService(fakeLogger{})
	service.Autostart = port

	result := service.EnsureAutostartEnabled("WebShare.AutoStart", `"C:\app.exe" start`)
	if !result.OK || result.Changed {
		t.Fatalf("expected unchanged OK result, got %+v", result)
	}
	if port.enableCalls != 0 {
		t.Fatalf("expected no enable calls, got %d", port.enableCalls)
	}
}

func TestEnsureAutostartDisabledWarnsWhenMissing(t *testing.T) {
	port := &fakeAutostartPort{
		inspectResults: []InspectResult{{Installed: false}},
	}
	service := NewService(fakeLogger{})
	service.Autostart = port

	result := service.EnsureAutostartDisabled("WebShare.AutoStart", `"C:\app.exe" start`)
	if !result.OK || result.Changed {
		t.Fatalf("expected unchanged OK result, got %+v", result)
	}
	if len(result.Warnings) == 0 {
		t.Fatalf("expected warning result, got %+v", result)
	}
}

func TestEnsureAutostartEnabledFailsVerification(t *testing.T) {
	port := &fakeAutostartPort{
		inspectResults: []InspectResult{{Installed: false}, {Installed: true, Dirty: true}},
	}
	service := NewService(fakeLogger{})
	service.Autostart = port

	result := service.EnsureAutostartEnabled("WebShare.AutoStart", `"C:\app.exe" start`)
	if result.OK {
		t.Fatalf("expected verification failure, got %+v", result)
	}
}

func TestEnsureTrayRunningStartsWhenMissing(t *testing.T) {
	port := &fakeTrayPort{
		inspectResults: []InspectResult{{Installed: false}, {Installed: true}},
	}
	service := NewService(fakeLogger{})
	service.Tray = port

	result := service.EnsureTrayRunning("C:\\app.exe")
	if !result.OK || !result.Changed {
		t.Fatalf("expected changed OK result, got %+v", result)
	}
	if port.startCalls != 1 {
		t.Fatalf("expected one start call, got %d", port.startCalls)
	}
}

func TestEnsureTrayStoppedWarnsWhenAlreadyStopped(t *testing.T) {
	port := &fakeTrayPort{
		inspectResults: []InspectResult{{Installed: false}},
	}
	service := NewService(fakeLogger{})
	service.Tray = port

	result := service.EnsureTrayStopped()
	if !result.OK || result.Changed {
		t.Fatalf("expected unchanged OK result, got %+v", result)
	}
	if len(result.Warnings) == 0 {
		t.Fatalf("expected warnings, got %+v", result)
	}
}

func TestEnsureTrayRunningFailsVerification(t *testing.T) {
	port := &fakeTrayPort{
		inspectResults: []InspectResult{{Installed: false}, {Installed: false}},
	}
	service := NewService(fakeLogger{})
	service.Tray = port

	result := service.EnsureTrayRunning("C:\\app.exe")
	if result.OK {
		t.Fatalf("expected verification failure, got %+v", result)
	}
}

func TestEnsureProgramStoppedRequestsStop(t *testing.T) {
	port := &fakeProgramPort{
		inspectResults: []InspectResult{{Installed: true}},
	}
	service := NewService(fakeLogger{})
	service.Program = port

	result := service.EnsureProgramStopped()
	if !result.OK || !result.Changed {
		t.Fatalf("expected changed OK result, got %+v", result)
	}
	if port.stopCalls != 1 {
		t.Fatalf("expected one stop call, got %d", port.stopCalls)
	}
}

func TestSnapshotAggregatesStatuses(t *testing.T) {
	service := NewService(fakeLogger{})
	service.ContextMenu = &fakeContextMenuPort{inspectResults: []InspectResult{{Installed: true, Dirty: true, Warnings: []string{"ctx dirty"}}}}
	service.Autostart = &fakeAutostartPort{inspectResults: []InspectResult{{Installed: false, Warnings: []string{"auto missing"}}}}
	service.Tray = &fakeTrayPort{inspectResults: []InspectResult{{Installed: true}}}

	snapshot, err := service.Snapshot("C:\\app.exe", "WebShare.AutoStart", `"C:\app.exe" start`)
	if err != nil {
		t.Fatalf("unexpected snapshot error: %v", err)
	}
	if !snapshot.ContextMenuInstalled || !snapshot.ContextMenuDirty {
		t.Fatalf("unexpected context snapshot: %+v", snapshot)
	}
	if snapshot.AutostartEnabled {
		t.Fatalf("expected autostart disabled snapshot: %+v", snapshot)
	}
	if !snapshot.TrayRunning {
		t.Fatalf("expected tray running snapshot: %+v", snapshot)
	}
	if len(snapshot.Warnings) != 2 {
		t.Fatalf("expected aggregated warnings, got %+v", snapshot)
	}
}

func TestEnsureManagerRunningStartsWhenMissing(t *testing.T) {
	port := &fakeManagerPort{
		inspectResults: []InspectResult{{Installed: false}, {Installed: true}},
	}
	service := NewService(fakeLogger{})
	service.Manager = port

	result := service.EnsureManagerRunning("C:\\app.exe")
	if !result.OK || !result.Changed {
		t.Fatalf("expected changed OK result, got %+v", result)
	}
	if port.startCalls != 1 {
		t.Fatalf("expected one start call, got %d", port.startCalls)
	}
}

func TestEnsureManagerStoppedWarnsWhenMissing(t *testing.T) {
	port := &fakeManagerPort{
		inspectResults: []InspectResult{{Installed: false}},
	}
	service := NewService(fakeLogger{})
	service.Manager = port

	result := service.EnsureManagerStopped()
	if !result.OK || result.Changed {
		t.Fatalf("expected unchanged OK result, got %+v", result)
	}
	if len(result.Warnings) == 0 {
		t.Fatalf("expected warnings, got %+v", result)
	}
}

func TestEnsureManagerRunningFailsVerification(t *testing.T) {
	port := &fakeManagerPort{
		inspectResults: []InspectResult{{Installed: false}, {Installed: false}},
	}
	service := NewService(fakeLogger{})
	service.Manager = port

	result := service.EnsureManagerRunning("C:\\app.exe")
	if result.OK {
		t.Fatalf("expected verification failure, got %+v", result)
	}
}
