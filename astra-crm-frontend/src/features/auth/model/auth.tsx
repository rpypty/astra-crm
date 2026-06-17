import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createContext, type PropsWithChildren, useContext } from "react";
import type { CurrentUser } from "@/lib/domain";
import { api } from "@/lib/api";
import { queryKeys } from "@/lib/query-keys";

type AuthContextValue = {
  user?: CurrentUser;
  isLoading: boolean;
  isAuthenticated: boolean;
  login: (input: { login: string; password: string }) => Promise<CurrentUser>;
  logout: () => Promise<void>;
};

const AuthContext = createContext<AuthContextValue | undefined>(undefined);

export function AuthProvider({ children }: PropsWithChildren) {
  const queryClient = useQueryClient();
  const meQuery = useQuery({
    queryKey: queryKeys.auth.me,
    queryFn: api.auth.me,
    retry: false,
  });

  const loginMutation = useMutation({
    mutationFn: api.auth.login,
    onSuccess: (response) => {
      queryClient.setQueryData(queryKeys.auth.me, response);
    },
  });

  const logoutMutation = useMutation({
    mutationFn: api.auth.logout,
    onSuccess: () => {
      queryClient.clear();
    },
  });

  const user = meQuery.data?.user;

  return (
    <AuthContext.Provider
      value={{
        user,
        isLoading: meQuery.isLoading,
        isAuthenticated: Boolean(user),
        login: async (input) => {
          const response = await loginMutation.mutateAsync(input);
          return response.user;
        },
        logout: async () => {
          await logoutMutation.mutateAsync();
        },
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error("useAuth must be used inside AuthProvider");
  }
  return context;
}
