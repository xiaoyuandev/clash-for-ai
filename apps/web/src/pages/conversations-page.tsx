import {
  ConversationsPage as SharedConversationsPage,
  type ConversationsPageDependencies
} from "../shared/conversations-page";
import { useCallback, useMemo } from "react";
import claudeLogo from "../assets/conversations/claude-color.svg";
import codexLogo from "../assets/conversations/codex-color.svg";
import { ToastRegion } from "../components/toast-region";
import { useI18n } from "../i18n/i18n-provider";
import {
  getConversationBackupConfig,
  getConversationCatalog,
  getConversationSession,
  runConversationBackupNow,
  updateConversationBackupConfig
} from "../services/api";
import {
  buttonClass,
  emptyStateClass,
  fieldLabelClass,
  heroTitleClass,
  iconButtonClass,
  inputClass,
  metaClass,
  modalBackdropClass,
  modalPanelClass,
  monoClass
} from "../ui";

interface ConversationsPageProps {
  apiBase?: string;
  onCopyText?: (text: string) => Promise<void>;
  onSelectBackupDirectory?: () => Promise<string | null>;
  onOpenBackupDirectory?: (path: string) => Promise<void>;
}

export function ConversationsPage(props: ConversationsPageProps) {
  const { locale, t } = useI18n();
  const translate = useCallback(
    (key: string, values?: Record<string, string | number>) =>
      t(key as Parameters<typeof t>[0], values),
    [t]
  );
  const dependencies = useMemo<ConversationsPageDependencies>(
    () => ({
      api: {
        getConversationBackupConfig,
        getConversationCatalog,
        getConversationSession,
        runConversationBackupNow,
        updateConversationBackupConfig
      },
      locale,
      logos: {
        "claude-code": claudeLogo,
        codex: codexLogo
      },
      t: translate,
      ToastRegion,
      ui: {
        buttonClass,
        emptyStateClass,
        fieldLabelClass,
        heroTitleClass,
        iconButtonClass,
        inputClass,
        metaClass,
        modalBackdropClass,
        modalPanelClass,
        monoClass
      }
    }),
    [locale, translate]
  );

  return <SharedConversationsPage {...props} dependencies={dependencies} />;
}
