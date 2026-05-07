export interface RequestLog {
  id: string;
  timestamp: string;
  provider_id: string;
  provider_name: string;
  method: string;
  path: string;
  model?: string;
  status_code?: number;
  is_stream: boolean;
  upstream_host: string;
  latency_ms: number;
  first_byte_ms?: number;
  first_token_ms?: number;
  error_type?: string;
  error_message?: string;
  error_snippet?: string;
}
