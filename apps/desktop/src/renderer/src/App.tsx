import { useEffect, useState } from "react";
import { LogsPage } from "./pages/logs-page";
import { ProvidersPage } from "./pages/providers-page";
import { SettingsPage } from "./pages/settings-page";

interface DesktopState {
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
}

export default function App() {
  const [desktopState, setDesktopState] = useState<DesktopState | null>(null);
  const [view, setView] = useState<"providers" | "logs" | "settings">("providers");
  const [bootError, setBootError] = useState<string | null>(null);

  useEffect(() => {
    if (!window.desktopBridge) {
      return;
    }

    let cancelled = false;

    async function syncDesktopState() {
      try {
        const state = await window.desktopBridge.ping();
        if (cancelled) {
          return;
        }
        setDesktopState(state);
        setBootError(state.core.lastError ?? null);
      } catch (error) {
        if (cancelled) {
          return;
        }
        setBootError(error instanceof Error ? error.message : "failed to load desktop state");
      }
    }

    void syncDesktopState();
    const intervalId = window.setInterval(() => {
      void syncDesktopState();
    }, 2000);

    return () => {
      cancelled = true;
      window.clearInterval(intervalId);
    };
  }, []);

  if (!desktopState && window.desktopBridge) {
    return (
      <main className="page-shell">
        <section className="hero">
          <div>
            <p className="eyebrow">Clash for AI</p>
            <h1>Desktop Boot</h1>
            <p className="subcopy">{bootError ?? "Waiting for desktop runtime..."}</p>
          </div>
        </section>
      </main>
    );
  }

  return (
    <>
      <nav className="top-nav">
        <button
          type="button"
          className={view === "providers" ? "nav-button active-nav" : "nav-button"}
          onClick={() => {
            setView("providers");
          }}
        >
          Providers
        </button>
        <button
          type="button"
          className={view === "logs" ? "nav-button active-nav" : "nav-button"}
          onClick={() => {
            setView("logs");
          }}
        >
          Logs
        </button>
        <button
          type="button"
          className={view === "settings" ? "nav-button active-nav" : "nav-button"}
          onClick={() => {
            setView("settings");
          }}
        >
          Settings
        </button>
        <span className="runtime-chip">
          core {desktopState?.core.running ? "running" : "not running"} on{" "}
          {desktopState?.core.port ?? "-"}
        </span>
      </nav>
      {view === "providers" ? (
        <ProvidersPage desktopState={desktopState} apiBase={desktopState?.apiBase} />
      ) : view === "logs" ? (
        <LogsPage apiBase={desktopState?.apiBase} />
      ) : (
        <SettingsPage
          desktopState={desktopState}
          onCoreRestart={async () => {
            if (!window.desktopBridge) {
              return;
            }

            const response = await window.desktopBridge.restartCore();
            setDesktopState((current) =>
              current
                ? {
                    ...current,
                    apiBase: response.core.apiBase,
                    core: response.core
                  }
                : null
            );
          }}
        />
      )}
    </>
  );
}
