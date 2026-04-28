export interface ModelSource {
  id: string;
  name: string;
  base_url: string;
  provider_type: string;
  default_model_id: string;
  enabled: boolean;
  position: number;
  api_key: string;
  api_key_masked: string;
}

export interface ModelSourceInput {
  name: string;
  base_url: string;
  provider_type: string;
  default_model_id: string;
  enabled: boolean;
  api_key: string;
}

