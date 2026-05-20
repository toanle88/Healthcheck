import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App.tsx'
import './index.css'
import { EventType, type AuthenticationResult } from "@azure/msal-browser";
import { MsalProvider } from "@azure/msal-react";
import { msalInstance } from "./authConfig";

import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

const queryClient = new QueryClient({
    defaultOptions: {
        queries: {
            refetchOnWindowFocus: false,
            retry: 1,
        },
    },
});

// Initialize MSAL and then render the app
msalInstance.initialize().then(() => {
    // ... same logic for redirect response ...
    msalInstance.handleRedirectPromise().then((response) => {
        if (response) {
            msalInstance.setActiveAccount(response.account);
        }

        const accounts = msalInstance.getAllAccounts();
        if (accounts.length > 0 && !msalInstance.getActiveAccount()) {
            msalInstance.setActiveAccount(accounts[0]);
        }

        msalInstance.addEventCallback((event) => {
            if (event.eventType === EventType.LOGIN_SUCCESS && event.payload) {
                const payload = event.payload as AuthenticationResult;
                msalInstance.setActiveAccount(payload.account);
            }
        });

        ReactDOM.createRoot(document.getElementById('root')!).render(
            <React.StrictMode>
                <MsalProvider instance={msalInstance}>
                    <QueryClientProvider client={queryClient}>
                        <App />
                    </QueryClientProvider>
                </MsalProvider>
            </React.StrictMode>,
        )
    }).catch((err) => {
        console.error("MSAL Redirect Error:", err);
    });
});
