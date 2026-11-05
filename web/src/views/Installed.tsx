import type { Load } from "../api";
import type { InstalledPack, Portal } from "../types";
import { Bar, Cmd, Empty, Loaded } from "../ui";

export function Installed({ load, portal }: { load: Load<InstalledPack[]>; portal: Load<Portal> }) {
  const fromPortal = portal.state === "ok" ? portal.data.skills : [];
  const run = portal.state === "ok" ? portal.data.run : "paws";
  return (
    <Loaded load={load}>
      {(packs) => {
        if (!packs.length && !fromPortal.length) {
          return (
            <Empty>
              No packs installed yet. Open <b>Catalog</b>, or run <Cmd text={`${run} add writing`} />
            </Empty>
          );
        }
        const max = Math.max(
          1,
          ...packs.flatMap((p) => p.bindings.map((b) => b.fires)),
          ...fromPortal.map((s) => s.bindings.reduce((n, b) => n + b.fires, 0)),
        );
        const portalCard = fromPortal.length > 0 && (
          <div className="card" key="portal">
            <div className="row">
              <h2 className="grow">From the portal</h2>
              <span className="badge">portal</span>
            </div>
            <p className="desc">Skills turned on in the paws portal. Change them there; paws picks up changes within 15 minutes, or run {run} sync.</p>
            <table>
              <thead>
                <tr>
                  <th>Skill</th>
                  <th className="num">Fired</th>
                  <th />
                </tr>
              </thead>
              <tbody>
                {fromPortal.map((s) => {
                  const fires = s.bindings.reduce((n, b) => n + b.fires, 0);
                  return (
                    <tr key={s.name}>
                      <td>
                        <code>{s.name}</code>
                      </td>
                      <td className="num">{fires}</td>
                      <td className="bar-cell">
                        <Bar value={fires} max={max} />
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        );
        const packCards = packs.map((p) => (
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
              <Cmd text={`${run} remove ${p.scope === "project" ? "--project " : ""}${p.name}`} />
            </p>
          </div>
        ));
        return [portalCard, ...packCards];
      }}
    </Loaded>
  );
}
