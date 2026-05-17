import { useMsal } from "@azure/msal-react";
import { useCallback, useMemo } from "react";
import { loginRequest, tokenRequest } from "../authConfig";

/**
 * Custom hook to manage MSAL authentication within the Healthcheck app.
 * Abstracts MSAL complexity and provides a clean interface for common auth operations.
 */
export const useAuth = () => {
  const { instance, accounts, inProgress } = useMsal();

  const isE2E = typeof window !== 'undefined' && (
    window.location.search.includes("test-mode=true") || 
    localStorage.getItem("playwright-mock-auth") === "true" ||
    (window as unknown as { playwrightMockAuth?: boolean | string }).playwrightMockAuth === true ||
    (window as unknown as { playwrightMockAuth?: boolean | string }).playwrightMockAuth === "true"
  );

  const realUser = useMemo(() => {
    if (accounts.length > 0) {
      return {
        name: accounts[0].name,
        username: accounts[0].username,
        account: accounts[0],
      };
    }
    return null;
  }, [accounts]);

  const user = useMemo(() => {
    if (isE2E) {
      return {
        name: "E2E Test User",
        username: "e2e@example.com",
        account: {} as unknown as typeof accounts[0]
      };
    }
    return realUser;
  }, [isE2E, realUser]);

  const realLogin = useCallback(async () => {
    try {
      await instance.loginRedirect(loginRequest);
    } catch (error) {
      console.error("Login failed:", error);
    }
  }, [instance]);

  const login = useCallback(async () => {
    if (isE2E) return;
    return realLogin();
  }, [isE2E, realLogin]);

  const realLogout = useCallback(() => {
    instance.logoutRedirect({
      postLogoutRedirectUri: window.location.origin,
    });
  }, [instance]);

  const logout = useCallback(() => {
    if (isE2E) return;
    realLogout();
  }, [isE2E, realLogout]);

  /**
   * Acquires a fresh access token for the CIAM API.
   * Attempts silent acquisition first, then falls back to interactive if needed.
   */
  const realGetAccessToken = useCallback(async () => {
    if (!realUser) throw new Error("No active account found. Please log in.");

    try {
      const response = await instance.acquireTokenSilent({
        ...tokenRequest,
        account: realUser.account,
      });
      return response.accessToken;
    } catch (silentError) {
      console.warn("Silent token acquisition failed, attempting popup:", silentError);
      try {
        const response = await instance.acquireTokenPopup(tokenRequest);
        return response.accessToken;
      } catch (popupError) {
        console.error("Interactive token acquisition failed:", popupError);
        throw popupError;
      }
    }
  }, [instance, realUser]);

  const getAccessToken = useCallback(async () => {
    if (isE2E) {
      return "mocked-e2e-token";
    }
    return realGetAccessToken();
  }, [isE2E, realGetAccessToken]);

  return {
    user,
    isAuthenticated: isE2E ? true : accounts.length > 0,
    isProcessing: isE2E ? false : inProgress !== "none",
    login,
    logout,
    getAccessToken,
  };
};
