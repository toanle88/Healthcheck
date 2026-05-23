# Classic DevOps & Systems Engineering Foundations

This guide covers generic Linux, networking, databases, Git, and systems troubleshooting questions. These represent the core foundational topics that are asked in almost every DevOps/SRE interview, regardless of the cloud provider or project architecture.

---

## 🐧 Category 1: Linux Systems Troubleshooting

### Q1: A Linux server is running extremely slow. What is your step-by-step process to find the bottleneck?
*   **Answer:**
    > "I troubleshoot Linux performance bottlenecks by isolating the issue into four primary resource categories: CPU, Memory, Disk I/O, and Network.
    > 
    > 1.  **CPU & Overall Load:** I run `top` or `htop` to check the System Load Average (1, 5, and 15-minute intervals). If the load average is higher than the number of CPU cores, the CPU is saturated. I look at CPU percentages: `us` (user space processing), `sy` (kernel/system processing), and `wa` (CPU waiting for Disk I/O to complete).
    > 2.  **Memory:** I run `free -m` or `free -g` to check total, used, and free memory. I look specifically at the `available` column and the use of `swap`. If swap usage is high, the system is low on RAM and is swapping to disk, which severely degrades performance. I check `dmesg | grep -i oom` to see if the kernel's Out-Of-Memory killer has terminated any processes.
    > 3.  **Disk Space and I/O:** I run `df -h` to verify if any disk volume is 100% full (which causes many applications to fail writes and hang). If space is fine, I run `iostat -xz 1` to check disk utilization `%util`. If utilization is high (e.g. near 100%) and wait times (`await`) are long, disk read/write throughput is the bottleneck.
    > 4.  **Network:** I run `ss -tulpn` or `netstat -tulpn` to check which ports are open and if there are excessive connections. I use `ping` or `traceroute` to test network latency and path routing, and check interface statistics to see if packets are being dropped."

### Q2: What is the difference between Hard Links and Symbolic Links (Symlinks) in Linux?
*   **Answer:**
    > "*   **Hard Link:** A hard link is an additional directory entry pointing directly to the same underlying disk storage location (**inode**) as the original file. 
    >     *   If you delete the original file, the hard link still works and retains the content, because the inode is only deleted when all hard links pointing to it are deleted. 
    >     *   Hard links cannot cross filesystems and cannot link directories.
    > *   **Symbolic Link (Symlink/Soft Link):** A symbolic link is a special file that contains the path to another file or directory (similar to a shortcut in Windows). 
    >     *   If you delete the target file, the symlink becomes 'broken' or 'dangling.'
    >     *   Symlinks can point to directories and can span across different filesystems."

---

## 🌐 Category 2: Networking & Protocols

### Q3: What happens behind the scenes when you type `https://example.com` into your browser and press Enter?
*   **Answer:**
    > "This is a multi-step process involving DNS, TCP, TLS, and HTTP:
    > 
    > 1.  **DNS Resolution:** The browser checks its local cache, OS cache, router cache, and ISP DNS server. If not found, it queries DNS root servers down to the Authoritative Name Server to resolve `example.com` into an IP address.
    > 2.  **TCP Connection (3-Way Handshake):** The browser initiates a TCP connection to the destination IP (usually port 443 for HTTPS) via:
    >     *   `SYN` (Client to Server)
    >     *   `SYN-ACK` (Server to Client)
    >     *   `ACK` (Client to Server)
    > 3.  **TLS Handshake (Security):** The client and server negotiate encryption parameters:
    >     *   Client sends a `ClientHello` (supported TLS versions, cipher suites).
    >     *   Server responds with `ServerHello`, its SSL/TLS Certificate (containing public key), and chosen cipher.
    >     *   The client validates the certificate signature against its pre-installed Trusted Certificate Authorities (CAs).
    >     *   The client and server generate a shared symmetric key (session key) using asymmetric encryption. All subsequent traffic is encrypted with this session key.
    > 4.  **HTTP Request & Response:** The browser sends an encrypted HTTP `GET` request. The web server processes the request, queries databases if necessary, and returns an HTTP response (e.g. `200 OK` along with HTML/JS/CSS).
    > 5.  **Rendering:** The browser parses the HTML and renders the page."

---

## 🛠️ Category 3: Git & Version Control

### Q4: What is the difference between `git merge` and `git rebase`? When would you use each?
*   **Answer:**
    > "*   **Git Merge:** Combines changes from one branch into another by creating a new 'merge commit' that binds both histories together.
    >     *   *Pros:* Preserves complete, chronological history of when commits actually occurred. It is safe because it doesn't rewrite history.
    >     *   *Cons:* Can make the commit graph messy and hard to read with frequent merge commits.
    > *   **Git Rebase:** Takes all commits from your branch and replays them on top of the target branch, rewriting the commit history.
    >     *   *Pros:* Results in a clean, perfectly linear commit history (easy to read and trace).
    >     *   *Cons:* Rewrites git history. **Never rebase public shared branches** (like `main` or `release`), as it will desynchronize other developers' histories.
    > *   **Usage:** I use `git rebase` locally on my personal feature branch to keep it up to date with `main` before submitting a Pull Request. Once approved, we use a squash-merge or standard merge to combine it into the main trunk."

---

## 💾 Category 4: Databases & Performance

### Q5: A production SQL database query suddenly becomes very slow. How do you analyze and resolve the issue?
*   **Answer:**
    > "1.  **Analyze the Query Execution Plan:** I run the query prefixed with `EXPLAIN` or `EXPLAIN ANALYZE` in PostgreSQL. This tells me exactly how the database engine is fetching data. I look for `Seq Scan` (sequential table scans), which means the database is reading the entire table row-by-row because there is no index, and compare it to `Index Scan`.
    > 2.  **Add Indexes:** If a query filters (`WHERE` clause) or joins (`JOIN`) on non-indexed columns, I create an index (e.g. `CREATE INDEX ON table (column)`).
    > 3.  **Check Connection Pool Saturation:** Sometimes, it is not the query that is slow, but the application is waiting to acquire a connection from the pool. I check active database connections and increase pool limits or configure connection timeouts.
    > 4.  **Check Table Bloat & Statistics:** In Postgres, I run `VACUUM ANALYZE` to clean up dead rows (bloat) and update the database statistics so the query planner can choose the most efficient execution path."

---

## 📜 Category 5: Scripting & Automation

### Q6: Write a quick script concept to find all lines in a file named `/var/log/nginx/error.log` that contain the word '500' and count how many times it occurred.
*   **Answer:**
    > "*   **Bash One-liner:**
    >     ```bash
    >     grep -c '500' /var/log/nginx/error.log
    >     ```
    >     *(Note: `grep -c` prints only the count of matching lines).*
    > *   **Python Script (if advanced processing is needed):**
    >     ```python
    >     count = 0
    >     try:
    >         with open('/var/log/nginx/error.log', 'r') as file:
    >             for line in file:
    >                 if '500' in line:
    >                     count += 1
    >         print(f"Total occurrences of 500: {count}")
    >     except FileNotFoundError:
    >         print("Log file not found.")
    >     ```"

---

## 🏗️ Category 6: Infrastructure Drift & Reliability Engineering

### Q7: What is "Configuration Drift" in infrastructure, how does it occur, and how do you detect and remediate it using Terraform?
*   **Answer:**
    > "**Configuration Drift** occurs when the actual state of cloud resources deviates from the defined state in your Infrastructure as Code (IaC) configuration. This typically happens when team members manually modify resources in the cloud portal (e.g. changing firewall rules, scaling VM sizes, or editing database permissions) bypassing the Git/IaC pipeline.
    > 
    > **Detection:**
    > *   Running `terraform plan` compares the active state in the cloud (queried via APIs) with the local configuration files and the recorded `terraform.tfstate`. 
    > *   If a resource has been modified or deleted manually, Terraform lists it as a drift difference in the plan output.
    > *   In a production setup, we automate this by scheduling a nightly speculative pipeline job that runs `terraform plan -detailed-exitcode`. If it returns an exit code of `2`, it indicates drift, and sends an alert.
    > 
    > **Remediation:**
    > *   **If the manual change was a mistake:** I simply run `terraform apply`. Terraform will automatically recreate, modify, or revert the cloud resources to match the configuration files.
    > *   **If the manual change was correct and needed:** I update the local Terraform code to match the manual change in the cloud, run `terraform plan` to verify there are zero differences, and commit the code so that future pipeline runs don't overwrite it."

### Q8: What is the difference between an SLI, an SLO, and an SLA in Site Reliability Engineering?
*   **Answer:**
    > "These terms define the reliability goals and agreements for a service:
    > 
    > 1.  **SLI (Service Level Indicator):** The quantitative metric that measures how a service is performing in real-time. It is expressed as: 
    >     $$\text{SLI} = \frac{\text{Good Events}}{\text{Total Events}} \times 100$$
    >     *   *Example:* The ratio of HTTP requests returning `2xx`/`3xx` status codes to total HTTP requests over a given window.
    > 2.  **SLO (Service Level Objective):** The target goal or reliability threshold set by the engineering and product teams. It defines the target SLI value.
    >     *   *Example:* We target an availability SLO of $\ge 99.9\%$ for our HTTP API over a rolling 30-day window.
    > 3.  **SLA (Service Level Agreement):** The formal, legal agreement made with the end customers. It commits the business to the SLO and outlines the financial, legal, or service credits paid back to the customer if the service fails to meet the commitment. The SLA threshold is almost always lower/looser than the internal SLO to give the engineering team a safety buffer.
    >     *   *Example:* The company promises $99.0\%$ availability. If we fall below it, the customer gets a $10\%$ refund on their monthly bill."
