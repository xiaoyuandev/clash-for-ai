export function isObject(value) {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

export function optionalString(value, path) {
  if (value === undefined) {
    return undefined;
  }
  if (typeof value !== "string") {
    throw new Error(`${path} must be a string`);
  }
  return value.trim() ? value.trim() : undefined;
}

export function requiredString(value, path) {
  if (typeof value !== "string" || !value.trim()) {
    throw new Error(`${path} must be a non-empty string`);
  }
  return value.trim();
}

export function optionalBoolean(value, path) {
  if (value === undefined) {
    return undefined;
  }
  if (typeof value !== "boolean") {
    throw new Error(`${path} must be a boolean`);
  }
  return value;
}

export function optionalStringList(value, path) {
  if (value === undefined) {
    return undefined;
  }
  if (!Array.isArray(value)) {
    throw new Error(`${path} must be an array`);
  }
  return value.map((item, index) => requiredString(item, `${path}[${index}]`));
}

export function validateAbsoluteURL(value, path) {
  try {
    const parsed = new URL(value);
    if (!parsed.protocol || !parsed.host) {
      throw new Error("missing host");
    }
  } catch {
    throw new Error(`${path} must be a valid absolute URL`);
  }
}
