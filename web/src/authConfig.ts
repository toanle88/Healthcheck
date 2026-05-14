import type { Configuration, PopupRequest } from "@azure/msal-browser";
import { getEnv } from "./config/env";

const clientId = getEnv("VITE_ENTRA_CLIENT_ID");
const tenantId = getEnv("VITE_ENTRA_TENANT_ID");
const tenantDomain = "toanlesandbox.ciamlogin.com";

export const msalConfig: Configuration = {
  auth: {
    clientId,
    authority: `https://${tenantDomain}/${tenantId}`,
    knownAuthorities: [tenantDomain],
    redirectUri: window.location.origin,
    postLogoutRedirectUri: window.location.origin,
  },
  cache: {
    cacheLocation: "sessionStorage",
  },
  system: {
    allowRedirectInIframe: true,
  }
};

export const loginRequest: PopupRequest = {
  scopes: ["openid", "profile", "email", "offline_access"]
};

export const tokenRequest = {
  scopes: [`api://${clientId}/access_as_user`]
};