import { useCallback, useState } from "react";
import { useAuth, usePolling } from "../hooks/usePolling";
import * as api from "../api/client";
import type { Device, Assignment, AdminUser } from "../api/types";

function StatusDot({ status }: { status: Device["status"] }) {
  const c = { online: "#22c55e", offline: "#ef4444", unknown: "#94a3b8" } as const;
  return (
    <span
      style={{
        display: "inline-block",
        width: 9,
        height: 9,
        borderRadius: "50%",
        background: c[status],
        marginRight: 6,
      }}
    />
  );
}

export function AdminDashboard() {
  const { instance, account } = useAuth();

  const deviceFetcher = useCallback(
    () => api.getDevices(instance, account!),
    [instance, account]
  );
  const assignmentFetcher = useCallback(
    () => api.getAssignments(instance, account!),
    [instance, account]
  );
  const userFetcher = useCallback(
    () => api.getAdminUsers(instance, account!),
    [instance, account]
  );

  const { data: devices, loading: dLoading, refetch: refetchDevices } = usePolling(deviceFetcher);
  const { data: assignments, refetch: refetchAssignments } = usePolling(assignmentFetcher);
  const { data: users } = usePolling(userFetcher);

  // ── Assignment modal state ─────────────────────────────────────────────────
  const [assignModal, setAssignModal] = useState<{
    deviceId: string;
    deviceName: string;
  } | null>(null);
  const [assignForm, setAssignForm] = useState({
    userOid: "",
    validFrom: "",
    validUntil: "",
  });
  const [assignError, setAssignError] = useState<string | null>(null);

  // ── Add device modal state ─────────────────────────────────────────────────
  const [addDeviceModal, setAddDeviceModal] = useState(false);
  const [addDeviceForm, setAddDeviceForm] = useState({
    name: "",
    ipAddress: "",
    location: "",
    password: "",
  });
  const [addDeviceError, setAddDeviceError] = useState<string | null>(null);

  function activeAssignment(deviceId: string): Assignment | undefined {
    const now = new Date().toISOString();
    return assignments?.find(
      (a) =>
        a.deviceId === deviceId &&
        a.validFrom <= now &&
        a.validUntil >= now
    );
  }

  function userName(oid: string): string {
    return users?.find((u) => u.oid === oid)?.displayName ?? oid;
  }

  async function handleAssign(e: React.FormEvent) {
    e.preventDefault();
    setAssignError(null);
    if (!assignModal) return;
    try {
      await api.createAssignment(instance, account!, {
        userOid: assignForm.userOid,
        deviceId: assignModal.deviceId,
        validFrom: new Date(assignForm.validFrom).toISOString(),
        validUntil: new Date(assignForm.validUntil).toISOString(),
      });
      setAssignModal(null);
      refetchAssignments();
    } catch (e) {
      setAssignError(e instanceof Error ? e.message : "Failed");
    }
  }

  async function handleRemoveAssignment(id: string) {
    await api.deleteAssignment(instance, account!, id);
    refetchAssignments();
  }

  async function handleAddDevice(e: React.FormEvent) {
    e.preventDefault();
    setAddDeviceError(null);
    try {
      await api.createDevice(instance, account!, addDeviceForm);
      setAddDeviceModal(false);
      setAddDeviceForm({ name: "", ipAddress: "", location: "", password: "" });
      refetchDevices();
    } catch (e) {
      setAddDeviceError(e instanceof Error ? e.message : "Failed");
    }
  }

  async function handleDeleteDevice(id: string) {
    if (!confirm("Remove this device?")) return;
    await api.deleteDevice(instance, account!, id);
    refetchDevices();
  }

  if (dLoading) return <p>Loading…</p>;

  const style: Record<string, React.CSSProperties> = {
    container: { maxWidth: 900, margin: "40px auto", fontFamily: "sans-serif" },
    table: { width: "100%", borderCollapse: "collapse", fontSize: 14 },
    th: { textAlign: "left", borderBottom: "2px solid #e2e8f0", padding: "8px 12px", color: "#64748b" },
    td: { padding: "10px 12px", borderBottom: "1px solid #f1f5f9" },
    btn: { padding: "4px 10px", borderRadius: 6, border: "none", cursor: "pointer", fontSize: 13 },
    modal: {
      position: "fixed", inset: 0, background: "rgba(0,0,0,.4)",
      display: "flex", alignItems: "center", justifyContent: "center", zIndex: 100,
    },
    modalBox: { background: "#fff", borderRadius: 12, padding: 28, minWidth: 340 },
  };

  return (
    <div style={style.container}>
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
        <h1>Admin Dashboard</h1>
        <button
          style={{ ...style.btn, background: "#3b82f6", color: "#fff", padding: "8px 16px" }}
          onClick={() => setAddDeviceModal(true)}
        >
          + Add Device
        </button>
      </div>

      {/* Device Table */}
      <table style={style.table}>
        <thead>
          <tr>
            <th style={style.th}>Device</th>
            <th style={style.th}>Location</th>
            <th style={style.th}>Status</th>
            <th style={style.th}>Last Seen</th>
            <th style={style.th}>Assigned To</th>
            <th style={style.th}>Actions</th>
          </tr>
        </thead>
        <tbody>
          {(devices ?? []).map((d) => {
            const asgn = activeAssignment(d.id);
            return (
              <tr key={d.id}>
                <td style={style.td}>
                  <strong>{d.name}</strong>
                  {d.fwVersion && (
                    <span style={{ color: "#94a3b8", marginLeft: 8, fontSize: 12 }}>
                      v{d.fwVersion}
                    </span>
                  )}
                </td>
                <td style={style.td}>{d.location ?? "—"}</td>
                <td style={style.td}>
                  <StatusDot status={d.status} />
                  {d.status}
                </td>
                <td style={style.td}>
                  {d.lastSeenAt ? new Date(d.lastSeenAt).toLocaleString() : "—"}
                </td>
                <td style={style.td}>
                  {asgn ? (
                    <span>
                      {userName(asgn.userOid)}{" "}
                      <button
                        style={{ ...style.btn, background: "#fef2f2", color: "#ef4444" }}
                        onClick={() => handleRemoveAssignment(asgn.id)}
                      >
                        Remove
                      </button>
                    </span>
                  ) : (
                    <em style={{ color: "#94a3b8" }}>Unassigned</em>
                  )}
                </td>
                <td style={style.td}>
                  <button
                    style={{ ...style.btn, background: "#eff6ff", color: "#3b82f6", marginRight: 6 }}
                    onClick={() => setAssignModal({ deviceId: d.id, deviceName: d.name })}
                  >
                    Assign
                  </button>
                  <button
                    style={{ ...style.btn, background: "#fef2f2", color: "#ef4444" }}
                    onClick={() => handleDeleteDevice(d.id)}
                  >
                    Delete
                  </button>
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>

      {/* Assignment Modal */}
      {assignModal && (
        <div style={style.modal} onClick={() => setAssignModal(null)}>
          <div style={style.modalBox} onClick={(e) => e.stopPropagation()}>
            <h2 style={{ marginTop: 0 }}>Assign {assignModal.deviceName}</h2>
            <form onSubmit={handleAssign}>
              <label style={{ display: "block", marginBottom: 4 }}>User</label>
              <select
                required
                style={{ width: "100%", padding: 8, marginBottom: 12, borderRadius: 6, border: "1px solid #e2e8f0" }}
                value={assignForm.userOid}
                onChange={(e) => setAssignForm((f) => ({ ...f, userOid: e.target.value }))}
              >
                <option value="">— Select user —</option>
                {(users ?? []).map((u: AdminUser) => (
                  <option key={u.oid} value={u.oid}>
                    {u.displayName} ({u.email})
                  </option>
                ))}
              </select>

              <label style={{ display: "block", marginBottom: 4 }}>Valid From</label>
              <input
                type="datetime-local"
                required
                style={{ width: "100%", padding: 8, marginBottom: 12, borderRadius: 6, border: "1px solid #e2e8f0" }}
                value={assignForm.validFrom}
                onChange={(e) => setAssignForm((f) => ({ ...f, validFrom: e.target.value }))}
              />

              <label style={{ display: "block", marginBottom: 4 }}>Valid Until</label>
              <input
                type="datetime-local"
                required
                style={{ width: "100%", padding: 8, marginBottom: 16, borderRadius: 6, border: "1px solid #e2e8f0" }}
                value={assignForm.validUntil}
                onChange={(e) => setAssignForm((f) => ({ ...f, validUntil: e.target.value }))}
              />

              {assignError && <p style={{ color: "red" }}>{assignError}</p>}

              <div style={{ display: "flex", gap: 8, justifyContent: "flex-end" }}>
                <button type="button" style={style.btn} onClick={() => setAssignModal(null)}>
                  Cancel
                </button>
                <button
                  type="submit"
                  style={{ ...style.btn, background: "#3b82f6", color: "#fff", padding: "6px 16px" }}
                >
                  Save
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Add Device Modal */}
      {addDeviceModal && (
        <div style={style.modal} onClick={() => setAddDeviceModal(false)}>
          <div style={style.modalBox} onClick={(e) => e.stopPropagation()}>
            <h2 style={{ marginTop: 0 }}>Add Device</h2>
            <form onSubmit={handleAddDevice}>
              {(["name", "ipAddress", "location", "password"] as const).map((field) => (
                <div key={field} style={{ marginBottom: 12 }}>
                  <label style={{ display: "block", marginBottom: 4, textTransform: "capitalize" }}>
                    {field === "ipAddress" ? "IP Address" : field}
                  </label>
                  <input
                    type={field === "password" ? "password" : "text"}
                    required={field !== "location"}
                    style={{ width: "100%", padding: 8, borderRadius: 6, border: "1px solid #e2e8f0" }}
                    value={addDeviceForm[field]}
                    onChange={(e) => setAddDeviceForm((f) => ({ ...f, [field]: e.target.value }))}
                  />
                </div>
              ))}
              {addDeviceError && <p style={{ color: "red" }}>{addDeviceError}</p>}
              <div style={{ display: "flex", gap: 8, justifyContent: "flex-end" }}>
                <button type="button" style={style.btn} onClick={() => setAddDeviceModal(false)}>
                  Cancel
                </button>
                <button
                  type="submit"
                  style={{ ...style.btn, background: "#3b82f6", color: "#fff", padding: "6px 16px" }}
                >
                  Add
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
