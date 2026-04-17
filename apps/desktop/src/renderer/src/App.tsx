import { useEffect, useState } from "react";
import { ProvidersPage } from "./pages/providers-page";

interface DesktopState {
  ok: boolean;
  runtime: string;
  platform: string;
}

export default function App() {
  const [desktopState, setDesktopState] = useState<DesktopState | null>(null);

  useEffect(() => {
    if (!window.desktopBridge) {
      return;
    }

    void window.desktopBridge.ping().then((state) => {
      setDesktopState(state);
    });
  }, []);

  return <ProvidersPage desktopState={desktopState} />;
}
