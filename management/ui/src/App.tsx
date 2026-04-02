import { useIsAuthenticated, useMsal, AuthenticatedTemplate, UnauthenticatedTemplate } from "@azure/msal-react";
import { useCallback, useEffect, useState } from "react";
import * as api from "./api/client";
import { UserDashboard } from "./pages/UserDashboard";
import { AdminDashboard } from "./pages/AdminDashboard";
import { LoginPage } from "./pages/LoginPage";

function AppContent() {
  const { instance, accounts } = useMsal();
  const account = accounts[0] ?? null;
  const [role, setRole] = useState<"admin" | "user" | null>(null);

  const loadRole = useCallback(async () => {
    if (!account) return;
    try {
      const me = await api.getMe(instance, account);
      setRole(me.role);
    } catch {
      setRole("user");
    }
  }, [instance, account]);

  useEffect(() => {
    loadRole();
  }, [loadRole]);

  function handleLogout() {
    instance.logoutPopup({ account: account ?? undefined });
  }

  return (
    <div>
      <nav
        style={{
          display: "flex",
          justifyContent: "flex-end",
          padding: "12px 24px",
          borderBottom: "1px solid #e2e8f0",
          fontFamily: "sans-serif",
          fontSize: 14,
          color: "#64748b",
          gap: 16,
          alignItems: "center",
        }}
      >
        <span>{account?.name}</span>
        <button
          onClick={handleLogout}
          style={{
            padding: "4px 12px",
            borderRadius: 6,
            border: "1px solid #e2e8f0",
            cursor: "pointer",
            background: "transparent",
          }}
        >
          Sign out
        </button>
      </nav>

      {role === "admin" ? <AdminDashboard /> : <UserDashboard />}
    </div>
  );
}

export default function App() {
  return (
    <>
      <AuthenticatedTemplate>
        <AppContent />
      </AuthenticatedTemplate>
      <UnauthenticatedTemplate>
        <LoginPage />
      </UnauthenticatedTemplate>
    </>
  );
}
