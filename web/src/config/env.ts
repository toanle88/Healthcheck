/**
 * Centralized utility for reading environment variables.
 * 
 * 1. In Production (Docker): It reads from `window.ENV` which is dynamically 
 *    injected at runtime via the entrypoint.sh script.
 * 2. In Local Dev (Vite): It falls back to `import.meta.env` which is 
 *    populated by Vite from the `.env.local` file.
 */
export const getEnv = (key: keyof Window['ENV']): string => {
  // Check if window.ENV exists and has the key (Production)
  if (window.ENV && window.ENV[key]) {
    return window.ENV[key] as string;
  }
  
  // Fallback to Vite's import.meta.env (Local Development)
  return import.meta.env[key] || "";
};
