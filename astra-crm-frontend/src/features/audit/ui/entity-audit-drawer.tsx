import { History } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";

type EntityAuditDrawerProps = {
  entityName: string;
};

export function EntityAuditDrawer({ entityName }: EntityAuditDrawerProps) {
  return (
    <Dialog>
      <DialogTrigger asChild>
        <Button type="button" variant="outline" size="sm">
          <History className="h-4 w-4" />
          Аудит
        </Button>
      </DialogTrigger>
      <DialogContent className="left-auto right-0 top-0 h-screen w-[min(520px,100vw)] translate-x-0 translate-y-0 rounded-none p-0">
        <DialogHeader className="border-b border-border p-5">
          <DialogTitle className="text-base font-semibold">Аудит: {entityName}</DialogTitle>
          <DialogDescription className="text-sm text-muted-foreground">
            История изменений будет подключена к API аудита.
          </DialogDescription>
        </DialogHeader>
        <div className="p-5">
          <Card className="p-4 text-sm text-muted-foreground">Событий пока нет.</Card>
        </div>
      </DialogContent>
    </Dialog>
  );
}
