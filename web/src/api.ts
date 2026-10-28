import { useEffect, useState } from "react";

export type Load<T> = { state: "loading" } | { state: "ok"; data: T } | { state: "error"; error: string };

// useJSON fetches one read-only endpoint of `paws ui` once.
export function useJSON<T>(path: string): Load<T> {
  const [load, setLoad] = useState<Load<T>>({ state: "loading" });
  useEffect(() => {
    let live = true;
    fetch(path, { cache: "no-store" })
      .then((r) => (r.ok ? r.json() : Promise.reject(new Error(`${path}: HTTP ${r.status}`))))
      .then((data: T) => live && setLoad({ state: "ok", data }))
      .catch((e: Error) => live && setLoad({ state: "error", error: e.message }));
    return () => {
      live = false;
    };
  }, [path]);
  return load;
}
