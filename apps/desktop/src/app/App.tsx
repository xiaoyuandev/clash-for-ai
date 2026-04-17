import { useEffect, useState } from "react";
import { ProvidersPage } from "../pages/providers-page";

interface DesktopState {
  ok: boolean;
  runtime: string;
  platform: string;
}

export function App() {
  const [desktopState, setDesktopState] = useState<DesktopState | null>(null);

  useEffect(() => {
    const bridge = window.desktopBridge;
    if (!bridge) {
      return;
    }

    void bridge.ping().then((state) => {
      setDesktopState(state);
    });
  }, []);

  return <ProvidersPage desktopState={desktopState} />;
}
