export interface Check {
  target: string;
  status: string;
  latency_ms: number;
  checked_at: string;
}

export interface ApiResponse {
  checks: Check[];
  count: number;
}
