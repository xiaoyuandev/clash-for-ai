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
    <div className="pointer-events-none fixed right-5 top-5 z-[60] grid w-[min(360px,calc(100vw-2rem))] gap-3">
      {items.map((item) => (
        <div
          key={item.id}
          className={`pointer-events-auto rounded-3xl border px-4 py-3 shadow-[0_18px_40px_rgba(15,23,42,0.18)] backdrop-blur-xl ${
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
            <p className="text-sm leading-6">{item.message}</p>
          </div>
        </div>
      ))}
    </div>
  );
}
