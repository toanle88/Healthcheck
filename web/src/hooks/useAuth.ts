import { useMsal } from "@azure/msal-react";
import { useCallback, useMemo } from "react";
import { loginRequest, tokenRequest } from "../authConfig";

/**
 * Custom hook to manage MSAL authentication within the Healthcheck app.
 * Abstracts MSAL complexity and provides a clean interface for common auth operations.
 */
export const useAuth = () => {
  const { instance, accounts, inProgress } = useMsal();

  const user = useMemo(() => {
    if (accounts.length > 0) {
      return {
        name: accounts[0].name,
        username: accounts[0].username,
        account: accounts[0],
      };
    }
    return null;
  }, [accounts]);

  const login = useCallback(async () => {
    try {
      await instance.loginRedirect(loginRequest);
    } catch (error) {
      console.error("Login failed:", error);
    }
  }, [instance]);

  const logout = useCallback(() => {
    instance.logoutRedirect({
      postLogoutRedirectUri: window.location.origin,
    });
  }, [instance]);

  /**
   * Acquires a fresh access token for the CIAM API.
   * Attempts silent acquisition first, then falls back to interactive if needed.
   */
  const getAccessToken = useCallback(async () => {
    if (!user) throw new Error("No active account found. Please log in.");

    try {
      const response = await instance.acquireTokenSilent({
        ...tokenRequest,
        account: user.account,
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
  }, [instance, user]);

  return {
    user,
    isAuthenticated: accounts.length > 0,
    isProcessing: inProgress !== "none",
    login,
    logout,
    getAccessToken,
  };
};
