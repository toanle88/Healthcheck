import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App.tsx'
import './index.css'
import { EventType, type AuthenticationResult } from "@azure/msal-browser";
import { MsalProvider } from "@azure/msal-react";
import { msalInstance } from "./authConfig";

import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { ToastProvider } from './components/common/ToastProvider';

const queryClient = new QueryClient({
    defaultOptions: {
        queries: {
            refetchOnWindowFocus: false,
            retry: 1,
        },
    },
});

const isIframe = typeof globalThis.window !== 'undefined' && 
    globalThis.window !== globalThis.window.parent && 
    !globalThis.window.opener;

if (isIframe) {
    console.log("Iframe detected, skipping React/MSAL bootstrap.");
} else {
    try {
        // Initialize MSAL and handle redirect
        await msalInstance.initialize();
        const response = await msalInstance.handleRedirectPromise();
        
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
                        <ToastProvider>
                            <App />
                        </ToastProvider>
                    </QueryClientProvider>
                </MsalProvider>
            </React.StrictMode>,
        )
    } catch (err) {
        console.error("MSAL Redirect Error:", err);
    }
}
