import { zodResolver } from "@hookform/resolvers/zod";
import { useNavigate } from "@tanstack/react-router";
import { Loader2 } from "lucide-react";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { useAuth } from "@/app/auth";
import { FormField } from "@/components/crm/form-field";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";

const loginSchema = z.object({
  login: z.string().min(1, "Введите логин"),
  password: z.string().min(1, "Введите пароль"),
});

type LoginForm = z.infer<typeof loginSchema>;

export function LoginPage() {
  const auth = useAuth();
  const navigate = useNavigate();
  const [serverError, setServerError] = useState<string | null>(null);
  const form = useForm<LoginForm>({
    resolver: zodResolver(loginSchema),
    defaultValues: { login: "teamlead", password: "demo123" },
  });

  async function onSubmit(values: LoginForm) {
    setServerError(null);
    try {
      const user = await auth.login(values);
      await navigate({ to: user.role === "teamlead" ? "/teamlead/dashboard" : "/trader/requisites" });
    } catch (error) {
      setServerError(error instanceof Error ? error.message : "Сервис временно недоступен. Попробуйте позже.");
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-background p-6">
      <Card className="w-full max-w-sm">
        <CardHeader>
          <div className="text-sm font-semibold uppercase tracking-normal text-primary">Astra CRM</div>
          <CardTitle className="text-xl">Вход в CRM</CardTitle>
          <p className="text-sm text-muted-foreground">Демо: teamlead/demo123 или trader_ivan/demo123</p>
        </CardHeader>
        <CardContent>
          <form className="space-y-4" onSubmit={form.handleSubmit(onSubmit)}>
            <FormField label="Логин" htmlFor="login" error={form.formState.errors.login?.message}>
              <Input id="login" autoComplete="username" {...form.register("login")} />
            </FormField>
            <FormField label="Пароль" htmlFor="password" error={form.formState.errors.password?.message}>
              <Input id="password" type="password" autoComplete="current-password" {...form.register("password")} />
            </FormField>
            {serverError ? (
              <div className="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-800">{serverError}</div>
            ) : null}
            <Button type="submit" className="w-full" disabled={form.formState.isSubmitting}>
              {form.formState.isSubmitting ? <Loader2 className="h-4 w-4 animate-spin" /> : null}
              Войти
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
