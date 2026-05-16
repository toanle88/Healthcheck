# Lesson 05: CI/CD & Security Compliance 🤖🛡️

Automation is what makes this project "Enterprise Grade." We use GitHub Actions to ensure that every change is tested, audited, and deployed safely.

## 1. OIDC: The "Secretless" Pipeline 🔑🚫

Look at your GitHub Workflow files (`.github/workflows/`). You won't find any Azure Client Secrets or Passwords there. 

We use **OIDC (OpenID Connect)**. 
1. GitHub generates a temporary "Trust Token."
2. Azure sees the token and says: *"I trust this specific GitHub Repository (toanle88/Healthcheck)."*
3. Azure gives GitHub a temporary access key.
4. The key expires the moment the job is done.

This is the **Gold Standard** for security. Even if someone hacks your GitHub account, there are no permanent passwords for them to steal.

## 2. Checkov: The Automated Auditor 🧐📑

We integrated **Checkov** into the pipeline. It reads your Terraform code *before* it's deployed and looks for mistakes.

We centralized all our "Skips" in `.checkov.yaml`.
- **Why we skip**: For a Dev environment, we don't need expensive things like "Geo-redundant storage" or "Private Endpoints for everything."
- **Why it’s professional**: Instead of just ignoring security, we **document** our exceptions. This is exactly what you do in a real job.

## 3. The Two-Stage Pipeline 🏗️🚀

- **`cicd.yml`**: Handles the "App." It builds the Docker images and pushes them to ACR.
- **`infra.yml`**: Handles the "Environment." It runs `terraform plan` and `apply`.

By separating these, we can update the code (Go) without necessarily touching the infrastructure, which makes deployments much faster.

## 4. Hardened Containers 📦🧱

Look at the `Dockerfile.api`. We use **Distroless** images.
- **Standard Docker**: Has a full operating system inside (shell, package manager, etc.).
- **Distroless**: Contains *only* your Go binary and the minimum files to run it.
- **Security Win**: A hacker cannot "shell" into your container because there is no `sh` or `bash` inside!

---

### Final Key Takeaway
Security isn't a "One-time" thing; it's a **Process**. By putting OIDC, Checkov, and Distroless into our CI/CD, we've made security **automatic**.

**Congratulations!** You've completed the walkthrough. You now understand the full lifecycle of a modern, secure, and automated Azure Cloud application. 🏁🏆✨
