import { useEffect, useState } from "react";
import { useJSON } from "./api";
import type { Catalog, InstalledPack, StatsResponse } from "./types";
import { Activity } from "./views/Activity";
import { CatalogView } from "./views/Catalog";
import { Installed } from "./views/Installed";

const tabs = [
  { id: "installed", label: "Installed" },
  { id: "catalog", label: "Catalog" },
  { id: "activity", label: "Activity" },
] as const;
type Tab = (typeof tabs)[number]["id"];

const isTab = (s: string | null): s is Tab => tabs.some((t) => t.id === s);

// startTab honours a #catalog / #activity deep link, else reopens the last tab.
function startTab(): Tab {
  const hash = location.hash.slice(1);
  if (isTab(hash)) return hash;
  try {
    const saved = localStorage.getItem("paws-tab");
    if (isTab(saved)) return saved;
  } catch {
    // Storage can be unavailable (private window); the default tab is fine.
  }
  return "installed";
}

export function App() {
  const [tab, setTab] = useState<Tab>(startTab);
  const installed = useJSON<InstalledPack[]>("/api/installed");
  const catalog = useJSON<Catalog>("/api/catalog");
  const stats = useJSON<StatsResponse>("/api/stats");

  useEffect(() => {
    const onHash = () => {
      const hash = location.hash.slice(1);
      if (isTab(hash)) setTab(hash);
    };
    window.addEventListener("hashchange", onHash);
    return () => window.removeEventListener("hashchange", onHash);
  }, []);

  useEffect(() => {
    history.replaceState(null, "", `#${tab}`);
    try {
      localStorage.setItem("paws-tab", tab);
    } catch {
      // Not remembering the tab is harmless.
    }
  }, [tab]);

  return (
    <>
      <header className="page">
        <h1>paws</h1>
        <p>Skills that actually fire. Everything here is read from this machine.</p>
      </header>
      <nav className="page" role="tablist">
        {tabs.map((t) => (
          <button key={t.id} role="tab" aria-selected={tab === t.id} onClick={() => setTab(t.id)}>
            {t.label}
          </button>
        ))}
      </nav>
      <main className="page">
        {tab === "installed" && <Installed load={installed} />}
        {tab === "catalog" && <CatalogView load={catalog} />}
        {tab === "activity" && <Activity load={stats} />}
      </main>
      <footer className="page">
        {stats.state === "ok" ? `Audit log: ${stats.data.log}. ` : ""}Nothing leaves this machine.
      </footer>
    </>
  );
}
