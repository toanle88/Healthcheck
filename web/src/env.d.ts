/// <reference types="vite/client" />

interface Window {
  ENV: {
    VITE_API_URL?: string;
    VITE_APP_VERSION?: string;
    VITE_ENTRA_CLIENT_ID?: string;
    VITE_ENTRA_TENANT_ID?: string;
    VITE_ENTRA_TENANT_DOMAIN?: string;
  };
}
