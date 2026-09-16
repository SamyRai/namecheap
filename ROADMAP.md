# ZoneKit Long-Term Roadmap

This document outlines the strategic vision and development roadmap for ZoneKit over the next 5 years.

## 🟢 0-3 Months: Foundation & Stability (v0.x - v1.0)

**Theme: "Safe, Reliable, and Correct"**

The immediate focus is on eliminating technical debt, ensuring data safety, and finalizing the core provider contract to reach a stable v1.0 release.

*   **Safety & Correctness (Priority #1)**
    *   [ ] **Atomic Operations**: Eliminate non-atomic bulk updates. Implement intelligent diffing in `Service.BulkUpdate` to minimize API calls and prevent data loss.
    *   [ ] **Context Propagation**: Ensure `context.Context` is threaded through every layer of the application for proper timeout and cancellation handling.
    *   [ ] **Validation**: Implement strict schema validation for all provider configurations and DNS records.

*   **Provider Ecosystem**
    *   [ ] **Conformance Suite**: Expand the conformance test harness to cover 100% of the `Provider` interface (CRUD, Edge Cases).
    *   [ ] **Core Providers**: Fully support Cloudflare, Namecheap, AWS Route53, and Google Cloud DNS with production-grade reliability.

*   **Developer Experience**
    *   [ ] **Structured Logging**: Replace ad-hoc logging with `log/slog` for structured, machine-readable output.
    *   [ ] **Error Handling**: Standardize error types across all providers (e.g., `ErrRecordNotFound`, `ErrAuthenticationFailed`).

---

## 🟡 3-6 Months: Advanced Features & Ecosystem (v1.x)

**Theme: "Power User & Automation"**

Once the core is stable, we shift focus to enabling complex workflows, automation, and broader integrations.

*   **Advanced DNS Management**
    *   [ ] **Zone Sync**: One-way synchronization between providers (e.g., "Primary: Cloudflare" -> "Backup: Route53").
    *   [ ] **Dry Run**: Reliable "what-if" analysis for all operations, showing exactly what records will be created, updated, or deleted.
    *   [ ] **Record Templates**: Support for templated zones (e.g., "Standard Mail Setup", "Web Server Basic") for rapid provisioning.

*   **Infrastructure as Code (IaC)**
    *   [ ] **Terraform Provider**: Release an official Terraform provider wrapping ZoneKit logic.
    *   [ ] **GitOps Integration**: Native support for managing DNS configuration via Git repositories (YAML/JSON definitions).

*   **Observability**
    *   [ ] **Metrics**: Expose Prometheus metrics for API calls, latencies, and error rates.
    *   [ ] **Audit Logs**: Comprehensive audit logging for all changes made via the tool.

---

## 🔵 6-12 Months: Enterprise & Scale (v2.x)

**Theme: "Enterprise Ready"**

Focus on multi-tenancy, team management, and handling massive scale.

*   **Enterprise Security**
    *   [ ] **SSO/OIDC**: Support for retrieving provider credentials via enterprise identity providers.
    *   [ ] **RBAC**: Granular permissions for API keys (e.g., "Read Only", "Zone Specific Write").
    *   [ ] **Vault Integration**: Native integration with HashiCorp Vault for secret management.

*   **Performance**
    *   [ ] **Parallel Execution**: Concurrent processing of multi-zone operations for high-performance updates.
    *   [ ] **Caching**: Intelligent caching layer to reduce API costs and improve latency for read operations.

---

## 🟣 1-5 Years: Platform & Intelligence (v3.x+)

**Theme: "The DNS Platform"**

Long-term evolution from a CLI tool to a comprehensive DNS management platform.

*   **SaaS Evolution**
    *   [ ] **ZoneKit Cloud**: A managed SaaS offering providing a web UI and unified API over all your DNS providers.
    *   [ ] **Global API**: A single, normalized API endpoint that routes to any underlying provider (Cloudflare, AWS, etc.).

*   **Intelligent Automation (AI)**
    *   [ ] **AI-Driven Optimization**: Automatic suggestions for DNS misconfigurations (e.g., missing SPF/DMARC, dangling CNAMEs).
    *   [ ] **Smart Routing**: Dynamic updates to DNS records based on real-time latency or uptime monitoring of endpoints.
    *   [ ] **Anomaly Detection**: Alerts for unusual DNS record changes or query patterns.

*   **Global Ecosystem**
    *   [ ] **Marketplace**: A community marketplace for custom provider plugins and automation scripts.
    *   [ ] **Standardization**: Work towards establishing the "ZoneKit Schema" as an industry-standard format for vendor-agnostic DNS definition.
