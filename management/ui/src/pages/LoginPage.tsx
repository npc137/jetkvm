import { useMsal, useIsAuthenticated } from "@azure/msal-react";
import { loginRequest } from "../authConfig";

export function LoginPage() {
  const { instance } = useMsal();
  const isAuthenticated = useIsAuthenticated();

  if (isAuthenticated) return null;

  return (
    <div
      style={{
        minHeight: "100vh",
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        fontFamily: "sans-serif",
        background: "#f8fafc",
      }}
    >
      <div
        style={{
          background: "#fff",
          borderRadius: 16,
          padding: 48,
          boxShadow: "0 4px 24px rgba(0,0,0,.10)",
          textAlign: "center",
          maxWidth: 360,
          width: "100%",
        }}
      >
        <h1 style={{ marginBottom: 8 }}>JetKVM</h1>
        <p style={{ color: "#64748b", marginBottom: 32 }}>Management Plane</p>
        <button
          onClick={() => instance.loginPopup(loginRequest)}
          style={{
            width: "100%",
            padding: "12px 0",
            fontSize: 16,
            fontWeight: 600,
            background: "#0f172a",
            color: "#fff",
            border: "none",
            borderRadius: 8,
            cursor: "pointer",
          }}
        >
          Sign in with Microsoft
        </button>
      </div>
    </div>
  );
}
