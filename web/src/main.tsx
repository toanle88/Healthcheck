import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App.tsx'
import './index.css'
import { PublicClientApplication, EventType, type AuthenticationResult } from "@azure/msal-browser";
import { MsalProvider } from "@azure/msal-react";
import { msalConfig } from "./authConfig";

const msalInstance = new PublicClientApplication(msalConfig);

// Default to using the first account if no account is active on page load
if (!msalInstance.getActiveAccount() && msalInstance.getAllAccounts().length > 0) {
    // Account selection logic is app dependent
    msalInstance.setActiveAccount(msalInstance.getAllAccounts()[0]);
}

// Listen for sign-in event and set active account
msalInstance.addEventCallback((event) => {
    if (event.eventType === EventType.LOGIN_SUCCESS && event.payload) {
        const payload = event.payload as AuthenticationResult;
        const account = payload.account;
        msalInstance.setActiveAccount(account);
    }
});

// Initialize MSAL and then render the app
msalInstance.initialize().then(() => {
    ReactDOM.createRoot(document.getElementById('root')!).render(
        <React.StrictMode>
            <MsalProvider instance={msalInstance}>
                <App />
            </MsalProvider>
        </React.StrictMode>,
    )
});
