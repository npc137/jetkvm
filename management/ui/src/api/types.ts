/** Types that mirror the management backend API models. */

export interface User {
  oid: string;
  email: string;
  displayName: string;
  role: "admin" | "user";
  lastLoginAt?: string;
}

export interface Device {
  id: string;
  name: string;
  location?: string;
  status: "online" | "offline" | "unknown";
  lastSeenAt?: string;
  fwVersion?: string;
  assignedTo?: string;
}

export interface Assignment {
  id: string;
  userOid: string;
  deviceId: string;
  validFrom: string;
  validUntil: string;
  createdAt: string;
}

export interface CreateAssignmentRequest {
  userOid: string;
  deviceId: string;
  validFrom: string;
  validUntil: string;
}

export interface AdminUser {
  oid: string;
  email: string;
  displayName: string;
  role: "admin" | "user";
  lastLoginAt?: string;
}
