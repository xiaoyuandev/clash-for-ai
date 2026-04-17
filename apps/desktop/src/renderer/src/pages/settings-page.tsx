import { useState } from "react";

interface SettingsPageProps {
  desktopState: {
    ok: boolean;
    runtime: string;
    platform: string;
    apiBase: string;
    core: {
      managed: boolean;
      running: boolean;
      apiBase: string;
      port: number;
      lastError?: string;
      command?: string;
    };
  } | null;
  onCoreRestart: () => Promise<void>;
}

export function SettingsPage({
  desktopState,
  onCoreRestart
}: SettingsPageProps) {
  const [busy, setBusy] = useState(false);
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
    </main>
  );
}
