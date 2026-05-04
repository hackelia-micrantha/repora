# SPEC-0003: Sample Configuration

Status: Draft

```yaml
repos:
  - id: payments-api
    canonical:
      provider: gitlab
      url: git@gitlab.com:org/payments-api.git
    mirrors:
      - provider: github
        url: git@github.com:org/payments-api.git
    mode: mirror

  - id: auth-service
    canonical:
      provider: gitlab
      url: git@gitlab.com:org/auth-service.git
    mirrors:
      - provider: github
        url: git@github.com:org/auth-service.git
    mode: mirror
```
