export function normalizeVersion(value?: string) {
  return (value ?? "").trim().replace(/^v/i, "");
}

export function compareVersions(current?: string, latest?: string) {
  const currentParts = normalizeVersion(current).split(".").map((part) => Number.parseInt(part, 10));
  const latestParts = normalizeVersion(latest).split(".").map((part) => Number.parseInt(part, 10));

  if (!currentParts.length || !latestParts.length || currentParts.some(Number.isNaN) || latestParts.some(Number.isNaN)) {
    return 0;
  }

  const length = Math.max(currentParts.length, latestParts.length);
  for (let index = 0; index < length; index += 1) {
    const currentValue = currentParts[index] ?? 0;
    const latestValue = latestParts[index] ?? 0;

    if (latestValue > currentValue) {
      return 1;
    }

    if (latestValue < currentValue) {
      return -1;
    }
  }

  return 0;
}
