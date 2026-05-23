# DevOps Hands-On Troubleshooting Drills

To *truly* understand how your infrastructure works, you must know how it fails. Running these drills on your local machine will give you the practical troubleshooting experience that interviewers love to ask about.

---

## 🛑 Drill 1: Simulate a Database Outage (Resilience Drill)

*   **Goal:** Observe how the Go application behaves when its database is offline. Does it crash loop? Does it reconnect automatically?
*   **The Execution:**
    1.  Start the local stack: `docker-compose up --build`
    2.  In a separate terminal, stop only the database container:
        ```bash
        docker-compose stop db
        ```
    3.  Refresh the React Frontend or call the status endpoint:
        ```bash
        curl http://localhost:8080/api/status
        ```
    4.  Observe the API container logs:
        ```bash
        docker-compose logs api
        ```
*   **What to Look For:**
    *   Does the API return a clean HTTP `500` error or does the Go binary panic and exit? (A well-designed Go backend should handle database connection errors gracefully without crashing the web server).
    *   Start the database again: `docker-compose start db`. Does the Go API automatically recover and reconnect, or does it require a restart? (Verify that `pgx/v5` connection pool handles auto-reconnection).

---

## 🚨 Drill 2: Fail a Smoke Test & Watch a CD Rollback

*   **Goal:** Force the CI/CD pipeline to deploy a broken container, and watch the pipeline automatically roll back to the previous stable release.
*   **The Execution:**
    1.  Open the Go API health check handler file: [internal/handler/health.go](file:///mnt/d/Dev/Projects/Healthcheck/internal/handler/health.go) (or the corresponding file handling `/health`).
    2.  Temporarily edit the code to force it to return a `500 Internal Server Error` instead of a `200 OK`.
    3.  Commit this broken change and push it to a development branch to trigger the pipeline.
    4.  Watch the CD deployment phase logs in GitHub Actions or Azure DevOps.
*   **What to Look For:**
    *   Verify that the deploy stage finishes building and pushes the image, but the **Smoke Test** step fails.
    *   Look at the pipeline logs to verify the execution of the rollback command.
    *   Verify in your Azure Portal that the active revision of the Container App was successfully reverted back to the previous, healthy revision.

---

## 🔒 Drill 3: Trigger a Checkov Policy Violation

*   **Goal:** Understand how static analysis security scans enforce standards in CI pipelines.
*   **The Execution:**
    1.  Open the dev infrastructure networking configuration: [infra/terraform/environments/dev/main.tf](file:///mnt/d/Dev/Projects/Healthcheck/infra/terraform/environments/dev/main.tf).
    2.  Add an insecure resource block (for example, a public security group rule that allows all traffic on port 22 (SSH) from `0.0.0.0/0`).
    3.  Run a Checkov scan locally to see how it flags the security issue:
        ```bash
        checkov -d infra/terraform/environments/dev
        ```
    4.  Commit and push this change to your repository.
*   **What to Look For:**
    *   Observe how the CI pipeline's **Checkov Static Analysis** step fails the build, preventing your pull request from being merged.
    *   Identify the exact rule ID (e.g. `CKV_AZURE_...`) that was violated and read the Checkov description of why this rule is vital in production.

---

## 🔍 Drill 4: Inspect Distributed Trace Propagation

*   **Goal:** Visually trace an HTTP call across container boundaries.
*   **The Execution:**
    1.  Ensure the entire local stack is running: `docker-compose up --build`
    2.  Open your browser and navigate to the local Jaeger dashboard: [http://localhost:16686](http://localhost:16686).
    3.  Open the React Frontend ([http://localhost:5173](http://localhost:5173)) and click around to trigger dashboard refreshes.
    4.  In Jaeger, select the `api` service and click **Find Traces**.
    5.  Select one of the pings or status traces and expand the tree.
*   **What to Look For:**
    *   Confirm that a single trace spans across the React frontend request, to the Go API, and shows the query time on the database.
    *   Observe the trace tags: can you see the HTTP route, status code, and IP addresses? This is exactly how SREs troubleshoot microservice latency spikes in production.
