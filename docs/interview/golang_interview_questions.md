# Go (Golang) Technical Interview Prep Guide

This guide contains 20 core Go language interview questions, ranging from foundations to advanced concurrency, memory management, and runtime internals. Many questions are tied directly to patterns used in the **Healthcheck Dashboard** backend.

---

## 🐹 Go-Specific Technical Interview Q&As

### Q1: Why is the `context` package so important in Go? How does your background worker use Context to prevent "Goroutine Leaks"?
*   **Answer:**
    > "In Go, `context.Context` is the standard way to carry deadlines, cancellation signals, and request-scoped values across API boundaries and goroutines.
    > 
    > In our background worker, we ping external APIs. If an external API hangs or responds very slowly, it could block our worker's goroutine indefinitely. If the cron trigger fires again, it will spin up *another* goroutine. Over time, these blocked goroutines accumulate, leaking memory until the container crashes (a **goroutine leak**).
    > 
    > To prevent this, we create a context with a timeout:
    > ```go
    > ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    > defer cancel()
    > ```
    > We pass this `ctx` to the HTTP client request: `http.NewRequestWithContext(ctx, ...)` and database queries. If the target doesn't respond in 10 seconds, the context sends a cancellation signal, the HTTP request terminates immediately, resources are cleaned up, and the goroutine exits safely. The `defer cancel()` call ensures resources are freed as soon as the operation finishes, even if it succeeds before the timeout."

### Q2: How does concurrency work in Go? In your API's SSE (Server-Sent Events) broker, how do you handle broadcasting messages to multiple clients without causing Race Conditions?
*   **Answer:**
    > "Go concurrency is built on **Communicating Sequential Processes (CSP)** using Goroutines (lightweight threads managed by the Go runtime) and Channels (typed pipelines to send/receive data).
    > 
    > In our Go API, we use a **Server-Sent Events (SSE) Broker** to stream real-time target status updates to browser clients. Since multiple clients connect concurrently, we have shared state (the list of active client channels). If we mutate this list from multiple goroutines simultaneously, it will trigger a race condition panic.
    > 
    > We solve this using Go channels to synchronize access (or a `sync.RWMutex`). The broker maintains:
    > 1. A channel to register new clients.
    > 2. A channel to unregister clients.
    > 3. A channel to broadcast messages.
    > 
    > The broker runs a single-threaded control loop in a background goroutine:
    > ```go
    > select {
    > case client := <-b.register:
    >     b.clients[client] = true
    > case client := <-b.unregister:
    >     delete(b.clients, client)
    > case msg := <-b.broadcast:
    >     for client := range b.clients {
    >         client <- msg
    >     }
    > }
    > ```
    > Because the shared map `b.clients` is *only* modified and read inside this single goroutine's loop, we guarantee thread safety and prevent race conditions without needing complex mutex locking."

### Q3: Go doesn't have traditional try/catch exceptions. How does Go handle errors, and how do you implement "Error Wrapping"?
*   **Answer:**
    > "Go treats errors as values. Functions that can fail return an `error` interface as their last return value, and the caller is expected to check it explicitly using `if err != nil`.
    > 
    > To add context to errors as they propagate up the call stack, we use **error wrapping** introduced in Go 1.13. We use the `%w` verb in `fmt.Errorf`:
    > ```go
    > if err != nil {
    >     return fmt.Errorf("failed to fetch target from database: %w", err)
    > }
    > ```
    > This creates an error tree. When the handler receives this error, it can check if the underlying root cause is a specific error (e.g. database connection lost) using `errors.Is(err, ErrDbConnection)` or extract a specific error type using `errors.As(err, &myCustomError)` while maintaining the debugging log chain."

### Q4: Why are interfaces crucial in Go, and how did you use them in this project to make your API handlers unit-testable?
*   **Answer:**
    > "Interfaces in Go are satisfied implicitly (structural typing). A type doesn't need to declare that it implements an interface; it just needs to implement the required methods.
    > 
    > In this project, to unit test our HTTP handlers (like `GetStatusHandler`), we want to test the routing and JSON responses *without* connecting to a live PostgreSQL database.
    > 
    > I defined a `Store` interface:
    > ```go
    > type Store interface {
    >     GetTargets(ctx context.Context) ([]Target, error)
    >     SaveResult(ctx context.Context, res Result) error
    > }
    > ```
    > In production, the API handler receives a `PostgresStore` struct which calls the real database.
    > In our unit tests, we pass a `MockStore` struct that implements those same methods but returns hardcoded slices. This allows us to run isolated unit tests in milliseconds without mock servers or cloud database dependencies."

### Q5: When writing code in Go, how do you decide whether to pass a struct pointer (`*User`) or pass it by value (`User`)?
*   **Answer:**
    > "I decide based on three criteria: Mutability, Size, and Consistency.
    > 
    > 1.  **Mutability:** If the function needs to modify the state of the struct, I must pass a pointer. If the struct is passed by value, Go makes a copy, and modifications only affect the local copy.
    > 2.  **Size:** If the struct is large (contains many fields or large slices), copying it on every function call is expensive. Passing a pointer (8 bytes on 64-bit systems) is much faster. For tiny structs (e.g. containing 2-3 fields), passing by value is often faster because it keeps the variable on the **stack** rather than escaping to the **heap**, avoiding garbage collection overhead.
    > 3.  **Consistency:** In method receivers, if some methods of a struct require a pointer receiver, all methods should use pointer receivers for consistency to avoid confusion."

### Q6: What is structured logging, and why is `slog` preferred in cloud production environments over standard stdout printing?
*   **Answer:**
    > "Standard logging (like `fmt.Println` or standard `log` package) outputs unstructured plain text. In cloud environments where logs are aggregated from thousands of containers, plain text is very difficult to query or filter.
    > 
    > **Structured logging** outputs logs in a machine-readable format—usually **JSON**. In Go 1.21, Microsoft and Google collaborated to release the built-in `log/slog` package.
    > 
    > Instead of printing: `[ERROR] user 123 failed to log in due to timeout`
    > `slog` outputs:
    > `{"time":"2026-05-23T22:00:00Z","level":"ERROR","msg":"login failed","user_id":123,"reason":"timeout"}`
    > 
    > This allows tools like Azure Log Analytics, Elasticsearch, or Datadog to index the `user_id` and `reason` fields as distinct queryable columns. We can then instantly search, alert, or build dashboards on failed logins by specific user IDs without running slow regular expressions over plain text logs."

### Q7: Explain the difference between an Array and a Slice in Go. How does `append` work under the hood?
*   **Answer:**
    > "*   **Array:** Has a fixed size defined at compile-time (e.g., `[5]int`). Its size is part of its type, and it cannot be resized. Passing an array to a function copies the entire array.
    > *   **Slice:** A dynamically-sized view into an underlying array. Its type is defined as `[]int`. It is a header containing a pointer to the underlying array, a length (`len`), and a capacity (`cap`).
    > 
    > **How `append` works:**
    > When you call `append(slice, element)`, Go checks if `len + 1 <= cap` of the underlying array:
    > *   **If yes:** It places the new element in the next slot of the array and increments the slice's length.
    > *   **If no:** The underlying array is full. Go allocates a new, larger array (usually double the size for small slices, or $1.25\times$ for larger slices), copies the existing elements over, appends the new element, and returns a new slice pointing to the new array."

### Q8: Are Go Maps thread-safe? How do you inspect maps for race conditions, and how do you make them thread-safe?
*   **Answer:**
    > "No, Go maps are **not thread-safe**. Concurrent reads and writes to the same map will cause a fatal runtime crash: `fatal error: concurrent map writes`.
    > 
    > *   **Detecting Races:** I run Go tests or compile binaries with the **race detector** enabled: `go test -race ./...` or `go run -race main.go`. This instruments the compiler to log warnings if multiple threads access memory without synchronization.
    > *   **Making Maps Safe:**
    >     1.  **Mutex:** Wrap the map in a struct alongside a `sync.Mutex` or `sync.RWMutex` to lock the map during reads/writes.
    >     2.  **sync.Map:** For specific high-concurrency cases where keys are mostly stable or disjoint, use the standard library's `sync.Map`.
    >     3.  **Channels:** Use channels to serialize map access inside a single coordinator goroutine (as we did in the SSE broker)."

### Q9: How does the `defer` keyword work in Go? What is the evaluation order of its arguments?
*   **Answer:**
    > "The `defer` statement pushes a function call onto a list. The list of saved calls is executed in **Last-In-First-Out (LIFO)** order (stacked) immediately after the surrounding function returns.
    > 
    > **Argument Evaluation:**
    > The arguments of a deferred function are **evaluated immediately** when the `defer` line is reached, not when the function actually executes.
    > ```go
    > i := 0
    > defer fmt.Println(i) // Prints '0' because i is evaluated now
    > i++
    > ```
    > *Note: If you defer a function closure (e.g., `defer func() { fmt.Println(i) }()`), the variable is evaluated when the closure runs, which would print `1`.*"

### Q10: Explain Go's GMP Scheduler model. How does it handle blocking calls?
*   **Answer:**
    > "Go uses an **M:N Scheduler** that multiplexes $N$ Goroutines onto $M$ Operating System threads using $P$ Logical Processors.
    > *   **G (Goroutine):** Represents the goroutine code, stack, and program counter.
    > *   **M (Machine):** Represents a physical OS thread managed by the kernel.
    > *   **P (Processor):** Represents logical resources/context required to execute Go code (defaults to the number of CPU cores).
    > 
    > **Handling Blocking Calls:**
    > *   **Syscalls / Disk I/O:** If a Goroutine blocks on a synchronous system call, the scheduler detaches the thread ($M$) executing it from the logical processor ($P$). The scheduler spins up or borrows a new OS thread ($M_2$) to keep executing other ready goroutines in the run queue of $P$.
    > *   **Network I/O / Channels:** If a Goroutine blocks on a network read or a channel, Go uses the **Netpoller** (which utilizes OS-level polling like `epoll` on Linux). The goroutine is parked in the Netpoller, freeing up the logical processor and thread to run other tasks. When the network data arrives, the Netpoller wakes the goroutine and places it back in a run queue."

### Q11: What is the difference between a Buffered and an Unbuffered Channel? What happens when you read/write to a closed or `nil` channel?
*   **Answer:**
    > "*   **Unbuffered Channel:** Created via `make(chan int)`. Writes block until a reader is ready to receive, and vice-versa. It acts as a synchronous handshake.
    > *   **Buffered Channel:** Created via `make(chan int, capacity)`. Writes do not block as long as the buffer has empty slots. Readers do not block as long as the buffer has elements. It acts as an asynchronous queue.
    > 
    > **Channel Edge Cases:**
    > 
    > | Operation | Unbuffered / Buffered | Closed Channel | Nil Channel |
    > | :--- | :--- | :--- | :--- |
    > | **Write** | Blocks if no receiver/buffer full | **Panics** | **Blocks forever** |
    > | **Read** | Blocks if no sender/buffer empty | Returns zero value (non-blocking) | **Blocks forever** |
    > | **Close** | Closes channel | **Panics** | **Panics** |"

### Q12: How do `panic` and `recover` work? How do you write a recovery middleware to keep an HTTP server running?
*   **Answer:**
    > "*   **Panic:** Stops the ordinary control flow of a goroutine. Deferred functions are executed, and the program exits with a stack trace.
    > *   **Recover:** A built-in function that regains control of a panicking goroutine. It is **only useful inside deferred functions**. Calling `recover()` returns the value passed to `panic()` and stops the panic sequence.
    > 
    > **Recovery Middleware for HTTP:**
    > ```go
    > func RecoveryMiddleware() gin.HandlerFunc {
    >     return func(c *gin.Context) {
    >         defer func() {
    >             if err := recover(); err != nil {
    >                 log.Printf("Recovered from panic: %v", err)
    >                 c.AbortWithStatus(500)
    >             }
    >         }()
    >         c.Next()
    >     }
    > }
    > ```
    > *Note: `recover` only catches panics in the **same goroutine**. If the handler spawns a new goroutine (`go func()`) and that goroutine panics, the main HTTP server will crash unless the new goroutine has its own internal recover block.*"

### Q13: How does the Garbage Collector (GC) work in Go? How can you write code to reduce GC pressure?
*   **Answer:**
    > "Go uses a **concurrent, tri-color mark-and-sweep garbage collector**. 
    > *   It runs concurrently with application execution to minimize Pause Times (Stop-The-World duration), aiming for sub-millisecond pauses.
    > *   It categorizes memory allocations into white (unreachable/garbage), grey (discovered, sub-objects unscanned), and black (reachable/active).
    > 
    > **Reducing GC Pressure:**
    > GC pressure is caused by allocations on the **heap**. To reduce it:
    > 1.  **Reduce allocations:** Avoid frequently allocating short-lived objects. Reuse memory buffers.
    > 2.  **Escape Analysis Optimization:** Keep variables on the **stack** (which is automatically cleared when the function returns, requiring no GC). Avoid passing pointers unnecessarily, returning pointers from functions, or storing pointers in structures when raw values work.
    > 3.  **sync.Pool:** Pre-allocate pools of structures (like byte buffers) and reuse them instead of creating new ones."

### Q14: How does the `init` function work in Go? What is the execution order across multiple packages?
*   **Answer:**
    > "The `init` function is a special function that takes no arguments and returns no values. It runs automatically before `main` starts.
    > 
    > **Execution Order:**
    > 1.  **Imports First:** If package `main` imports package `A`, and `A` imports `B`, Go initializes package `B` first, then package `A`, then package `main`.
    > 2.  **Package-Level Variables:** Variables declared at the package level are evaluated first, then any `init` functions in that package run.
    > 3.  **Multiple `init` functions:** A single package or file can have multiple `init` functions, and they execute in the order they are defined.
    > 4.  **Once only:** Even if a package is imported multiple times, its `init` functions run exactly **once**."

### Q15: What are Struct Tags in Go? How does the JSON package read them?
*   **Answer:**
    > "Struct tags are string literals attached to fields in a struct. They provide metadata that can be queried at runtime.
    > ```go
    > type Target struct {
    >     URL string `json:"url" db:"endpoint_url"`
    > }
    > ```
    > **How they are read:**
    > Packages like `encoding/json` or `sqlx` use Go's **reflection package (`reflect`)** to inspect the types and tags. When marshalling or unmarshalling, the json encoder retrieves the field tag string, parses the `json:"url"` key, and uses that string as the key in the resulting JSON output instead of the default Go field name."

### Q16: How does the `select` statement select a case if multiple channels are ready? How do you implement custom timeouts?
*   **Answer:**
    > "If multiple channel cases are ready to communicate, the `select` statement **randomly selects one** to execute. This prevents starvation of other channels in a busy loop.
    > 
    > **Implementing Custom Timeouts:**
    > We combine `select` with `time.After`, which returns a channel that receives the current time after a duration:
    > ```go
    > select {
    > case res := <-dataChannel:
    >     fmt.Println("Received:", res)
    > case <-time.After(2 * time.Second):
    >     fmt.Println("Timed out waiting for data!")
    > }
    > ```"

### Q17: When should you use Mutexes vs. Channels? Explain the Go proverb: "Do not communicate by sharing memory; instead, share memory by communicating."
*   **Answer:**
    > "The proverb means that instead of protecting shared state with locks (Mutexes) so different threads can access it, you should pass the data (state) over channels so that only **one goroutine owns the data at any point**.
    > 
    > **Guidelines:**
    > *   **Use Channels when:**
    >     *   Passing ownership of data (producer/consumer queues).
    >     *   Coordinating state or broadcasting signals (e.g. broker events).
    >     *   Task delegation.
    > *   **Use Mutexes (`sync.Mutex` or `sync.RWMutex`) when:**
    >     *   Accessing a simple local cache or map.
    >     *   Performance is critical (Mutexes are faster with less overhead than channels).
    >     *   Updating a simple counter or in-memory state variable."

### Q18: What is `sync.Pool`, and how does it optimize memory allocations in high-performance web servers?
*   **Answer:**
    > "`sync.Pool` is a cache of temporary objects that can be individually saved and retrieved. It is used to pre-allocate and reuse frequently allocated buffers or structs to **reduce heap allocations** and **minimize GC pressure**.
    > 
    > **Example in Web Servers:**
    > In high-throughput HTTP servers, encoding JSON responses requires allocating byte buffers. If you allocate a new `[]byte` slice for every request, the GC will consume significant CPU. 
    > Instead, you pull a buffer from `sync.Pool`, write your response, and then return the buffer to the pool using `Put()`. If the pool is empty, it calls a custom allocator function to make a new one."

### Q19: What is the difference between `new()` and `make()` in Go?
*   **Answer:**
    > "*   **`new()`:** A built-in function that allocates memory for a type, zeroes the memory, and returns a **pointer** to it (`*T`). It can be used for any type (structs, ints, arrays).
    >     *   *Example:* `p := new(int)` returns a pointer to an int initialized to `0`.
    > *   **`make()`:** A built-in function that is used **only** for initializing the built-in reference types: **Slices, Maps, and Channels**. It allocates memory, initializes the internal data structures (like maps' hash buckets or slices' headers), and returns an **initialized value** of type `T` (not a pointer).
    >     *   *Example:* `m := make(map[string]int)` returns an active, writable map, whereas `m := new(map[string]int)` returns a pointer to a `nil` map that will panic if written to."

### Q20: What is Variable Shadowing? Give an example and explain how to prevent it.
*   **Answer:**
    > "**Variable Shadowing** occurs when a variable declared inside an inner block (like an `if` statement or loop) has the same name as a variable in an outer block. The inner variable 'shadows' the outer one, rendering the outer variable inaccessible within that block.
    > 
    > **Example:**
    > ```go
    > var client *http.Client // Outer variable
    > if env == "local" {
    >     // Shadowing occurs here due to the ':=' operator
    >     client, err := getLocalClient() 
    >     if err != nil {
    >         return err
    >     }
    >     // 'client' is set correctly inside this scope
    > }
    > // Outer 'client' is still nil here!
    > ```
    > **Prevention:**
    > 1.  Declare variables explicitly and assign them using `=` instead of `:=` inside blocks if you intend to reuse the outer variable:
    >     ```go
    >     var err error
    >     client, err = getLocalClient()
    >     ```
    > 2.  Use linters like `govet` with shadowing checks enabled (`go vet -vettool=$(which shadow)`) to detect shadowed variables in your CI pipeline."
