import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App.tsx'
import './index.css'
import { PublicClientApplication, EventType, type AuthenticationResult } from "@azure/msal-browser";
import { MsalProvider } from "@azure/msal-react";
import { msalConfig } from "./authConfig";

const msalInstance = new PublicClientApplication(msalConfig);

// Initialize MSAL and then render the app
msalInstance.initialize().then(() => {
    // Check for redirect response on page load
    msalInstance.handleRedirectPromise().then((response) => {
        if (response) {
            msalInstance.setActiveAccount(response.account);
        }

        // Default to using the first account if no account is active on page load
        const accounts = msalInstance.getAllAccounts();
        if (accounts.length > 0 && !msalInstance.getActiveAccount()) {
            msalInstance.setActiveAccount(accounts[0]);
        }

        // Listen for sign-in event and set active account
        msalInstance.addEventCallback((event) => {
            if (event.eventType === EventType.LOGIN_SUCCESS && event.payload) {
                const payload = event.payload as AuthenticationResult;
                msalInstance.setActiveAccount(payload.account);
            }
        });

        ReactDOM.createRoot(document.getElementById('root')!).render(
            <React.StrictMode>
                <MsalProvider instance={msalInstance}>
                    <App />
                </MsalProvider>
            </React.StrictMode>,
        )
    }).catch((err) => {
        console.error("MSAL Redirect Error:", err);
    });
});
