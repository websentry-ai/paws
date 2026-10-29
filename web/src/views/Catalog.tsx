import { useState } from "react";
import type { Load } from "../api";
import type { Catalog, CatalogBinding, CatalogPack } from "../types";
import { Cmd, Empty, Loaded } from "../ui";

const matches = (q: string, ...fields: (string | null | undefined)[]) =>
  !q || fields.join(" ").toLowerCase().includes(q);

export function CatalogView({ load }: { load: Load<Catalog> }) {
  const [query, setQuery] = useState("");
  const q = query.trim().toLowerCase();
  return (
    <>
      <input
        type="search"
        placeholder="Search packs, skills, tags…"
        aria-label="Search the catalog"
        value={query}
        onChange={(e) => setQuery(e.target.value)}
      />
      <Loaded load={load}>
        {(c) => {
          const byPath = new Map(c.bindings.map((b) => [b.path, b]));
          const packs = c.packs.filter((p) =>
            matches(q, p.name, p.description, (p.tags ?? []).join(" "), (p.bindings ?? []).join(" ")),
          );
          const bindings = c.bindings.filter((b) =>
            matches(q, b.name, b.description, b.skill, (b.tags ?? []).join(" "), b.path),
          );
          return (
            <>
              <h3>Packs</h3>
              {packs.length ? (
                packs.map((p) => <PackCard key={p.name} pack={p} byPath={byPath} />)
              ) : (
                <Empty>No pack matches.</Empty>
              )}
              <h3>Bindings</h3>
              {bindings.length ? <BindingTable bindings={bindings} /> : <Empty>No binding matches.</Empty>}
            </>
          );
        }}
      </Loaded>
    </>
  );
}

function PackCard({ pack: p, byPath }: { pack: CatalogPack; byPath: Map<string, CatalogBinding> }) {
  const rels = [...(p.bindings ?? []), ...(p.pick_one ?? []).flatMap((g) => Object.values(g.options).flat())];
  const skills = [...new Set(rels.map((rel) => byPath.get(rel)?.skill).filter(Boolean))];
  return (
    <div className="card">
      <div className="row">
        <h2 className="grow">{p.name}</h2>
        {(p.tags ?? []).map((t) => (
          <span className="tag" key={t}>
            {t}
          </span>
        ))}
      </div>
      <p className="desc">{p.description}</p>
      {skills.length > 0 && (
        <p className="desc">
          Skills:{" "}
          {skills.map((s, i) => (
            <span key={s}>
              {i > 0 && " · "}
              <code>{s}</code>
            </span>
          ))}
        </p>
      )}
      {(p.pick_one ?? []).map((g) => (
        <p className="desc" key={g.name}>
          Pick one ({g.name}):{" "}
          {Object.keys(g.options).map((o, i) => (
            <span key={o}>
              {i > 0 && " / "}
              {o === g.default ? <b>{o}</b> : o}
            </span>
          ))}
        </p>
      ))}
      <Cmd text={p.add} />
    </div>
  );
}

function BindingTable({ bindings }: { bindings: CatalogBinding[] }) {
  return (
    <div className="card scroll">
      <table>
        <thead>
          <tr>
            <th>Binding</th>
            <th>Event</th>
            <th>Skill</th>
            <th>Source</th>
          </tr>
        </thead>
        <tbody>
          {bindings.map((b) => (
            <tr key={b.path}>
              <td>
                <code>{b.name}</code>
                <div className="desc tight">{b.description}</div>
                <Cmd text={b.add} />
              </td>
              <td className="nowrap">{b.event}</td>
              <td className="nowrap">{b.skill && <code>{b.skill}</code>}</td>
              <td>
                {b.upstream ? (
                  <>
                    {b.upstream.repo}
                    <div className="desc tight">
                      {b.provenance?.stars ? `★ ${b.provenance.stars.toLocaleString()} · ` : ""}
                      {b.provenance?.license ?? ""}
                    </div>
                  </>
                ) : (
                  "paws"
                )}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
