export const appShellClass = "relative h-screen overflow-hidden bg-[var(--app-background)] text-[color:var(--color-text)]";

export const appBackdropClass =
  "pointer-events-none absolute inset-0 [background:var(--overlay-sheen)] transition-[background] duration-200";

export const glassPanelClass =
  "rounded-xl border [border-color:var(--border-soft)] [background:var(--panel-glass)] shadow-[var(--shadow-panel)] transition-[background,border-color,box-shadow] duration-200";

export const softPanelClass =
  "rounded-lg border [border-color:var(--border-soft)] [background:var(--panel-soft)] shadow-[var(--shadow-soft)] transition-[background,border-color,box-shadow] duration-200";

export const pageShellClass =
  "relative mx-auto flex h-full min-h-0 w-full max-w-[1600px] flex-col gap-3 overflow-y-auto px-2.5 py-2.5 sm:px-3 sm:py-3 xl:px-4";

export const heroClass =
  "flex flex-col gap-3 rounded-xl border [border-color:var(--border-soft)] [background:var(--panel-hero)] px-3 py-3 shadow-[var(--shadow-soft)] transition-[background,border-color,box-shadow] duration-200 lg:flex-row lg:items-center lg:justify-between";

export const eyebrowClass =
  "mb-1.5 text-[10px] font-semibold uppercase tracking-[0.18em] text-[color:var(--color-subtle)]";

export const heroTitleClass =
  "max-w-4xl text-xl font-semibold text-[color:var(--color-heading)] sm:text-2xl";

export const heroCopyClass = "max-w-3xl text-sm leading-5 text-[color:var(--color-muted)]";
export const heroContentClass = "space-y-2";
export const heroLabelStackClass = "space-y-1";

export const heroPillsClass = "flex flex-wrap items-center gap-2 lg:max-w-xl lg:justify-end";

export const pillBaseClass =
  "inline-flex min-h-7 items-center rounded-md border px-2.5 py-1 text-[11px] font-medium tracking-[0.01em] transition-[background,border-color,color] duration-200";

export function statusPillClass(variant: "default" | "success" | "danger" | "warning" = "default") {
  const variants = {
    default:
      "[border-color:var(--border-soft)] [background:var(--panel-solid)] text-[color:var(--color-muted)]",
    success:
      "[border-color:var(--success-border)] [background:var(--success-soft)] text-[color:var(--success-text)]",
    danger:
      "[border-color:var(--danger-border)] [background:var(--danger-soft)] text-[color:var(--danger-text)]",
    warning:
      "[border-color:var(--warning-border)] [background:var(--warning-soft)] text-[color:var(--warning-text)]"
  };

  return `${pillBaseClass} ${variants[variant]}`;
}

export function buttonClass(variant: "primary" | "secondary" | "danger" | "ghost" = "primary") {
  const base =
    "inline-flex min-h-8 cursor-pointer items-center justify-center rounded-lg border px-3 py-1.5 text-sm font-medium transition duration-200 ease-out focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[color:var(--accent-strong)]/55 focus-visible:ring-offset-2 focus-visible:ring-offset-[var(--app-background)] disabled:cursor-not-allowed disabled:opacity-50";
  const variants = {
    primary:
      "[border-color:var(--accent-strong)] [background:var(--accent)] text-[color:var(--accent-text)] shadow-[0_8px_18px_color-mix(in_srgb,var(--accent)_18%,transparent)] hover:brightness-105",
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
  return `${buttonClass(active ? "primary" : "ghost")} w-full justify-start gap-2 px-3 text-left`;
}

export const inputClass =
  "min-h-8 w-full rounded-lg border [border-color:var(--border-soft)] [background:var(--panel-input)] px-2.5 py-1.5 text-sm text-[color:var(--color-text)] outline-none transition placeholder:text-[color:var(--color-subtle)] focus:[border-color:var(--accent-strong)] focus:[background:var(--panel-input-focus)] focus:ring-2 focus:ring-[color:var(--accent-strong)]/20";

export const labelClass = "flex flex-col gap-1.5 text-sm text-[color:var(--color-text)]";
export const fieldLabelClass = "text-[11px] font-semibold uppercase tracking-[0.14em] text-[color:var(--color-subtle)]";
export const hintClass = "text-xs leading-5 text-[color:var(--color-muted)]";
export const metaClass = "text-sm leading-5 text-[color:var(--color-muted)]";
export const monoClass = "break-all font-mono text-[12px] leading-5 text-[color:var(--color-text)]";

export const sectionHeadClass = "flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between";
export const sectionTitleClass = "text-[16px] font-semibold text-[color:var(--color-heading)] sm:text-[18px]";
export const sectionMetaClass = "text-sm text-[color:var(--color-muted)]";

export const sectionCardClass = `${glassPanelClass} p-3`;
export const nestedCardClass = `${softPanelClass} p-3`;
export const infoCardClass =
  "rounded-lg border [border-color:var(--border-soft)] [background:var(--panel-solid)] p-3 transition-[background,border-color] duration-200";
export const emptyStateClass =
  "rounded-lg border border-dashed [border-color:var(--border-soft)] [background:var(--panel-solid)] px-3 py-5 text-sm leading-6 text-[color:var(--color-muted)]";
export const gridStatsClass = "grid gap-3 sm:grid-cols-2 xl:grid-cols-4";
export const compactStatGridClass = "grid gap-3 sm:grid-cols-2 xl:grid-cols-3";
export const iconBadgeClass =
  "inline-flex h-7 w-7 items-center justify-center rounded-lg border [border-color:var(--border-soft)] [background:var(--panel-soft)] text-[color:var(--accent)]";
export const metricValueClass =
  "mt-1.5 text-base font-semibold text-[color:var(--color-heading)]";
export const metricNumberClass =
  "mt-1.5 text-[20px] font-semibold text-[color:var(--color-heading)]";
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
  "rounded-lg border [border-color:var(--border-soft)] [background:var(--panel-soft)] p-3 shadow-[var(--shadow-soft)] transition-[background,border-color,box-shadow] duration-200";

export const successNoticeClass =
  "rounded-lg border [border-color:var(--success-border)] [background:var(--success-soft)] p-3 text-sm text-[color:var(--success-text)]";

export const dangerNoticeClass =
  "rounded-lg border [border-color:var(--danger-border)] [background:var(--danger-soft)] p-3 text-sm text-[color:var(--danger-text)]";

export function selectableItemClass(active: boolean) {
  return [
    "w-full cursor-pointer rounded-lg border p-3 text-left transition duration-200",
    active
      ? "[border-color:var(--accent-strong)] [background:var(--accent-soft)] shadow-[0_8px_18px_color-mix(in_srgb,var(--accent)_12%,transparent)]"
      : "[border-color:var(--border-soft)] [background:var(--panel-solid)] hover:[border-color:var(--border-strong)] hover:[background:var(--panel-soft)]"
  ].join(" ");
}

export const listClass = "grid gap-2.5";
export const splitLayoutClass = "grid gap-3 lg:grid-cols-[280px_minmax(0,1fr)] 2xl:grid-cols-[304px_minmax(0,1fr)]";
export const actionRowClass = "flex flex-wrap items-center gap-2";
export const stickySearchClass =
  "sticky top-0 z-10 -mx-2 rounded-lg px-2 py-1.5 [background:var(--panel-glass)]";
export const columnCardClass = `${nestedCardClass} flex min-h-0 min-w-0 flex-col overflow-hidden`;
export const scrollListClass = "grid min-h-0 flex-1 gap-2.5 overflow-y-auto pr-1";
export const queueItemClass =
  "relative flex items-start justify-between gap-3 rounded-lg border [border-color:var(--border-soft)] [background:var(--panel-solid)] p-3 transition hover:[border-color:var(--border-strong)] hover:[background:var(--panel-soft)]";
export const iconButtonClass =
  "inline-flex min-h-8 min-w-8 cursor-pointer items-center justify-center rounded-lg border [border-color:var(--border-soft)] [background:var(--panel-solid)] text-[color:var(--color-text)] transition hover:[border-color:var(--border-strong)] hover:[background:var(--panel-soft)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[color:var(--accent-strong)]/55 focus-visible:ring-offset-2 focus-visible:ring-offset-[var(--app-background)]";
export const iconButtonSmallClass =
  "inline-flex min-h-7 min-w-7 cursor-pointer items-center justify-center rounded-md border [border-color:var(--border-soft)] [background:var(--panel-solid)] text-[color:var(--color-text)] transition hover:[border-color:var(--border-strong)] hover:[background:var(--panel-soft)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[color:var(--accent-strong)]/55 focus-visible:ring-offset-2 focus-visible:ring-offset-[var(--app-background)]";
export const modalBackdropClass =
  "fixed inset-0 z-50 grid place-items-center px-4 py-6 backdrop-blur-sm [background:var(--modal-backdrop)]";
export const modalPanelClass = `${glassPanelClass} max-h-[calc(100vh-3rem)] w-full max-w-4xl overflow-auto p-4`;
export const floatingModalPanelClass = `${glassPanelClass} max-h-[calc(100vh-3rem)] w-full max-w-4xl overflow-visible p-4`;
