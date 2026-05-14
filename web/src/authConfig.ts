import type { Configuration, PopupRequest } from "@azure/msal-browser";

const clientId = import.meta.env.VITE_ENTRA_CLIENT_ID || "";
const tenantId = import.meta.env.VITE_ENTRA_TENANT_ID || "";
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
  scopes: ["api://a91de45f-2874-438b-aaa5-e0ae74985f40/access_as_user"]
};