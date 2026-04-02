import { Configuration, PopupRequest } from "@azure/msal-browser";

// These values are injected at build time via Vite's env mechanism.
// Set VITE_ENTRA_TENANT_ID and VITE_ENTRA_CLIENT_ID in your .env file.
const tenantId = import.meta.env.VITE_ENTRA_TENANT_ID as string;
const clientId = import.meta.env.VITE_ENTRA_CLIENT_ID as string;

export const msalConfig: Configuration = {
  auth: {
    clientId,
    authority: `https://login.microsoftonline.com/${tenantId}/v2.0`,
    redirectUri: window.location.origin,
    postLogoutRedirectUri: window.location.origin,
  },
  cache: {
    cacheLocation: "sessionStorage",
    storeAuthStateInCookie: false,
  },
};

/** Scopes requested for the management API access token. */
export const loginRequest: PopupRequest = {
  scopes: ["openid", "profile", "email"],
};

/** Base URL of the management backend API (same origin as the SPA). */
export const API_BASE = "";
