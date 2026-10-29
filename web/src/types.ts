// Shapes of the JSON that internal/cli/ui.go serves.

export interface InstalledBinding {
  path: string;
  name: string;
  fires: number;
}

export interface InstalledPack {
  name: string;
  scope: "user" | "project";
  at: string;
  picks?: Record<string, string>;
  skills?: string[];
  bindings: InstalledBinding[];
}

export interface PickOne {
  name: string;
  description?: string;
  default: string;
  options: Record<string, string[]>;
}

export interface CatalogPack {
  name: string;
  description: string;
  tags: string[] | null;
  bindings: string[] | null;
  pick_one?: PickOne[];
  add: string;
}

export interface CatalogBinding {
  path: string;
  name: string;
  description: string;
  event: string;
  skill?: string;
  tags: string[] | null;
  upstream?: { repo: string; path: string; ref: string };
  provenance?: { stars?: number; license?: string };
  add: string;
}

export interface Catalog {
  packs: CatalogPack[];
  bindings: CatalogBinding[];
}

export interface RuleStats {
  rule: string;
  fires: number;
  denied: number;
  already_loaded: number;
  loaded_after_deny: number;
  delivered: number;
}

export interface Stats {
  rules: RuleStats[] | null;
  dormant: string[] | null;
  skills_invoked: Record<string, number> | null;
  sessions: number;
  turns: number;
  tool_calls: number;
}

export interface StatsResponse {
  stats: Stats;
  log: string;
}
