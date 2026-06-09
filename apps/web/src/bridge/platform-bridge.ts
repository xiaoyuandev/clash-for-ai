export async function copyText(text: string) {
  await navigator.clipboard.writeText(text);
}

type DesktopBridge = {
  selectDirectory?: () => Promise<string | null>;
};

export function canSelectDirectory() {
  return typeof window !== "undefined" && Boolean((window as typeof window & { desktopBridge?: DesktopBridge }).desktopBridge?.selectDirectory);
}

export async function selectDirectory() {
  const bridge = (window as typeof window & { desktopBridge?: DesktopBridge }).desktopBridge;
  if (!bridge?.selectDirectory) {
    return null;
  }
  return bridge.selectDirectory();
}
