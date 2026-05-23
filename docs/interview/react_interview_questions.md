# React & Frontend Engineering Technical Interview Prep Guide

This guide contains 20 core React and frontend architecture interview questions, covering React 19, state management, MSAL authentication, TanStack Query, build tools (Vite), and real-time streaming (SSE). Many questions connect directly to the architecture of the **Healthcheck Dashboard** frontend.

---

## ⚛️ React & Frontend Technical Interview Q&As

### Q1: What is the Virtual DOM, and how does React's Reconciliation process work? Why is the `key` prop so important?
*   **Answer:**
    > "The **Virtual DOM (VDOM)** is an in-memory representation of the real HTML DOM. Real DOM updates are slow because they trigger browser repaints and layout recalculations. 
    > 
    > **Reconciliation (via React Fiber):**
    > When a component's state changes, React creates a new Virtual DOM tree. It then compares this new tree with the previous one using a diffing algorithm (a process called reconciliation) to find the minimum number of changes required. Finally, it applies only those specific changes to the real DOM (a process called 'patching' or 'commit phase').
    > 
    > **Importance of the `key` prop:**
    > When rendering lists of elements, the `key` prop serves as a unique identifier for each item. It helps React identify which items have changed, been added, or been removed. Without unique keys (or using array indices as keys), React might re-render or mismatch DOM states of elements, leading to slow rendering, visual bugs, or lost state in input fields."

### Q2: What is new in React 19? Explain how "Actions" simplify asynchronous state management.
*   **Answer:**
    > "React 19 introduces native support for handling async operations, transitions, and form submissions, collectively called **Actions**.
    > 
    > **How Actions help:**
    > Previously, during async operations (like submitting a form or checking health targets), we had to manually manage state for pending states, error states, and optimistic UI updates (using multiple `useState` hooks).
    > 
    > In React 19, if you pass an async function to a transition or form action, React automatically handles the pending state, error boundaries, and rollbacks:
    > *   **`useActionState`:** A hook that wraps async actions, returning the action's current state and a `isPending` boolean to easily disable buttons or show loaders.
    > *   **`useFormStatus`:** Allows nested child components to access parent form submission status (like `pending`) without prop drilling.
    > *   **`useOptimistic`:** Simplifies displaying optimistic states (showing the change immediately before the server confirms it)."

### Q3: How does the `useEffect` hook work, and why is the "Cleanup Function" critical when listening to Server-Sent Events (SSE)?
*   **Answer:**
    > "`useEffect` lets you synchronize a component with an external system (side effects). It runs after the browser paints the screen. It takes a dependency array:
    > *   If empty (`[]`), it runs once when the component mounts.
    > *   If populated (`[state]`), it runs on mount and whenever the dependencies change.
    > 
    > **The Cleanup Function:**
    > If `useEffect` returns a function, React runs this cleanup function before the component unmounts or before running the effect again.
    > 
    > In our Healthcheck dashboard, we connect to a **Server-Sent Events (SSE)** stream on the Go backend using `EventSource`. If the user navigates away from the dashboard and the component unmounts, the browser connection to the SSE stream will stay open if we don't close it, causing a resource leak and unnecessary server load.
    > 
    > We prevent this by returning a cleanup function that closes the connection:
    > ```typescript
    > useEffect(() => {
    >     const eventSource = new EventSource("/api/status/stream");
    >     eventSource.onmessage = (event) => { setData(JSON.parse(event.data)); };
    >     
    >     return () => {
    >         eventSource.close(); // Clean up!
    >     };
    > }, []);
    > ```"

### Q4: Why do we write Custom Hooks in React? Give an example of one you would use in a dashboard application.
*   **Answer:**
    > "Custom Hooks are JavaScript functions that start with `use` and can call other hooks. We use them to **separate business logic from UI rendering**, promote code reusability, and keep components dry.
    > 
    > **Example in our Dashboard:**
    > Instead of fetching targets directly inside our dashboard component, we write a custom hook called `useTargets`:
    > ```typescript
    > export function useTargets() {
    >     const { data, isLoading, error } = useQuery({
    >         queryKey: ['targets'],
    >         queryFn: fetchTargets,
    >     });
    >     
    >     const healthyCount = data?.filter(t => t.status === "healthy").length || 0;
    >     
    >     return { targets: data, isLoading, error, healthyCount };
    > }
    > ```
    > This keeps our visual dashboard component focused entirely on layout and JSX. Any other component (like a header widget showing target counts) can reuse `useTargets` without replicating the fetch logic."

### Q5: When should you use React Context API, and what are its performance drawbacks? How do you mitigate them?
*   **Answer:**
    > "The **Context API** is designed to share global state that doesn't change frequently (e.g. current user authentication, theme settings, language) across the component tree without manually passing props down (prop drilling).
    > 
    > **Performance Drawbacks:**
    > When a Context Provider's value changes, **all consumers of that Context re-render**, even if they only read a part of the state that did not change. This can cause severe rendering bottlenecks in complex components.
    > 
    > **Mitigations:**
    > 1.  **Split Contexts:** Instead of one large `AppContext`, create separate, focused contexts (e.g. `ThemeContext` and `AuthContext`).
    > 2.  **Memoization:** Memoize the context value object using `useMemo` so that the object reference only changes when its dependencies change.
    > 3.  **Use Dedicated State Managers:** For high-frequency state updates (like real-time metrics or canvas coordinates), use dedicated libraries like **Zustand** or **Redux**, which use selector-based subscriptions to prevent unnecessary re-renders."

### Q6: Why use TanStack Query (React Query) instead of fetching data inside `useEffect` with `axios`?
*   **Answer:**
    > "Using `useEffect` for data fetching leads to boilerplate code and bugs. TanStack Query provides enterprise-grade data synchronization out of the box:
    > 
    > 1.  **Automatic Caching:** It stores query data in memory. If a component mounts, unmounts, and remounts, it immediately displays the cached data first, updating it in the background (Stale-While-Revalidate).
    > 2.  **Deduplication:** If three different components on the screen trigger the same fetch request at the same time, TanStack Query deduplicates them into a single network call.
    > 3.  **Automatic Retries:** If a request fails due to network instability, it automatically retries with exponential backoff.
    > 4.  **State Management:** It provides built-in `isLoading`, `isError`, and `isFetching` states, removing the need for manual boolean hooks."

### Q7: Explain the difference between `React.memo`, `useMemo`, and `useCallback`. When should you use (and NOT use) them?
*   **Answer:**
    > "*   **`React.memo`:** A Higher-Order Component that wraps a component to prevent it from re-rendering if its props have not changed (performs a shallow comparison of props).
    > *   **`useMemo`:** A hook that memoizes (caches) the *result* of a costly calculation so it doesn't recalculate on every render.
    > *   **`useCallback`:** A hook that memoizes the *function definition itself* so its reference remains stable across renders.
    > 
    > **When NOT to use them:**
    > Memoization is not free; it consumes memory and CPU for dependency checks.
    > *   Do **not** wrap every simple function in `useCallback`. If a function doesn't pass down as a prop to a memoized child component (wrapped in `React.memo`), the overhead of recreating the function is negligible compared to the overhead of running `useCallback` dependency checks on every render."

### Q8: What is Code Splitting and Lazy Loading? How do you implement them in a React Vite application?
*   **Answer:**
    > "By default, build tools bundle all application code into a single large JavaScript file. This increases initial loading times. **Code splitting** breaks the bundle into smaller chunks that are loaded on demand.
    > 
    > **Implementation:**
    > We use `React.lazy` to load components dynamically, and wrap them in a `Suspense` block to show a fallback loader:
    > ```typescript
    > import { lazy, Suspense } from 'react';
    > 
    > const SettingsPage = lazy(() => import('./pages/SettingsPage'));
    > 
    > function App() {
    >     return (
    >         <Suspense fallback={<div>Loading Page...</div>}>
    >             <Routes>
    >                 <Route path="/settings" element={<SettingsPage />} />
    >             </Routes>
    >         </Suspense>
    >     );
    > }
    > ```
    > Vite will automatically detect this import statement and generate a separate `.js` file for `SettingsPage` during the build process, loading it only when the user navigates to `/settings`."

### Q9: How does the frontend handle authentication using MSAL (Microsoft Authentication Library) and Entra ID CIAM?
*   **Answer:**
    > "Our React app is wrapped in an `MsalProvider` from `@azure/msal-react`, initialized with a configuration object containing our Entra Client ID and Tenant ID.
    > 
    > 1.  **Auth Flow:** The user clicks log in, triggering `msalInstance.loginRedirect()` or `loginPopup()`.
    > 2.  **Token Storage:** After successful authentication, MSAL securely stores the ID Token and Access Token in the browser's cache (Local Storage or Session Storage).
    > 3.  **Route Protection:** We wrap routes in `MsalAuthenticationTemplate` or check `useIsAuthenticated()` to block unauthenticated users.
    > 4.  **API Requests:** To communicate with the secure Go API, we call `msalInstance.acquireTokenSilent()` to fetch the access token. If the token is expired, MSAL automatically uses a refresh token to fetch a new one silently, which we attach to the Axios authorization header as a Bearer token."

### Q10: What is the difference between state (`useState`) and a ref (`useRef`)? When should you use `useRef`?
*   **Answer:**
    > "*   **`useState`:** Triggers a component re-render when the state value changes. Used to track data that directly impacts the visual output (JSX) of the component.
    > *   **`useRef`:** A mutable object whose `.current` property persists across renders. **Changing its value does not trigger a re-render**.
    > 
    > **When to use `useRef`:**
    > 1.  **Direct DOM Access:** Referencing input focus, triggering video play/pause, or measuring DOM node dimensions.
    > 2.  **Storing Instance Variables:** Storing timer IDs (for `setInterval`), previous state values, or flag variables (like `isMounted`) that we want to inspect without causing unnecessary UI updates."

### Q11: What is React Strict Mode? Why does it cause hooks to run twice in development?
*   **Answer:**
    > "**Strict Mode** is a development-only tool that checks for potential bugs and unsafe lifecycle methods in your components. It does not affect production builds.
    > 
    > **Running Hooks Twice:**
    > In development, Strict Mode intentionally **mounts, unmounts, and remounts** every component (running setup and cleanup functions twice).
    > 
    > It does this to:
    > *   Verify that your components are pure (they return the same output for the same props).
    > *   Ensure that your `useEffect` cleanups are correctly implemented and do not leak memory (such as forgetting to close an SSE stream or clear a timeout)."

### Q12: How does the new React 19 `use` hook work, and how does it differ from traditional hooks?
*   **Answer:**
    > "The new `use` hook in React 19 is unique because, unlike standard hooks (`useState`, `useEffect`), it **can be called conditionally and inside loops or `if` statements**.
    > 
    > **Capabilities:**
    > *   **Reading Promises:** You can pass a Promise (like a fetch request) to `use(promise)`. React will suspend the component (triggering the nearest `Suspense` boundary) until the promise resolves.
    > *   **Reading Context:** You can write `const theme = use(ThemeContext)`. Calling it conditionally inside an `if` block allows you to subscribe to context only when specific conditions are met, saving memory."

### Q13: What is an Error Boundary, and how do you implement one?
*   **Answer:**
    > "An **Error Boundary** is a class component that catches JavaScript errors anywhere in its child component tree, logs those errors, and displays a fallback UI instead of letting the entire application crash to a blank white screen.
    > 
    > **Implementation:**
    > To implement an Error Boundary, a class component must define the lifecycle methods `static getDerivedStateFromError()` (to update state and trigger fallback UI) and `componentDidCatch()` (to log error details to services like Sentry).
    > 
    > *Note: Error boundaries do not catch errors inside async callbacks, event handlers, or during server-side rendering.*"

### Q14: What is the difference between Controlled and Uncontrolled Components in React forms?
*   **Answer:**
    > "*   **Controlled Components:** The form data is handled by the React state. The input value is bound to a state variable (e.g. `value={name}`), and changes are updated via `onChange` handlers. React is the 'single source of truth'.
    >     *   *Pros:* Easy to validate inputs on the fly, disable submit buttons, or format inputs dynamically.
    > *   **Uncontrolled Components:** The form data is handled by the DOM itself. We use a `useRef` to pull the value from the input when needed (e.g. on submit).
    >     *   *Pros:* Faster to write for simple forms, can offer better performance since it avoids re-rendering the component on every keystroke."

### Q15: How does your React app handle expired JWT tokens silently using Axios Interceptors?
*   **Answer:**
    > "We configure an Axios **response interceptor**. 
    > 
    > When our app makes an API request and the server returns a `401 Unauthorized` status (indicating our access token has expired):
    > 1.  The interceptor catches the error.
    > 2.  It halts the request and calls MSAL's `acquireTokenSilent()` method.
    > 3.  MSAL contacts Entra ID using the cached refresh token to acquire a fresh access token.
    > 4.  The interceptor updates the header with the new token and retries the original failed request.
    > 
    > This entire cycle occurs seamlessly in the background without forcing the user to log out or refresh the page."

### Q16: How do you handle TypeScript types for React props, event handlers, and custom hooks?
*   **Answer:**
    > "*   **Props Typing:** We use interfaces to define prop contracts:
    >     ```typescript
    >     interface TargetCardProps {
    >         target: Target;
    >         onRefresh: (id: string) => void;
    >     }
    >     ```
    > *   **Event Handlers:** We use React's built-in types:
    >     *   Click events: `React.MouseEvent<HTMLButtonElement>`
    >     *   Input changes: `React.ChangeEvent<HTMLInputElement>`
    > *   **Hooks Typing:** TypeScript inferring works for most hooks, but we explicitly type `useState<Target | null>(null)` or define generic returns for custom hooks to maintain strict compiler safety."

### Q17: What are the benefits of using Tailwind CSS in React? How do you handle dynamic classes cleanly?
*   **Answer:**
    > "**Benefits:** Tailwind prevents style bloat by using utility classes, keeps styles colocated within JSX files (no shifting between `.css` and `.tsx` files), and provides a consistent design system (colors, padding, spacing).
    > 
    > **Handling Dynamic Classes:**
    > If we dynamically construct class strings, we can run into issues where different utilities conflict. We solve this using **`clsx`** and **`tailwind-merge`** (often wrapped in a helper function called `cn`):
    > ```typescript
    > import { clsx, type ClassValue } from 'clsx';
    > import { twMerge } from 'tailwind-merge';
    > 
    > export function cn(...inputs: ClassValue[]) {
    >     return twMerge(clsx(inputs));
    > }
    > ```
    > This lets us conditionally apply classes safely (e.g. `cn("bg-green-500", isError && "bg-red-500")`) and ensures the final class string has no duplicates or overrides."

### Q18: Why use a library like React Hook Form instead of native controlled states for complex forms?
*   **Answer:**
    > "In complex forms, using standard controlled components (`useState` on every input) causes the entire form component to re-render on **every single keystroke**. In forms with dozens of fields, this causes typing lag and performance issues.
    > 
    > **React Hook Form** uses **uncontrolled components under the hood** (registering inputs via refs). 
    > *   It only re-renders the specific input fields when their validation state changes.
    > *   It dramatically reduces code boilerplate for validation rules, error message rendering, and submission states."

### Q19: Why is Vite preferred over Create React App (CRA) for modern React development?
*   **Answer:**
    > "Create React App uses Webpack under the hood, which builds the entire application bundle before starting the development server. As the app grows, dev startup and Hot Module Replacement (HMR) get extremely slow.
    > 
    > **Vite** is much faster because:
    > 1.  **Native ESM:** It serves source code over native ES Modules. The browser requests individual files dynamically, meaning dev server startup is nearly instant.
    > 2.  **Esbuild Transpilation:** It compiles files using **Esbuild** (written in Go), which is $10\times$ to $100\times$ faster than Webpack's Babel compilation.
    > 3.  **Rollup for Production:** It uses Rollup for highly optimized production bundling with efficient tree-shaking."

### Q20: How does the frontend handle real-time streaming updates from the Go API? How do you manage EventSource reconnects?
*   **Answer:**
    > "We connect using the browser's native `EventSource` API pointing to the `/api/status/stream` SSE endpoint.
    > 
    > **Handling updates:**
    > When the Go worker pings targets, it pushes status payloads. The `EventSource.onmessage` handler parses the JSON data and updates our state.
    > 
    > **Handling Reconnects:**
    > The browser's `EventSource` automatically handles reconnection out of the box. If the Go backend restarts or the network drops, the browser waits a few seconds and retries the connection automatically. 
    > However, if we receive specific HTTP errors (like `401 Unauthorized`), we must manually close the `EventSource` instance in our cleanup effect, refresh the token, and instantiate a new `EventSource` connection with the new authorization headers passed in the query string or cookie."
