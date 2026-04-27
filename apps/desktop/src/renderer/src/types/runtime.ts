export type RuntimeMode = "legacy" | "external-portkey";

export interface RuntimeConfig {
  mode: RuntimeMode;
  base_url: string;
}

export interface RuntimeHealth {
  mode: RuntimeMode;
  base_url: string;
  status: string;
  message: string;
  checked_at: string;
}

