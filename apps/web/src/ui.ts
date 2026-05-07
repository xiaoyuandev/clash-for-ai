export const appShellClass = "relative h-screen overflow-hidden text-[color:var(--color-text)]";

export const appBackdropClass =
  "pointer-events-none absolute inset-0 [background:var(--overlay-sheen)] transition-[background] duration-300";

export const glassPanelClass =
  "rounded-[20px] border [border-color:var(--border-soft)] [background:var(--panel-glass)] shadow-[var(--shadow-panel)] backdrop-blur-xl transition-[background,border-color,box-shadow] duration-300";

export const softPanelClass =
  "rounded-[18px] border [border-color:var(--border-soft)] [background:var(--panel-soft)] shadow-[var(--shadow-soft)] backdrop-blur-lg transition-[background,border-color,box-shadow] duration-300";

export const pageShellClass =
  "relative mx-auto flex h-full min-h-0 w-full max-w-[1600px] flex-col gap-4 overflow-y-auto px-3 py-3 sm:px-4 sm:py-4 xl:px-6";

export const heroClass =
  "flex flex-col gap-3 rounded-[22px] border [border-color:var(--border-soft)] [background:var(--panel-hero)] px-4 py-4 shadow-[var(--shadow-panel)] backdrop-blur-2xl transition-[background,border-color,box-shadow] duration-300 lg:flex-row lg:items-start lg:justify-between lg:px-5 lg:py-4";

export const eyebrowClass =
  "mb-3 text-[11px] font-semibold uppercase tracking-[0.28em] text-[color:var(--accent)]/80";

export const heroTitleClass =
  "max-w-4xl text-[28px] font-semibold tracking-[-0.04em] text-[color:var(--color-heading)] sm:text-[32px] xl:text-[38px]";

export const heroCopyClass = "max-w-2xl text-sm leading-6 text-[color:var(--color-muted)]";
export const heroContentClass = "space-y-3";
export const heroLabelStackClass = "space-y-1";

export const heroPillsClass = "flex flex-wrap items-center gap-2 lg:max-w-sm lg:justify-end";

export const pillBaseClass =
  "inline-flex min-h-9 items-center rounded-full border px-3 py-1.5 text-[11px] font-medium tracking-[0.02em] backdrop-blur-md transition-[background,border-color,color] duration-300";

export function statusPillClass(variant: "default" | "success" | "danger" | "warning" = "default") {
  const variants = {
    default:
      "[border-color:var(--border-soft)] [background:var(--panel-solid)] text-[color:var(--color-muted)]",
    success:
      "[border-color:var(--success-border)] [background:var(--success-soft)] text-[color:var(--success-text)]",
    danger:
      "[border-color:var(--danger-border)] [background:var(--danger-soft)] text-[color:var(--danger-text)]",
    warning:
      "[border-color:var(--success-border)] [background:color-mix(in_srgb,var(--success-soft)_70%,transparent)] text-[color:var(--accent)]"
  };

  return `${pillBaseClass} ${variants[variant]}`;
}

export function buttonClass(variant: "primary" | "secondary" | "danger" | "ghost" = "primary") {
  const base =
    "inline-flex min-h-9 items-center justify-center rounded-xl border px-3.5 py-2 text-sm font-medium transition duration-200 ease-out focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[color:var(--accent-strong)]/55 focus-visible:ring-offset-2 focus-visible:ring-offset-transparent disabled:cursor-not-allowed disabled:opacity-50";
  const variants = {
    primary:
      "[border-color:var(--accent-strong)]/25 [background:linear-gradient(135deg,var(--accent)_0%,var(--accent-strong)_100%)] text-[color:var(--accent-text)] shadow-[0_14px_32px_color-mix(in_srgb,var(--accent)_24%,transparent)] hover:brightness-105",
    secondary:
      "[border-color:var(--border-soft)] [background:var(--panel-solid)] text-[color:var(--color-text)] hover:[border-color:var(--border-strong)] hover:[background:var(--panel-soft)]",
    danger:
      "[border-color:var(--danger-border)] [background:var(--danger-soft)] text-[color:var(--danger-text)] hover:brightness-105",
    ghost:
      "border-transparent bg-transparent text-[color:var(--color-muted)] hover:[border-color:var(--border-soft)] hover:[background:var(--panel-solid)]"
  };

  return `${base} ${variants[variant]}`;
}

export function navButtonClass(active: boolean) {
  return `${buttonClass(active ? "primary" : "ghost")} w-full justify-start px-4 text-left`;
}

export const inputClass =
  "min-h-9 w-full rounded-xl border [border-color:var(--border-soft)] [background:var(--panel-input)] px-3 py-2 text-sm text-[color:var(--color-text)] outline-none transition placeholder:text-[color:var(--color-subtle)] focus:[border-color:var(--accent-strong)] focus:[background:var(--panel-input-focus)] focus:ring-2 focus:ring-[color:var(--accent-strong)]/20";

export const labelClass = "flex flex-col gap-2 text-sm text-[color:var(--color-text)]";
export const fieldLabelClass = "text-[11px] font-semibold uppercase tracking-[0.22em] text-[color:var(--color-subtle)]";
export const hintClass = "text-xs leading-5 text-[color:var(--color-muted)]";
export const metaClass = "text-sm leading-5 text-[color:var(--color-muted)]";
export const monoClass = "font-mono text-[12px] leading-5 text-[color:var(--color-text)] break-all";

export const sectionHeadClass = "flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between";
export const sectionTitleClass = "text-[17px] font-semibold tracking-[-0.02em] text-[color:var(--color-heading)] sm:text-[19px]";
export const sectionMetaClass = "text-sm text-[color:var(--color-muted)]";

export const sectionCardClass = `${glassPanelClass} p-4 sm:p-4.5`;
export const nestedCardClass = `${softPanelClass} p-3.5 sm:p-4`;
export const infoCardClass =
  "rounded-[16px] border [border-color:var(--border-soft)] [background:var(--panel-solid)] p-3.5 backdrop-blur-md transition-[background,border-color] duration-300";
export const emptyStateClass =
  "rounded-[16px] border border-dashed [border-color:var(--border-soft)] [background:var(--panel-solid)] px-4 py-6 text-sm leading-6 text-[color:var(--color-muted)]";
export const gridStatsClass = "grid gap-4 sm:grid-cols-2 xl:grid-cols-4";
export const compactStatGridClass = "grid gap-3 sm:grid-cols-2 xl:grid-cols-3";
export const iconBadgeClass =
  "inline-flex h-8 w-8 items-center justify-center rounded-xl border [border-color:var(--border-soft)] [background:var(--panel-solid)] text-[color:var(--accent)]";
export const metricValueClass =
  "mt-2 text-base font-semibold tracking-[-0.02em] text-[color:var(--color-heading)]";
export const metricNumberClass =
  "mt-2 text-[22px] font-semibold tracking-[-0.03em] text-[color:var(--color-heading)]";
export function statusDotClass(variant: "default" | "success" | "danger" | "warning" = "default") {
  const variants = {
    default: "bg-[color:var(--color-subtle)]/65",
    success: "bg-[color:var(--accent-strong)]",
    danger: "bg-rose-400",
    warning: "bg-amber-400"
  };

  return `inline-flex h-2.5 w-2.5 rounded-full ${variants[variant]}`;
}

export const surfaceCardClass =
  "rounded-[18px] border [border-color:var(--border-soft)] [background:var(--panel-soft)] p-4 shadow-[var(--shadow-soft)] backdrop-blur-lg transition-[background,border-color,box-shadow] duration-300";

export const successNoticeClass =
  "rounded-[18px] border [border-color:var(--success-border)] [background:var(--success-soft)] p-3 text-sm text-[color:var(--success-text)] backdrop-blur-xl";

export const dangerNoticeClass =
  "rounded-[18px] border [border-color:var(--danger-border)] [background:var(--danger-soft)] p-3 text-sm text-[color:var(--danger-text)] backdrop-blur-xl";

export function selectableItemClass(active: boolean) {
  return [
    "w-full rounded-[16px] border p-3.5 text-left transition duration-200",
    active
      ? "[border-color:var(--accent-strong)]/25 [background:color-mix(in_srgb,var(--accent-soft)_92%,transparent)] shadow-[0_10px_24px_color-mix(in_srgb,var(--accent)_12%,transparent)]"
      : "[border-color:var(--border-soft)] [background:var(--panel-solid)] hover:[border-color:var(--border-strong)] hover:[background:var(--panel-soft)]"
  ].join(" ");
}

export const listClass = "grid gap-3";
export const splitLayoutClass = "grid gap-3 xl:grid-cols-[320px_minmax(0,1fr)]";
export const actionRowClass = "flex flex-wrap items-center gap-2";
export const stickySearchClass =
  "sticky top-0 z-10 -mx-2 rounded-xl px-2 py-1.5 backdrop-blur-xl [background:color-mix(in_srgb,var(--panel-solid)_88%,transparent)]";
export const columnCardClass = `${nestedCardClass} flex min-h-0 min-w-0 flex-col overflow-hidden`;
export const scrollListClass = "grid min-h-0 flex-1 gap-3 overflow-y-auto pr-1";
export const queueItemClass =
  "relative flex items-start justify-between gap-3 rounded-[16px] border [border-color:var(--border-soft)] [background:var(--panel-solid)] p-3.5 transition hover:[border-color:var(--border-strong)] hover:[background:var(--panel-soft)]";
export const iconButtonClass =
  "inline-flex min-h-9 min-w-9 items-center justify-center rounded-xl border [border-color:var(--border-soft)] [background:var(--panel-solid)] text-[color:var(--color-text)] transition hover:[border-color:var(--border-strong)] hover:[background:var(--panel-soft)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[color:var(--accent-strong)]/55 focus-visible:ring-offset-2 focus-visible:ring-offset-transparent";
export const iconButtonSmallClass =
  "inline-flex min-h-8 min-w-8 items-center justify-center rounded-lg border [border-color:var(--border-soft)] [background:var(--panel-solid)] text-[color:var(--color-text)] transition hover:[border-color:var(--border-strong)] hover:[background:var(--panel-soft)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[color:var(--accent-strong)]/55 focus-visible:ring-offset-2 focus-visible:ring-offset-transparent";
export const modalBackdropClass =
  "fixed inset-0 z-50 grid place-items-center px-4 py-6 backdrop-blur-md [background:var(--modal-backdrop)]";
export const modalPanelClass = `${glassPanelClass} max-h-[calc(100vh-3rem)] w-full max-w-4xl overflow-auto p-4 sm:p-5`;
