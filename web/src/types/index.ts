export interface Check {
  target: string;
  status: string;
  latency_ms: number;
  checked_at: string;
  uptime_sla: number;
}

export interface ApiResponse {
  checks: Check[];
  count: number;
}

export interface Target {
  id: number;
  name: string;
  url: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}
