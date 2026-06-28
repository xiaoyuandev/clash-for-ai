import { useEffect } from "react";

export interface ToastItem {
  id: string;
  message: string;
  tone: "success" | "error" | "default";
}

interface ToastRegionProps {
  items: ToastItem[];
  onDismiss: (id: string) => void;
}

export function ToastRegion({ items, onDismiss }: ToastRegionProps) {
  useEffect(() => {
    if (items.length === 0) {
      return;
    }

    const timers = items.map((item) =>
      window.setTimeout(() => {
        onDismiss(item.id);
      }, 3200)
    );

    return () => {
      timers.forEach((timer) => window.clearTimeout(timer));
    };
  }, [items, onDismiss]);

  if (items.length === 0) {
    return null;
  }

  return (
    <div className="pointer-events-none fixed right-3 top-3 z-[60] grid w-[min(360px,calc(100vw-1.5rem))] gap-2">
      {items.map((item) => (
        <div
          key={item.id}
          className={`pointer-events-auto rounded-lg border px-3 py-2.5 shadow-[var(--shadow-panel)] ${
            item.tone === "success"
              ? "[border-color:var(--success-border)] [background:var(--success-soft)] text-[color:var(--success-text)]"
              : item.tone === "error"
                ? "[border-color:var(--danger-border)] [background:var(--danger-soft)] text-[color:var(--danger-text)]"
                : "[border-color:var(--border-soft)] [background:var(--panel-solid)] text-[color:var(--color-text)]"
          }`}
        >
          <div className="flex items-start gap-3">
            <span
              className={`mt-1 inline-flex h-2.5 w-2.5 shrink-0 rounded-full ${
                item.tone === "success"
                  ? "bg-[color:var(--accent-strong)]"
                  : item.tone === "error"
                    ? "bg-rose-400"
                    : "bg-[color:var(--color-subtle)]"
              }`}
            />
            <p className="text-sm leading-5">{item.message}</p>
          </div>
        </div>
      ))}
    </div>
  );
}
