import { useEffect, useState } from "react";
import { LogsPage } from "./pages/logs-page";
import { ProvidersPage } from "./pages/providers-page";

interface DesktopState {
  ok: boolean;
  runtime: string;
  platform: string;
}

export default function App() {
  const [desktopState, setDesktopState] = useState<DesktopState | null>(null);
  const [view, setView] = useState<"providers" | "logs">("providers");

  useEffect(() => {
    if (!window.desktopBridge) {
      return;
    }

    void window.desktopBridge.ping().then((state) => {
      setDesktopState(state);
    });
  }, []);
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
      </nav>
      {view === "providers" ? (
        <ProvidersPage desktopState={desktopState} />
      ) : (
        <LogsPage />
      )}
    </>
  );
}
