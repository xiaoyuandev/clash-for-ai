export interface GatewayModel {
  id: string;
  name: string;
  model_id: string;
  base_url: string;
  api_key: string;
  provider_type: string;
  protocol: string;
  enabled: boolean;
  position: number;
}

export interface GatewayModelInput {
  name: string;
  model_id: string;
  base_url: string;
  api_key: string;
  provider_type: string;
  protocol: string;
  enabled: boolean;
}

