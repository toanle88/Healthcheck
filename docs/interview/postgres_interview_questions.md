# PostgreSQL Technical Interview Prep Guide

This guide contains 20 core PostgreSQL interview questions, covering database architecture, transaction internals, performance tuning, locks, index types, and modern cloud deployment configurations (such as passwordless Azure Flexible Server).

---

## 🐘 PostgreSQL Technical Interview Q&As

### Q1: Explain PostgreSQL's connection architecture. How does it handle client connections, and why is a Connection Pooler needed?
*   **Answer:**
    > "PostgreSQL uses a **process-based connection model** (process-per-connection) rather than a thread-based model. When a client connects, the main postmaster process forks a new, independent backend process (called `postgres: username dbname`) to handle that connection.
    > 
    > **Implications:**
    > Forking processes consumes significant OS memory (approx 10MB per connection) and CPU for context switching. If an application opens 1,000 active connections, the database server will likely crawl to a halt due to resource exhaustion.
    > 
    > **Connection Poolers (PgBouncer / Application Poolers):**
    > To prevent this, we use connection poolers. A pooler maintains a small pool of persistent connections to the database (e.g. 50 connections) and multiplexes hundreds of incoming application client requests over those 50 connections. In our Go code, we use `pgxpool.Pool` to manage this at the application layer, ensuring the API doesn't overwhelm Postgres."

### Q2: What is MVCC (Multi-Version Concurrency Control) in PostgreSQL? How does it affect reads and writes?
*   **Answer:**
    > "**MVCC** is the mechanism Postgres uses to ensure that **readers do not block writers, and writers do not block readers**.
    > 
    > **How it works:**
    > When a transaction updates a row, Postgres does not overwrite the existing data on disk. Instead, it marks the old row as physically deprecated (setting a transaction ID `xmax`) and inserts a completely new version of the row (setting `xmin` to the current transaction ID).
    > 
    > Because multiple versions of the same row exist on disk, other transactions can read the old, consistent version of the data without waiting for the writing transaction to commit.
    > 
    > **Trade-off:**
    > This creates **dead tuples** (unused, old row versions). If left unchecked, these dead tuples cause 'table bloat' and slow down queries."

### Q3: What is the purpose of the `VACUUM` command in PostgreSQL, and how does `autovacuum` work?
*   **Answer:**
    > "Since MVCC leaves dead tuples on disk, we need a way to clean them up. The `VACUUM` command:
    > 1.  Scans tables to find dead tuples that are no longer visible to any active transaction.
    > 2.  Removes those dead tuples and marks the disk space as available for future writes (though it does not return the space to the OS; only `VACUUM FULL` does that by rewriting the table, which requires an exclusive lock).
    > 3.  Updates the visibility map to speed up index-only scans.
    > 
    > **Autovacuum:**
    > `autovacuum` is a background daemon that runs continuously. It periodically checks table statistics (based on inserts, updates, and deletes) and automatically runs a vacuum and analyze (which updates query planner statistics) on tables that have crossed thresholds."

### Q4: Explain the four SQL transaction isolation levels. Which ones are supported by Postgres, and what is the default?
*   **Answer:**
    > "The isolation levels define how changes made by one transaction are visible to others:
    > 
    > 1.  **Read Uncommitted:** (Not supported in Postgres; it defaults to Read Committed). Allows 'Dirty Reads' (reading uncommitted data).
    > 2.  **Read Committed (Postgres Default):** Prevents dirty reads. A query inside a transaction only sees data committed *before that specific query began*. If a row is modified and committed mid-transaction, a second query in the same transaction *will* see the new data (Non-repeatable read).
    > 3.  **Repeatable Read:** Prevents dirty reads and non-repeatable reads. All queries inside the transaction see a snapshot of the database taken *when the transaction first started*. 
    > 4.  **Serializable:** The strictest level. It locks transactions or monitors read/write locks to guarantee that the execution of concurrent transactions yields the exact same state as if they were run one-after-another sequentially (prevents serialization anomalies).
    > 
    > In Postgres, if a concurrent transaction modifies data in a way that violates Repeatable Read or Serializable constraints, Postgres aborts the transaction with a `serialization_failure` (SQLSTATE `40001`), and the application must retry it."

### Q5: What are the main index types in PostgreSQL, and when would you use a GIN or BRIN index instead of a standard B-Tree index?
*   **Answer:**
    > "*   **B-Tree (Default):** Used for comparison operations ($<, \le, =, \ge, >$) and sorting. Ideal for unique IDs, numbers, and strings.
    > *   **GIN (Generalized Inverted Index):** Used for indexing composite values where you want to search for elements *within* the value. Ideal for **JSONB** documents (indexing keys/values), array columns, and Full-Text Search.
    > *   **BRIN (Block Range Index):** Designed for extremely large tables (billions of rows) where data is naturally ordered by physical layout (e.g. timestamp logging tables). Instead of indexing every row, it stores the min/max values for ranges of blocks (e.g., every 1MB of disk space). It occupies a fraction of the size of a B-Tree index.
    > *   **GiST / SP-GiST:** Used for geometric, spatial, and range data (often with PostGIS).
    > *   **Hash:** Fast for simple equality (`=`) checks but rarely used because B-Tree is more versatile."

### Q6: What is the difference between `EXPLAIN` and `EXPLAIN ANALYZE`?
*   **Answer:**
    > "*   **`EXPLAIN`:** Shows the execution plan generated by the query planner based on database statistics, without actually running the query. It lists estimated costs, rows, and operations (e.g. Sequential Scan vs. Index Scan).
    > *   **`EXPLAIN ANALYZE`:** Actually **executes the query** inside the database, measures the real execution times and row counts, and outputs them alongside the planner estimates.
    > 
    > *Warning: Since `EXPLAIN ANALYZE` runs the query, if you run it on a `DELETE` or `INSERT` query, the database will modify the data. To prevent this, you should run it inside a transaction block that you rollback: `BEGIN; EXPLAIN ANALYZE ...; ROLLBACK;`.*"

### Q7: When running database migrations in production, what types of table schema changes require exclusive locks, and how do you avoid downtime?
*   **Answer:**
    > "Operations like adding constraints or creating indexes lock the table, blocking reads and writes:
    > 
    > *   **Adding a Column with a Default Value:** In older Postgres versions ($< 11$), this rewrote the entire table on disk, holding an exclusive lock. In Postgres 11+, this is instant and safe.
    > *   **Creating an Index:** A standard `CREATE INDEX` holds a `ShareLock` which blocks writes. To prevent downtime, we must create indexes concurrently:
    >     ```sql
    >     CREATE INDEX CONCURRENTLY idx_target_url ON targets (url);
    >     ```
    > *   **Adding a Foreign Key:** Acquires an exclusive lock to validate existing rows. To avoid locks:
    >     1. Add the constraint using `NOT VALID` (locks briefly to add definition).
    >     2. Validate it later in a separate command: `ALTER TABLE ... VALIDATE CONSTRAINT ...` (which scans rows without holding an exclusive write lock)."

### Q8: How does passwordless active directory (Entra ID) authentication work in Azure PostgreSQL Flexible Server?
*   **Answer:**
    > "1.  In our Terraform config, we define an Entra ID Admin (such as the User-Assigned Managed Identity of our container apps).
    > 2.  When the Go application starts, it uses the Azure SDK to fetch an OAuth 2.0 access token from the Entra ID token endpoint using its managed identity.
    > 3.  The database client (pgx) opens a connection and presents this access token as the password.
    > 4.  Postgres Flexible Server validates the signature of the Entra ID token, extracts the identity client ID, maps it to a database role we created for that identity, and logs the container in passwordless."

### Q9: What is the Write-Ahead Log (WAL) in PostgreSQL, and why is it vital for durability and recovery?
*   **Answer:**
    > "The **Write-Ahead Log (WAL)** is a transaction log where changes are written *before* they are applied to the actual data pages on disk.
    > 
    > **Durability (ACID):**
    > Writing changes to random data pages on disk is slow because it involves random I/O. Writing to the WAL is append-only (sequential I/O), which is extremely fast. 
    > 
    > When a transaction commits, Postgres guarantees that the changes are flushed to the WAL on disk. If the server crashes or loses power:
    > 1. On startup, Postgres reads the WAL.
    > 2. It compares the WAL entries with the data pages on disk.
    > 3. It replays any committed changes that were not yet flushed to the data pages (**redo**), ensuring zero data loss."

### Q10: What are Geo-Redundant Backups, and how do they differ from Local Backups? What is Point-in-Time Recovery (PITR)?
*   **Answer:**
    > "*   **Local Backups:** Backup files are stored in the same geographic region as the database server. If the entire datacenter region goes offline due to a disaster, the database and the backups are lost.
    > *   **Geo-Redundant Backups:** Backups are replicated asynchronously to a secondary paired region (e.g. from East Asia to Southeast Asia). If the primary region is destroyed, we can restore the database in the secondary region.
    > *   **Point-in-Time Recovery (PITR):** A recovery method that uses a combination of full database backups and continuous WAL archiving. It allows us to restore the database to the exact second of our choice within our retention window (e.g., restoring to 09:34:12 AM yesterday right before an engineer ran a bad delete query)."

### Q11: What is the difference between JSON and JSONB in PostgreSQL? When should you use JSONB, and how do you index it?
*   **Answer:**
    > "*   **JSON:** Stores the exact text representation of the JSON input. It preserves whitespace and duplicate keys. 
    >     *   *Pros:* Fast write times.
    >     *   *Cons:* Very slow read times (Postgres has to parse the text every time you query a key).
    > *   **JSONB (JSON Binary):** Stores JSON in a decomposed binary format. It strips whitespace and removes duplicate keys.
    >     *   *Pros:* Extremely fast query times because keys are indexed internally.
    >     *   *Cons:* Slightly slower write times due to parsing overhead.
    > 
    > **Indexing JSONB:**
    > We use a **GIN index** on JSONB columns to enable fast searches:
    > ```sql
    > CREATE INDEX idx_results_payload ON results USING gin (payload);
    > ```
    > This allows queries like `payload @> '{"status": "healthy"}'` to execute in milliseconds using the index."

### Q12: How do you identify and kill blocked queries or deadlocks in PostgreSQL?
*   **Answer:**
    > "I query the system catalog view `pg_stat_activity` to find blocked transactions:
    > ```sql
    > SELECT pid, query, state, wait_event_type, wait_event 
    > FROM pg_stat_activity 
    > WHERE wait_event IS NOT NULL;
    > ```
    > To find which query is blocking another, I can write a join between `pg_locks` and `pg_stat_activity`.
    > 
    > **Remediation:**
    > Once I identify the Process ID (`pid`) of the blocking query:
    > 1.  Cancel the query politely: `SELECT pg_cancel_backend(pid);`
    > 2.  If it doesn't respond, terminate the connection forcefully: `SELECT pg_terminate_backend(pid);`"

### Q13: How does the `ON DELETE CASCADE` constraint affect performance and lock management in large tables?
*   **Answer:**
    > "While `ON DELETE CASCADE` is convenient (deleting a parent row automatically deletes all child rows with foreign keys), it has hidden dangers in large databases:
    > 
    > 1.  **Row Locks:** Deleting a parent row triggers a cascade that locks matching rows in the child tables. If there are millions of child rows, this holds locks for a long time, causing transaction queues to pile up.
    > 2.  **Performance:** If the child table's foreign key column is **not indexed**, Postgres must perform a slow **sequential scan** on the child table for *every* parent row deleted to find children to delete. (Best practice: Always index foreign key columns)."

### Q14: What is the difference between a CTE (Common Table Expression) and a Subquery? Are CTEs materialized in Postgres?
*   **Answer:**
    > "*   **CTE:** Defined using the `WITH` clause. It makes complex queries readable by breaking them into logical blocks.
    > *   **Subquery:** A query nested inside another query (e.g. `SELECT * FROM (SELECT...)`).
    > 
    > **Materialization:**
    > Historically (Postgres $< 12$), CTEs were always **materialized** (the database executed the CTE query, wrote the results to a temporary table in memory, and then ran the outer query on that temp table). This prevented the query planner from pushing indexes from the outer query into the CTE, causing performance bottlenecks.
    > 
    > In Postgres 12+, CTEs are **inlined by default** unless you explicitly add the `MATERIALIZED` keyword, allowing the planner to optimize them just like a subquery."

### Q15: What is a Window Function, and how does it differ from a `GROUP BY` clause?
*   **Answer:**
    > "*   **`GROUP BY`:** Collapses multiple rows into a single summary row (reducing the total row count).
    > *   **Window Function:** Performs a calculation across a set of table rows related to the current row, but **retains all individual rows** in the output. It uses the `OVER` clause.
    > 
    > *Example:* Calculating a running total or a moving average:
    > ```sql
    > SELECT date, status, 
    >        COUNT(*) OVER(PARTITION BY status ORDER BY date) as running_count
    > FROM ping_results;
    > ```
    > This returns every single ping result row, but attaches a cumulative count of pings for that status next to each row."

### Q16: When and why would you use Table Partitioning in PostgreSQL?
*   **Answer:**
    > "Table Partitioning splits one logically large table into smaller physical pieces (partitions) on disk.
    > 
    > **When to use:**
    > When a table exceeds the size of physical memory (e.g., billions of rows of historical ping logs in our app).
    > 
    > **Why to use:**
    > 1.  **Query Performance:** Postgres uses **partition pruning**. If a query filters by date (`WHERE created_at > '2026-05-01'`), the query planner skips scanning all partitions for other months, executing the query in a fraction of the time.
    > 2.  **Maintenance:** Dropping old data becomes instant. Instead of running a slow `DELETE FROM logs WHERE created_at < '2025-01-01'` (which generates massive WAL logs and dead tuples), we can simply run:
    >     ```sql
    >     ALTER TABLE logs DETACH PARTITION logs_y2024;
    >     DROP TABLE logs_y2024;
    >     ```"

### Q17: What is the difference between Physical (Streaming) Replication and Logical Replication?
*   **Answer:**
    > "*   **Physical Replication:** Copies the actual byte changes of the disk blocks (WAL blocks) from the primary server to the replica. 
    >     *   *Pros:* Extremely fast, low overhead.
    >     *   *Cons:* The replica must be a read-only mirror of the *entire* database cluster, running the exact same major version of Postgres.
    > *   **Logical Replication:** Streams database changes based on SQL-like operations (inserts, updates, deletes) targeting specific tables (publish/subscribe model).
    >     *   *Pros:* Allows replication of a subset of tables, replicating between different major versions of Postgres, and bidirectional replication.
    >     *   *Cons:* Higher CPU overhead than physical replication."

### Q18: How do Prepared Statements protect against SQL Injection? How does the Go `pgx` driver leverage them?
*   **Answer:**
    > "Prepared statements separate the query structure from the query data:
    > 
    > 1.  The application sends the query template to Postgres: `PREPARE my_query AS SELECT * FROM users WHERE username = $1;`
    > 2.  Postgres compiles, parses, and plans the query.
    > 3.  The application executes the query passing parameters: `EXECUTE my_query('john');`
    > 
    > Because the parameters are sent separately, even if a user passes a string like `'john; DROP TABLE users;'`, the database treats it strictly as a literal text value for `$1`, preventing it from executing as SQL code.
    > 
    > **Go pgx driver:**
    > The `pgx` driver automatically manages prepared statements under the hood. When you execute queries using placeholders (e.g., `db.Query(ctx, "SELECT ... WHERE id = $1", id)`), `pgx` prepares the query on the database connection and caches the prepared statement, improving performance on subsequent executions."

### Q19: Explain what an "Upsert" is, and how you write one in PostgreSQL.
*   **Answer:**
    > "An **Upsert** is an operation that inserts a row if it does not exist, or updates the existing row if it conflicts with a unique key constraint.
    > 
    > In Postgres, this is written using the `ON CONFLICT` clause:
    > ```sql
    > INSERT INTO targets (name, url, updated_at) 
    > VALUES ('GitHub', 'https://api.github.com', NOW())
    > ON CONFLICT (url) 
    > DO UPDATE SET updated_at = EXCLUDED.updated_at;
    > ```
    > *Here, `url` must have a UNIQUE constraint. If the URL already exists, Postgres aborts the insert and updates the `updated_at` column instead, ensuring the operation is idempotent.*"

### Q20: What is Transaction ID (TxID) Wraparound, and why is it a critical database danger?
*   **Answer:**
    > "In PostgreSQL, transaction IDs are represented as 32-bit integers (approx 4.2 billion values). At any point, half of these numbers are in the 'past' and half are in the 'future' to support MVCC visibility checks.
    > 
    > If a database executes **2 billion write transactions** without a freeze vacuum running, the transaction IDs will 'wrap around' the 32-bit limit. To Postgres, active old data will suddenly appear to be in the 'future,' making old database rows invisible.
    > 
    > **Prevention:**
    > To prevent catastrophic data loss, if the transaction ID age reaches a critical threshold, PostgreSQL will enter **emergency read-only mode** and shut down write operations. 
    > The solution is to ensure **autovacuum** is running properly, which performs 'freeze' operations (marking old rows as permanently visible so their transaction ID is retired)."
