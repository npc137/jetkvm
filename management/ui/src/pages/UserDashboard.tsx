import { useCallback, useState } from "react";
import { useAuth, usePolling } from "../hooks/usePolling";
import * as api from "../api/client";
import type { Device } from "../api/types";

function StatusBadge({ status }: { status: Device["status"] }) {
  const colors: Record<Device["status"], string> = {
    online: "#22c55e",
    offline: "#ef4444",
    unknown: "#94a3b8",
  };
  return (
    <span
      style={{
        display: "inline-block",
        width: 10,
        height: 10,
        borderRadius: "50%",
        background: colors[status],
        marginRight: 6,
      }}
    />
  );
}

export function UserDashboard() {
  const { instance, account } = useAuth();
  const [connecting, setConnecting] = useState(false);
  const [connectError, setConnectError] = useState<string | null>(null);

  const fetcher = useCallback(
    () => api.getDevices(instance, account!),
    [instance, account]
  );
  const { data: devices, loading, error } = usePolling(fetcher);

  const device = devices?.[0] ?? null;

  async function handleConnect() {
    if (!device) return;
    setConnecting(true);
    setConnectError(null);
    try {
      const { connectUrl } = await api.requestConnect(instance, account!, device.id);
      window.location.href = connectUrl;
    } catch (e) {
      setConnectError(e instanceof Error ? e.message : "Connection failed");
    } finally {
      setConnecting(false);
    }
  }

  if (loading) return <p>Loading…</p>;
  if (error) return <p style={{ color: "red" }}>Error: {error}</p>;

  return (
    <div style={{ maxWidth: 480, margin: "60px auto", fontFamily: "sans-serif" }}>
      <h1>My KVM Access</h1>
      {!device ? (
        <p>You have no device assigned. Contact your administrator.</p>
      ) : (
        <div
          style={{
            border: "1px solid #e2e8f0",
            borderRadius: 12,
            padding: 24,
            boxShadow: "0 2px 8px rgba(0,0,0,.08)",
          }}
        >
          <h2 style={{ margin: "0 0 4px" }}>
            <StatusBadge status={device.status} />
            {device.name}
          </h2>
          {device.location && (
            <p style={{ margin: "0 0 16px", color: "#64748b" }}>{device.location}</p>
          )}
          <p style={{ margin: "0 0 4px", fontSize: 14, color: "#64748b" }}>
            Status: <strong>{device.status}</strong>
          </p>
          {device.lastSeenAt && (
            <p style={{ margin: "0 0 16px", fontSize: 14, color: "#64748b" }}>
              Last seen: {new Date(device.lastSeenAt).toLocaleString()}
            </p>
          )}
          <button
            onClick={handleConnect}
            disabled={device.status !== "online" || connecting}
            style={{
              width: "100%",
              padding: "12px 0",
              fontSize: 16,
              fontWeight: 600,
              background: device.status === "online" ? "#3b82f6" : "#94a3b8",
              color: "#fff",
              border: "none",
              borderRadius: 8,
              cursor: device.status === "online" ? "pointer" : "not-allowed",
            }}
          >
            {connecting ? "Connecting…" : "Connect"}
          </button>
          {connectError && (
            <p style={{ color: "red", marginTop: 8 }}>{connectError}</p>
          )}
          {device.status !== "online" && (
            <p style={{ color: "#94a3b8", fontSize: 13, marginTop: 8, textAlign: "center" }}>
              Device is {device.status} — connect is unavailable
            </p>
          )}
        </div>
      )}
    </div>
  );
}
