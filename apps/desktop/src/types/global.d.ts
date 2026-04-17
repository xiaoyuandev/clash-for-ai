interface DesktopBridge {
  ping: () => Promise<{
    ok: boolean;
    runtime: string;
    platform: string;
  }>;
}

declare global {
  interface Window {
    desktopBridge?: DesktopBridge;
  }
}

export {};
