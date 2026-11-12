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

export interface PortalSkill {
  name: string;
  bindings: InstalledBinding[];
}

export interface Portal {
  logged_in: boolean;
  account?: string;
  api?: string;
  sync_minutes: number;
  run: string;
  config_minutes: number;
  skills: PortalSkill[];
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
