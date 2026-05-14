import type { Configuration, PopupRequest } from "@azure/msal-browser";

// These values are injected by Terraform into the container environment
// For local development, you can set these in your .env file
const clientId = import.meta.env.VITE_ENTRA_CLIENT_ID || "";
const tenantId = import.meta.env.VITE_ENTRA_TENANT_ID || "";

export const msalConfig: Configuration = {
    auth: {
        clientId: clientId,
        authority: `https://login.microsoftonline.com/${tenantId}`,
        redirectUri: window.location.origin, // Returns to the same page
        postLogoutRedirectUri: window.location.origin,
    },
    cache: {
        cacheLocation: "sessionStorage", // Switched back to sessionStorage for stable redirects
    },
    system: {
        allowRedirectInIframe: true,
    }
};

// Add here scopes for id token to be used at MS Identity Platform endpoints.
export const loginRequest: PopupRequest = {
    scopes: ["User.Read", `api://${clientId}/access_as_user`]
};

// Add here scopes for access token to be used at MS Graph API endpoints.
export const tokenRequest = {
    scopes: [`api://${clientId}/access_as_user`]
};
