export function getRuntimeLabel(
  runtime: string | null | undefined,
  labels: {
    desktopApp: string;
    browser: string;
    unknown: string;
  }
) {
  if (!runtime) {
    return labels.unknown;
  }

  const normalized = runtime.trim().toLowerCase();
  if (normalized === "electron" || normalized === "desktop") {
    return labels.desktopApp;
  }

  if (normalized === "browser") {
    return labels.browser;
  }

  return runtime;
}
