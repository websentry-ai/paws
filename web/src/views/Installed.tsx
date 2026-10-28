import type { Load } from "../api";
import type { InstalledPack } from "../types";
import { Bar, Cmd, Empty, Loaded } from "../ui";

export function Installed({ load }: { load: Load<InstalledPack[]> }) {
  return (
    <Loaded load={load}>
      {(packs) => {
        if (!packs.length) {
          return (
            <Empty>
              No packs installed yet. Open <b>Catalog</b>, or run <Cmd text="paws add writing" />
            </Empty>
          );
        }
        const max = Math.max(1, ...packs.flatMap((p) => p.bindings.map((b) => b.fires)));
        return packs.map((p) => (
          <div className="card" key={`${p.scope}/${p.name}`}>
            <div className="row">
              <h2 className="grow">{p.name}</h2>
              <span className="badge">{p.scope}</span>
            </div>
            <p className="desc">
              Installed {new Date(p.at).toLocaleString()}
              {p.skills?.length ? ` · skills ${p.skills.join(", ")}` : ""}
              {p.picks
                ? ` · ${Object.entries(p.picks)
                    .map(([group, option]) => `${group}: ${option}`)
                    .join(", ")}`
                : ""}
            </p>
            <table>
              <thead>
                <tr>
                  <th>Binding</th>
                  <th className="num">Fired</th>
                  <th />
                </tr>
              </thead>
              <tbody>
                {p.bindings.map((b) => (
                  <tr key={b.path}>
                    <td>
                      <code>{b.name}</code>
                    </td>
                    <td className="num">{b.fires}</td>
                    <td className="bar-cell">
                      <Bar value={b.fires} max={max} />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
            <p className="desc after-table">
              <Cmd text={`paws remove ${p.scope === "project" ? "--project " : ""}${p.name}`} />
            </p>
          </div>
        ));
      }}
    </Loaded>
  );
}
