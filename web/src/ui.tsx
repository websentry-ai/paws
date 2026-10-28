import { useState, type ReactNode } from "react";
import type { Load } from "./api";

// Cmd is a shell command with a copy button.
export function Cmd({ text }: { text: string }) {
  const [copied, setCopied] = useState(false);
  const copy = () =>
    navigator.clipboard.writeText(text).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 1200);
    });
  return (
    <span className="cmd">
      <code>{text}</code>
      <button onClick={copy}>{copied ? "Copied" : "Copy"}</button>
    </span>
  );
}

export function Empty({ children }: { children: ReactNode }) {
  return <div className="empty">{children}</div>;
}

// Loaded renders a fetch's loading and error states, and its data through children.
export function Loaded<T>({ load, children }: { load: Load<T>; children: (data: T) => ReactNode }) {
  if (load.state === "loading") return <Empty>Loading…</Empty>;
  if (load.state === "error") return <Empty>{load.error}</Empty>;
  return <>{children(load.data)}</>;
}

export function Bar({ value, max }: { value: number; max: number }) {
  return (
    <div className="bar">
      <span style={{ width: `${(100 * value) / Math.max(1, max)}%` }} />
    </div>
  );
}
