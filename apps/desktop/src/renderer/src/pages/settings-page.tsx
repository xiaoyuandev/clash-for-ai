import { useState } from "react";

interface SettingsPageProps {
  desktopState: {
    ok: boolean;
    runtime: string;
    platform: string;
    apiBase: string;
    updates: {
      currentVersion: string;
      status:
        | "idle"
        | "checking"
        | "available"
        | "not-available"
        | "downloading"
        | "downloaded"
        | "error"
        | "unsupported";
      availableVersion?: string;
      downloadedVersion?: string;
      progressPercent?: number;
      message?: string;
    };
    core: {
      managed: boolean;
      running: boolean;
      apiBase: string;
      port: number;
      logRetentionDays: number;
      logMaxRecords: number;
      lastError?: string;
      command?: string;
    };
  } | null;
  onCheckUpdates: () => Promise<void>;
  onDownloadUpdate: () => Promise<void>;
  onQuitAndInstallUpdate: () => Promise<void>;
  onCoreRestart: () => Promise<void>;
}

export function SettingsPage({
  desktopState,
  onCheckUpdates,
  onDownloadUpdate,
  onQuitAndInstallUpdate,
  onCoreRestart
}: SettingsPageProps) {
  const [busy, setBusy] = useState(false);
  const [updateBusy, setUpdateBusy] = useState(false);
  const [feedback, setFeedback] = useState<string | null>(null);

  async function handleRestart() {
    setBusy(true);
    setFeedback(null);

    try {
      await onCoreRestart();
      setFeedback("Core restarted.");
    } catch (error) {
      setFeedback(error instanceof Error ? error.message : "Failed to restart core");
    } finally {
      setBusy(false);
    }
  }

  async function handleCheckUpdates() {
    setUpdateBusy(true);
    setFeedback(null);

    try {
      await onCheckUpdates();
    } catch (error) {
      setFeedback(error instanceof Error ? error.message : "Failed to check updates");
    } finally {
      setUpdateBusy(false);
    }
  }

  async function handleDownloadUpdate() {
    setUpdateBusy(true);
    setFeedback(null);

    try {
      await onDownloadUpdate();
    } catch (error) {
      setFeedback(error instanceof Error ? error.message : "Failed to download update");
    } finally {
      setUpdateBusy(false);
    }
  }

  async function handleQuitAndInstallUpdate() {
    setUpdateBusy(true);
    setFeedback(null);

    try {
      await onQuitAndInstallUpdate();
    } catch (error) {
      setFeedback(
        error instanceof Error ? error.message : "Failed to install downloaded update"
      );
      setUpdateBusy(false);
    }
  }

  return (
    <main className="page-shell">
      <section className="hero">
        <div>
          <p className="eyebrow">Clash for AI</p>
          <h1>Settings</h1>
          <p className="subcopy">
            Runtime details for the local desktop shell and the managed Go core.
          </p>
        </div>
      </section>

      {feedback ? <p className="panel info-panel">{feedback}</p> : null}

      <section className="panel">
        <div className="section-head">
          <h2>Runtime</h2>
          <span>{desktopState?.runtime ?? "unknown"}</span>
        </div>

        <div className="settings-grid">
          <div className="settings-card">
            <p className="settings-label">Platform</p>
            <p className="mono">{desktopState?.platform ?? "-"}</p>
          </div>
          <div className="settings-card">
            <p className="settings-label">API Base</p>
            <p className="mono">{desktopState?.apiBase ?? "-"}</p>
          </div>
          <div className="settings-card">
            <p className="settings-label">Core Port</p>
            <p className="mono">{desktopState?.core.port ?? "-"}</p>
          </div>
          <div className="settings-card">
            <p className="settings-label">Core Managed</p>
            <p className="mono">{desktopState?.core.managed ? "yes" : "no"}</p>
          </div>
          <div className="settings-card">
            <p className="settings-label">Core Running</p>
            <p className="mono">{desktopState?.core.running ? "yes" : "no"}</p>
          </div>
          <div className="settings-card">
            <p className="settings-label">Launch Command</p>
            <p className="mono">{desktopState?.core.command ?? "-"}</p>
          </div>
          <div className="settings-card">
            <p className="settings-label">Log Retention</p>
            <p className="mono">
              {desktopState
                ? `${desktopState.core.logRetentionDays} days / ${desktopState.core.logMaxRecords.toLocaleString()} records`
                : "-"}
            </p>
            <p className="meta">
              Request logs are pruned automatically to keep SQLite size under control.
            </p>
          </div>
        </div>

        <div className="settings-actions">
          <button type="button" onClick={() => void handleRestart()} disabled={busy}>
            {busy ? "Restarting..." : "Restart Core"}
          </button>
        </div>

        {desktopState?.core.lastError ? (
          <p className="log-error">
            <span className="mono">{desktopState.core.lastError}</span>
          </p>
        ) : null}
      </section>

      <section className="panel">
        <div className="section-head">
          <h2>Updates</h2>
          <span>{desktopState?.updates.currentVersion ?? "-"}</span>
        </div>

        <div className="settings-grid">
          <div className="settings-card">
            <p className="settings-label">Current Version</p>
            <p className="mono">{desktopState?.updates.currentVersion ?? "-"}</p>
          </div>
          <div className="settings-card">
            <p className="settings-label">Update Status</p>
            <p className="mono">{desktopState?.updates.status ?? "-"}</p>
            {desktopState?.updates.message ? (
              <p className="meta">{desktopState.updates.message}</p>
            ) : null}
          </div>
          <div className="settings-card">
            <p className="settings-label">Available Version</p>
            <p className="mono">
              {desktopState?.updates.availableVersion ??
                desktopState?.updates.downloadedVersion ??
                "-"}
            </p>
          </div>
        </div>

        <div className="settings-actions">
          <button
            type="button"
            className="secondary-button"
            onClick={() => void handleCheckUpdates()}
            disabled={updateBusy || desktopState?.updates.status === "unsupported"}
          >
            {updateBusy && desktopState?.updates.status === "checking"
              ? "Checking..."
              : "Check for Updates"}
          </button>
          <button
            type="button"
            className="secondary-button"
            onClick={() => void handleDownloadUpdate()}
            disabled={
              updateBusy ||
              (desktopState?.updates.status !== "available" &&
                desktopState?.updates.status !== "downloading")
            }
          >
            {desktopState?.updates.status === "downloading"
              ? `Downloading ${Math.round(desktopState.updates.progressPercent ?? 0)}%`
              : "Download Update"}
          </button>
          <button
            type="button"
            onClick={() => void handleQuitAndInstallUpdate()}
            disabled={updateBusy || desktopState?.updates.status !== "downloaded"}
          >
            Restart to Install
          </button>
        </div>
      </section>
    </main>
  );
}
