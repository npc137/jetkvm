# JetKVM Management Plane — Deployment Guide

## Prerequisites

- Docker Engine ≥ 24 and Docker Compose v2
- A DNS record pointing `kvm.company.com` (or your chosen hostname) to this server
- Ports 80 and 443 open on the host firewall
- A Microsoft Entra ID app registration (see §1 below)

---

## 1 · Entra ID App Registration

1. In the [Azure Portal](https://portal.azure.com) go to **App registrations → New registration**.
2. Name: `JetKVM Management Plane`
3. Supported account types: *Accounts in this organizational directory only*
4. Redirect URI: `https://kvm.company.com/` (Single-page application)
5. After creation, note the **Application (client) ID** and **Directory (tenant) ID**.
6. Under **Authentication**, enable *Access tokens* and *ID tokens*.
7. Under **API permissions**, add `openid`, `profile`, `email` (Microsoft Graph delegated).

---

## 2 · Configure Environment

```bash
cd management/deploy
cp .env.example .env
# Edit .env with your ENTRA_TENANT_ID, ENTRA_CLIENT_ID, MANAGEMENT_HOST, etc.
```

---

## 3 · Start the Stack

```bash
docker compose up -d
```

This starts:
| Container | Role |
|-----------|------|
| `management-backend` | Go API server (port 8080, internal only) |
| `management-ui` | React SPA (port 3000, internal only) |
| `traefik` | Reverse proxy (ports 80 + 443, public) |

---

## 4 · Promote the First Admin

After the first user signs in via the dashboard, promote them to admin:

```bash
# Find their Entra OID in the database:
docker exec -it deploy-management-backend-1 \
  sqlite3 /data/management.db "SELECT entra_oid, email FROM users;"

# Promote:
docker exec -it deploy-management-backend-1 \
  sqlite3 /data/management.db \
  "UPDATE users SET role='admin' WHERE entra_oid='<oid>';"
```

---

## 5 · Register a JetKVM Device

Use the admin API (or the dashboard once #4 is done):

```bash
curl -X POST https://kvm.company.com/api/admin/devices \
  -H "Authorization: Bearer <admin-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Server Room KVM 1",
    "ipAddress": "192.168.1.10",
    "location": "DC1 Rack A3",
    "password": "device-local-password"
  }'
```

> ⚠️ The device password is stored in the SQLite database. Ensure the host
> running this stack uses full-disk encryption.

---

## 6 · Network Security

Ensure JetKVM devices are on a **VLAN or subnet that is NOT reachable from the
internet or from user workstations directly**. Only the Docker host running
Traefik should be able to reach the device IPs.

The management plane enforces authorization before any traffic reaches a device,
but network-level segmentation is the defence-in-depth layer.

---

## 7 · Upgrading

```bash
docker compose pull
docker compose up -d --build
```
