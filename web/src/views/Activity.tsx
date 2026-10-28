import type { Load } from "../api";
import type { StatsResponse } from "../types";
import { Empty, Loaded } from "../ui";

export function Activity({ load }: { load: Load<StatsResponse> }) {
  return (
    <Loaded load={load}>
      {({ stats: s }) => {
        const rules = s.rules ?? [];
        const dormant = s.dormant ?? [];
        const skills = Object.entries(s.skills_invoked ?? {}).sort((a, b) => b[1] - a[1]);
        const tiles: [string, number][] = [
          ["Fires", rules.reduce((n, r) => n + r.fires, 0)],
          ["Sessions", s.sessions],
          ["Turns", s.turns],
          ["Tool calls", s.tool_calls],
        ];
        return (
          <>
            <div className="tiles">
              {tiles.map(([label, value]) => (
                <div className="tile" key={label}>
                  <b>{value.toLocaleString()}</b>
                  <span>{label}</span>
                </div>
              ))}
            </div>
            <h3>By binding</h3>
            {rules.length ? (
              <div className="card scroll">
                <table>
                  <thead>
                    <tr>
                      <th>Binding</th>
                      <th className="num">Fires</th>
                      <th className="num" title="Denied until the skill was loaded">
                        Denied
                      </th>
                      <th className="num" title="The skill was loaded before paws asked">
                        Already loaded
                      </th>
                      <th className="num" title="Loaded after a deny in the same session">
                        Loaded after deny
                      </th>
                      <th className="num" title="The skill's digest arrived in the deny instead">
                        Digest delivered
                      </th>
                    </tr>
                  </thead>
                  <tbody>
                    {rules.map((r) => (
                      <tr key={r.rule}>
                        <td>
                          <code>{r.rule}</code>
                        </td>
                        <td className="num">{r.fires}</td>
                        <td className="num denied">{r.denied}</td>
                        <td className="num ok">{r.already_loaded}</td>
                        <td className="num">{r.loaded_after_deny}</td>
                        <td className="num">{r.delivered}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            ) : (
              <Empty>Nothing has fired yet. A binding fires when the agent tries a matching action.</Empty>
            )}
            {skills.length > 0 && (
              <>
                <h3>Skills invoked</h3>
                <div className="card">
                  {skills.map(([name, n]) => (
                    <div className="row" key={name}>
                      <code className="grow">{name}</code>
                      <span>{n}</span>
                    </div>
                  ))}
                </div>
              </>
            )}
            {dormant.length > 0 && (
              <>
                <h3>Never fired here</h3>
                <div className="card desc">
                  {dormant.map((d, i) => (
                    <span key={d}>
                      {i > 0 && " · "}
                      <code>{d}</code>
                    </span>
                  ))}
                </div>
              </>
            )}
          </>
        );
      }}
    </Loaded>
  );
}
