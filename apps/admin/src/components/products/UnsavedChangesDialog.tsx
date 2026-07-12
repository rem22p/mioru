/**
 * UnsavedChangesDialog — generic two-button confirmation built on Radix Dialog
 * (so we get focus trap, Esc handling, and proper accessibility for free).
 *
 * One component, two variants:
 * - `restore` — "У вас есть несохранённый черновик товара от {date}.
 *   Восстановить его?" Primary action = restore, cancel = start fresh.
 * - `close`   — "У вас есть несохранённые изменения. Закрыть без сохранения?"
 *   Destructive action = close, cancel = stay in the form.
 *
 * The `open` state lives in the parent (ProductForm) so the parent can decide
 * whether to render. Outside-clicks and Esc are wired to `onCancel` because
 * both dialogs are functionally the same: confirm the side-effect or stay.
 */
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";

export type DialogVariant = "restore" | "close";

interface UnsavedChangesDialogProps {
  open: boolean;
  variant: DialogVariant;
  /** ISO 8601 timestamp — only meaningful for `variant === "restore"`. */
  draftSavedAt?: string;
  onConfirm: () => void;
  onCancel: () => void;
}

const DATE_OPTIONS: Intl.DateTimeFormatOptions = {
  day: "2-digit",
  month: "2-digit",
  year: "numeric",
  hour: "2-digit",
  minute: "2-digit",
};

function formatRuDate(iso: string): string {
  return new Date(iso).toLocaleString("ru-RU", DATE_OPTIONS);
}

export default function UnsavedChangesDialog({
  open,
  variant,
  draftSavedAt,
  onConfirm,
  onCancel,
}: UnsavedChangesDialogProps) {
  const isRestore = variant === "restore";
  const title = isRestore
    ? "Восстановить черновик?"
    : "Закрыть без сохранения?";
  const description = isRestore
    ? `У вас есть несохранённый черновик товара${draftSavedAt ? ` от ${formatRuDate(draftSavedAt)}` : ""}. Восстановить его?`
    : "У вас есть несохранённые изменения. Закрыть без сохранения?";
  const confirmLabel = isRestore ? "Восстановить" : "Закрыть без сохранения";
  const confirmVariant = isRestore ? "default" : "destructive";

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (!next) onCancel();
      }}
    >
      <DialogContent
        data-testid={isRestore ? "pf-restore-dialog" : "pf-close-dialog"}
      >
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription>{description}</DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            onClick={onCancel}
            data-testid={isRestore ? "pf-restore-cancel" : "pf-close-cancel"}
          >
            Отмена
          </Button>
          <Button
            type="button"
            variant={confirmVariant}
            onClick={onConfirm}
            data-testid={
              isRestore ? "pf-restore-confirm" : "pf-close-confirm"
            }
          >
            {confirmLabel}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
