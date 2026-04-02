import { AccountInfo, IPublicClientApplication } from "@azure/msal-browser";
import { loginRequest, API_BASE } from "../authConfig";
import type {
  User,
  Device,
  Assignment,
  CreateAssignmentRequest,
  AdminUser,
} from "./types";

/** Acquire a fresh access token silently (falls back to popup if needed). */
async function getToken(
  instance: IPublicClientApplication,
  account: AccountInfo
): Promise<string> {
  try {
    const result = await instance.acquireTokenSilent({
      ...loginRequest,
      account,
    });
    return result.idToken;
  } catch {
    const result = await instance.acquireTokenPopup(loginRequest);
    return result.idToken;
  }
}

async function apiFetch<T>(
  path: string,
  instance: IPublicClientApplication,
  account: AccountInfo,
  init?: RequestInit
): Promise<T> {
  const token = await getToken(instance, account);
  const res = await fetch(`${API_BASE}${path}`, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${token}`,
      ...(init?.headers ?? {}),
    },
  });
  if (!res.ok) {
    const body = await res.text();
    throw new Error(`${res.status} ${res.statusText}: ${body}`);
  }
  if (res.status === 204) return undefined as T;
  return res.json() as T;
}

// ── User ─────────────────────────────────────────────────────────────────────

export const getMe = (
  instance: IPublicClientApplication,
  account: AccountInfo
) => apiFetch<User>("/api/me", instance, account);

// ── Devices ───────────────────────────────────────────────────────────────────

export const getDevices = (
  instance: IPublicClientApplication,
  account: AccountInfo
) => apiFetch<Device[]>("/api/devices", instance, account);

export const getDeviceStatus = (
  instance: IPublicClientApplication,
  account: AccountInfo,
  id: string
) => apiFetch<{ status: string; lastSeenAt?: string }>(`/api/devices/${id}/status`, instance, account);

// ── Connect ───────────────────────────────────────────────────────────────────

export const requestConnect = (
  instance: IPublicClientApplication,
  account: AccountInfo,
  deviceId: string
) =>
  apiFetch<{ connectUrl: string }>(`/api/connect/${deviceId}`, instance, account, {
    method: "POST",
  });

// ── Assignments ───────────────────────────────────────────────────────────────

export const getAssignments = (
  instance: IPublicClientApplication,
  account: AccountInfo
) => apiFetch<Assignment[]>("/api/assignments", instance, account);

export const createAssignment = (
  instance: IPublicClientApplication,
  account: AccountInfo,
  req: CreateAssignmentRequest
) =>
  apiFetch<Assignment>("/api/assignments", instance, account, {
    method: "POST",
    body: JSON.stringify(req),
  });

export const deleteAssignment = (
  instance: IPublicClientApplication,
  account: AccountInfo,
  id: string
) =>
  apiFetch<void>(`/api/assignments/${id}`, instance, account, {
    method: "DELETE",
  });

// ── Admin ─────────────────────────────────────────────────────────────────────

export const getAdminUsers = (
  instance: IPublicClientApplication,
  account: AccountInfo
) => apiFetch<AdminUser[]>("/api/admin/users", instance, account);

export const setUserRole = (
  instance: IPublicClientApplication,
  account: AccountInfo,
  oid: string,
  role: "admin" | "user"
) =>
  apiFetch<void>(`/api/admin/users/${oid}/role`, instance, account, {
    method: "PUT",
    body: JSON.stringify({ role }),
  });

export const createDevice = (
  instance: IPublicClientApplication,
  account: AccountInfo,
  data: { name: string; ipAddress: string; location?: string; password: string }
) =>
  apiFetch<{ id: string }>("/api/admin/devices", instance, account, {
    method: "POST",
    body: JSON.stringify(data),
  });

export const deleteDevice = (
  instance: IPublicClientApplication,
  account: AccountInfo,
  id: string
) =>
  apiFetch<void>(`/api/admin/devices/${id}`, instance, account, {
    method: "DELETE",
  });
