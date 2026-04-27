export interface PortkeyTemplateEntry {
  name: string;
  model_id: string;
  base_url: string;
  provider_type: string;
  protocol: string;
  enabled: boolean;
  position: number;
}

export interface PortkeyTemplate {
  runtime_mode: string;
  runtime_url: string;
  generated_at: string;
  total_entries: number;
  enabled_count: number;
  disabled_count: number;
  entries: PortkeyTemplateEntry[];
}
