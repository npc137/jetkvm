# Management Backend

Standalone Go service providing the management API for the JetKVM management plane.

## Build

```bash
go build -o management-backend .
```

## Run

```bash
export ENTRA_TENANT_ID=<your-tenant-id>
export ENTRA_CLIENT_ID=<your-client-id>
export PUBLIC_BASE_URL=https://kvm.company.com
./management-backend
```

## Docker

```bash
docker build -t jetkvm-management-backend .
docker run -p 8080:8080 \
  -e ENTRA_TENANT_ID=... \
  -e ENTRA_CLIENT_ID=... \
  -e PUBLIC_BASE_URL=https://kvm.company.com \
  -v management-data:/data \
  jetkvm-management-backend
```

## API reference

See the inline handler comments in `handlers/` or the main PR description for the full endpoint table.
