import React from 'react';
import { render } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MsalProvider } from "@azure/msal-react";
import { PublicClientApplication } from "@azure/msal-browser";

const createTestQueryClient = () => new QueryClient({
  defaultOptions: {
    queries: {
      retry: false,
    },
  },
});

export function renderWithProviders(ui: React.ReactElement) {
  const queryClient = createTestQueryClient();
  const msalInstance = new PublicClientApplication({
    auth: {
      clientId: 'test-client-id',
      authority: 'https://login.microsoftonline.com/common',
    }
  });

  return render(
    <MsalProvider instance={msalInstance}>
      <QueryClientProvider client={queryClient}>
        {ui}
      </QueryClientProvider>
    </MsalProvider>
  );
}

export function createWrapper() {
  const queryClient = createTestQueryClient();
  const msalInstance = new PublicClientApplication({
    auth: {
      clientId: 'test-client-id',
      authority: 'https://login.microsoftonline.com/common',
    }
  });

  return ({ children }: { children: React.ReactNode }) => (
    <MsalProvider instance={msalInstance}>
      <QueryClientProvider client={queryClient}>
        {children}
      </QueryClientProvider>
    </MsalProvider>
  );
}
