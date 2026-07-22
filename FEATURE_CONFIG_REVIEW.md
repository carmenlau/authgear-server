# `authgear.features.yaml` (Feature Config) Review

Source: `pkg/lib/config/feature*.go` (schema + Go structs), `pkg/lib/config/default.go` (default-filling
mechanism), `pkg/lib/config/testdata/default_feature.yaml` (fixture), verified against actual
consumers in `pkg/lib/ratelimit/`, `pkg/lib/messaging/`, `pkg/lib/authn/otp/`, `pkg/admin/graphql/`,
`pkg/portal/model/app.go`.

## How feature config works

Feature config (`authgear.features.yaml`) is a **separate file from `authgear.yaml`** (the project/app
config). It is loaded by the same reflection-based defaulting mechanism (`SetFieldDefaults` in
`default.go`), and — for SaaS deployments — is generated/managed per-plan and merged in layers via
`FeatureConfig.Merge`, unlike `authgear.yaml` which tenants edit directly.

### The four merge layers, confirmed from the resource-loading code

The user's mental model of **code default → cluster → plan → app** is directionally correct. Traced through
`pkg/util/resource/fs.go`, `manager.go`, and `pkg/lib/config/configsource/database.go`:

| # | Layer | Source | Where it's built |
|---|---|---|---|
| 1 (lowest) | **Code default** (`FsLevelBuiltin`) | Files embedded into the binary at compile time from `resources/authgear/` (`pkg/lib/deps/providers.go`). | **No `authgear.features.yaml` exists in that embedded tree today**, so this layer currently contributes nothing at merge time. The real "code default" behavior instead comes from `SetFieldDefaults`/each type's `SetDefaults()` (`default.go`), which is applied **once, after all 4 layers are merged**, to fill in whatever no layer set. |
| 2 | **Cluster** (`FsLevelCustom`) | An OS directory mounted into the server process, from the `CUSTOM_RESOURCE_DIRECTORY` (or `PORTAL_CUSTOM_RESOURCE_DIRECTORY`) env var (`pkg/util/resource/manager.go:41-48`). | **Optional** — only exists if that env var is set for the deployment. Shared by every app on that install; this is the true "cluster/install-wide override" layer. |
| 3 | **Plan** | A named pricing plan's `feature_config`, stored in Postgres table `_portal_plan`, looked up per-app by `PlanName` (`pkg/lib/config/plan/store.go`, `configsource/database.go:407-421,455`). Managed via the portal pricing CLI (`cmd/portal/cmd/cmdpricing`). | Only present in the DB-backed (SaaS) config source. **Absent in the local-fs/self-hosted config source** (`configsource/local_fs.go`) — single-tenant installs have no plan layer at all. |
| 4 (highest) | **App** | The app's own `authgear.features.yaml`, stored in its DB config source row (or a local directory in self-hosted mode). | Always present as the final overlay: `d.BaseResources.Overlay(planFs).Overlay(appFs)` (`configsource/database.go:469-470`). |

**Precedence is confirmed strictly by call order**, not by field-level "smartest wins" logic: `viewEffectiveResource`
(`configsource/resources.go:694-717`) parses every layer's raw YAML, then folds them left-to-right —
`mergedConfig = mergedConfig.Merge(cfg)` for each layer in turn — so **whichever layer is folded in last wins**.
Since the filesystem stack is assembled as `[Builtin, Custom?, Plan, App]`, the effective order is:

```
App  >  Plan  >  Cluster (Custom)  >  Code default
(highest precedence)                  (lowest precedence)
```

Defaults (`SetFieldDefaults`) are applied last of all, via `config.ParseFeatureConfig` on the fully-merged
YAML (`resources.go:713`) — so a field is only ever defaulted if **none** of the 4 layers set it.

### Merge granularity differs per section — whole-section replace vs. field-level merge

`FeatureConfig.Merge` calls each top-level section's own `Merge` method, and **that method decides whether
"a higher layer touches this section" means "override every field in it" or "override only the specific
sub-fields it sets."** This is not uniform across the file — confirmed by reading every `feature_*.go`:

- **Whole-section replace** (most sections): `identity`, `authentication`, `authenticator`, `ui`,
  `custom_domain`, `hook`, `audit_log`, `google_tag_manager`, `rate_limits` (top-level), `collaborator`,
  `test_mode`, `fraud_protection`. Each of these `Merge` methods is a one-liner: `if layer.X == nil {
  return c }; return layer.X` — i.e. if a higher layer's YAML mentions this section **at all**, that
  layer's *entire* section replaces the lower layer's, field and all. **Practical trap:** if the Plan
  layer sets `test_mode: { fixed_oob_otp: {...} }` and the App layer separately sets
  `test_mode: { sms: { suppressed: true } }`, the App layer's `test_mode` object wins *in its entirety* —
  the Plan's `fixed_oob_otp` is silently lost, because App didn't repeat it. There is no per-field
  inheritance within these sections across layers.
- **Field-level merge** (deep merge, each leaf independently inherited/overridden): `oauth.client`
  (`OAuthClientFeatureConfig.Merge` merges `maximum`/`soft_maximum`/`custom_ui_enabled`/`app2app_enabled`
  independently), `messaging` (`MessagingFeatureConfig.Merge` merges `rate_limits`/`sms_usage`/`email_usage`
  /`whatsapp_usage`/`sms_usage_count_disabled`/etc. independently — though `rate_limits` itself, once you're
  inside it, is again whole-object: the entire `sms`/`sms_per_ip`/etc. bundle replaces together), `admin_api`
  (`create_session_enabled`/`user_import_usage`/`user_export_usage` independently merged), and `usage.limits`
  (each of `sms`/`email`/`whatsapp`/`user_import`/`user_export` independently merged, but see below for `hooks`).
- **Append, not replace** (unique to one field): `usage.hooks[]` — `FeatureUsageConfig.Merge` does
  `merged.Hooks = append(merged.Hooks, layer.Usage.Hooks...)`, so usage-threshold webhooks from Cluster,
  Plan, and App **all accumulate** rather than the higher layer replacing the lower one.
- **Never actually merges — confirmed bug/gap**: `web3` (`Deprecated_Web3FeatureConfig`) has **no `Merge`
  method at all** and no `MergeableFeatureConfig` assertion (the only section missing one — verified against
  all 16 other `feature_*.go` files, which all declare `var _ MergeableFeatureConfig = &XFeatureConfig{}`).
  Because `FeatureConfig.Merge`'s reflection loop only touches fields that implement the interface, `web3`
  is silently skipped on every merge pass — **no layer (Cluster/Plan/App) can ever actually override
  `web3.nft.maximum`**; it always resolves to the hardcoded default of `3` regardless of what any layer's
  YAML says. This is consistent with `web3` being dead/deprecated, but is worth flagging explicitly as a correctness gap, not just an unused feature.

Across almost every section, the same two-layer *conceptual* pattern (as opposed to the *merge mechanics*
above) repeats between `authgear.yaml` and `authgear.features.yaml` as files:

- **App config (`authgear.yaml`)** — what the tenant *wants*: a value, a limit, a rule.
- **Feature config (`authgear.features.yaml`)** — what the tenant's *plan allows*: an on/off switch or a
  ceiling that the app config's value is clamped against.

Two concrete mechanisms implement this, confirmed by reading the consumers:

1. **Ceiling/ratchet** (e.g. messaging rate limits): both configs are read, and the *stricter* one wins.
   `pkg/lib/ratelimit/ratelimits.go` picks whichever of app vs. feature config has the lower `Rate()`.
2. **Independent OR-gate** (e.g. test mode suppression, fixed OTP): the app config decides *which
   targets* a rule applies to; the feature config is a separate, blanket plan-tier switch. Either one
   firing suppresses/overrides — they don't merge their values, they're just two independent checks in
   sequence (see `pkg/lib/messaging/sender.go`, `pkg/lib/authn/otp/form.go`).

---

## 1. Full default config (YAML)

This is the authoritative default, derived from every `SetDefaults()` in `pkg/lib/config/feature*.go`
and cross-checked against `ParseFeatureConfig(ctx, []byte("{}"))`. It differs slightly from
`pkg/lib/config/testdata/default_feature.yaml` — see [Appendix: stale test fixture data](#appendix-stale-test-fixture-data-found)
for the two discrepancies found and why they don't affect real config files.

```yaml
identity:
  login_id:
    types:
      phone:
        disabled: false
  oauth:
    maximum_providers: 99
    providers:
      google:
        disabled: false
      facebook:
        disabled: false
      github:
        disabled: false
      linkedin:
        disabled: false
      azureadv2:
        disabled: false
      azureadb2c:
        disabled: false
      adfs:
        disabled: false
      apple:
        disabled: false
      wechat:
        disabled: false
  biometric:
    disabled: false
authentication:
  secondary_authenticators:
    oob_otp_sms:
      disabled: false
authenticator:
  password:
    policy:
      minimum_guessable_level:
        disabled: false
      excluded_keywords:
        disabled: false
      history:
        disabled: false
ui:
  white_labeling:
    disabled: false
  phone_input: {}
custom_domain:
  disabled: false
oauth:
  client:
    maximum: 99
    soft_maximum: 99
    custom_ui_enabled: false
    app2app_enabled: false
hook:
  blocking_handler:
    maximum: 99
  non_blocking_handler:
    maximum: 99
audit_log:
  retrieval_days: -1
google_tag_manager:
  disabled: false
rate_limits:
  disabled: false
messaging:
  rate_limits:
    sms: {}
    sms_per_ip:
      enabled: true
      period: 1m
      burst: 60
    sms_per_target:
      enabled: true
      period: 1h
      burst: 10
    email: {}
    email_per_ip:
      enabled: true
      period: 1m
      burst: 200
    email_per_target:
      enabled: true
      period: 24h
      burst: 50
  sms_usage:
    enabled: false
  email_usage:
    enabled: false
  whatsapp_usage:
    enabled: false
  sms_usage_count_disabled: false
  whatsapp_usage_count_disabled: false
  custom_sms_provider_disabled: false
  template_customization_disabled: false
  custom_smtp_disabled: false
# usage: (omitted — nullable, stays unset/nil unless explicitly configured; no auto-populated defaults)
collaborator: {}
web3:
  nft:
    maximum: 3
admin_api:
  create_session_enabled: false
  user_import_usage:
    enabled: false
  user_export_usage:
    enabled: false
test_mode:
  fixed_oob_otp:
    enabled: false
    code: ""
  deterministic_link_otp:
    enabled: false
  sms:
    suppressed: false
  email:
    suppressed: false
  whatsapp:
    suppressed: false
fraud_protection:
  is_modifiable: false
```

---

## 2. Field reference by section

### `identity`

| Field | Default | Description | Overlaps with `authgear.yaml`? | Redundant? | Layer merge behavior |
|---|---|---|---|---|---|
| `login_id.types.phone.disabled` | `false` | Plan gate: whether the `phone` login ID type can be used at all. | **Yes** — `identity.login_id.keys[]` entries where `type: phone` (`LoginIDKeyConfig` in `identity.go`; there is no `identity.login_id.types.phone` in app config — `LoginIDTypesConfig` only has `email`/`username` sub-objects, phone is configured purely via a `keys[]` entry). Feature flag gates availability; app config's `keys[]` entry configures behavior (max length, create/update/delete disabled, etc.). | **No** — different axis (can it be used at all vs. how it behaves); removing either changes behavior. | **Whole-section replace.** If any layer sets `identity:` at all, its entire `identity` tree (login_id + oauth + biometric together) replaces the lower layer's — not merged field-by-field. Highest layer that mentions `identity` wins outright. |
| `oauth.maximum_providers` | `99` | Max number of distinct OAuth/SSO provider *types* (google, facebook, etc.) that may be configured. | **Yes** — `identity.oauth.providers[]` (`OAuthSSOConfig.Providers`, `identity.go`) is where providers are actually configured; this is a count cap on that list. | **No** — a count cap over a list isn't the same data as the list itself. | Same whole-`identity`-section replace as above. |
| `oauth.providers.<name>.disabled` (9 providers) | `false` each | Per-provider plan gate (e.g. disable `apple` SSO on a lower tier). | **Yes** — `identity.oauth.providers[]` entries (`OAuthSSOConfig.Providers`, `identity.go`) with matching `type`/`alias`; this disables a specific provider type regardless of what's configured there. | **No** — plan-tier kill-switch, independent of whatever the tenant configures. | Same whole-`identity`-section replace as above. |
| `biometric.disabled` | `false` | Plan gate for biometric identity/login. | Partial — biometric-related app config exists per platform client config, not a single toggle. | **No** — no single equivalent field exists to be redundant with. | Same whole-`identity`-section replace as above. |

### `authentication`

| Field | Default | Description | Overlaps? | Redundant? | Layer merge behavior |
|---|---|---|---|---|---|
| `secondary_authenticators.oob_otp_sms.disabled` | `false` | Plan gate: whether SMS OTP can be used as a 2FA method. | **Yes** — `authentication.secondary_authenticators` (`AuthenticationConfig.SecondaryAuthenticators`, `authentication.go`) is the array where the tenant lists `oob_otp_sms` as an enabled 2FA option; this is the plan-level ceiling. | **No** — plan ceiling above the tenant's own on/off choice; dropping it would let any tenant self-enable SMS 2FA regardless of plan. | **Whole-section replace** — if any layer sets `authentication:`, its whole tree wins outright over lower layers (no per-field inheritance). |

### `authenticator`

| Field | Default | Description | Overlaps? | Redundant? | Layer merge behavior |
|---|---|---|---|---|---|
| `password.policy.minimum_guessable_level.disabled` | `false` | Plan gate for the "minimum guessable level" (zxcvbn strength) password rule. | **Yes** — `authenticator.password.policy.minimum_guessable_level` (`PasswordPolicyConfig.MinimumGuessableLevel`, `authenticator.go`) is the actual threshold int. | **No** — gate vs. value; the value is meaningless if the plan doesn't allow the rule. | **Whole-section replace** — a layer that sets `authenticator:` at all replaces the whole tree (all 3 password-policy sub-gates together), not per-field. |
| `password.policy.excluded_keywords.disabled` | `false` | Plan gate for the excluded-keywords password rule. | **Yes** — `authenticator.password.policy.excluded_keywords` (`PasswordPolicyConfig.ExcludedKeywords`, `authenticator.go`) is the actual keyword list. | **No** — same pattern as above. | Same whole-`authenticator`-section replace as above. |
| `password.policy.history.disabled` | `false` | Plan gate for password-history reuse prevention. | **Yes** — `authenticator.password.policy.history_size` / `.history_days` (`PasswordPolicyConfig.HistorySize`/`HistoryDays`, `authenticator.go`) are the actual values. | **No** — same pattern as above. | Same whole-`authenticator`-section replace as above. |

> Note: the schema (`feature_authenticator.go`) only declares these 3 sub-fields. See the appendix — the
> testdata fixture lists 5 more (`min_length`, `uppercase_required`, `lowercase_required`,
> `digit_required`, `symbol_required`) that don't exist in the current Go struct/schema at all.

### `ui`

| Field | Default | Description | Overlaps? | Redundant? | Layer merge behavior |
|---|---|---|---|---|---|
| `white_labeling.disabled` | `false` | Plan gate: whether "Powered by Authgear" branding can be removed. | No direct app config equivalent (this is a pure plan feature). | **No** — nothing to be redundant with. | **Whole-section replace** — a layer that sets `ui:` at all replaces the whole tree (`white_labeling` + `phone_input` together). |
| `phone_input.allowlist` | unset | Restricts which countries' dialing codes are selectable in phone inputs. | No app config equivalent found. | **No** — nothing to be redundant with. | Same whole-`ui`-section replace as above. |

### `custom_domain`

| Field | Default | Description | Overlaps? | Redundant? | Layer merge behavior |
|---|---|---|---|---|---|
| `disabled` | `false` | Plan gate: whether custom domains can be attached to the project. | No — custom domains are managed out-of-band (portal/DNS), not via `authgear.yaml`. | **No**. | **Whole-section replace** (trivial here — the section is a single field). |

### `oauth` (client)

| Field | Default | Description | Overlaps? | Redundant? | Layer merge behavior |
|---|---|---|---|---|---|
| `client.maximum` | `99` | Max number of OAuth clients (apps) the project may define. | **Yes** — `oauth.clients[]` (`OAuthConfig.Clients`, `oauth.go`, the actual client list) is app config; this is the count cap on `len(clients)`. | **No** — cap vs. list, not the same data. | **Field-level merge** — `OAuthFeatureConfig.Merge` delegates to `OAuthClientFeatureConfig.Merge`, which independently overrides `maximum`/`soft_maximum`/`custom_ui_enabled`/`app2app_enabled` only where a higher layer sets that specific sub-field. A Plan layer setting only `custom_ui_enabled` will **not** clobber an App-layer `maximum`. |
| `client.soft_maximum` | `99` | Soft warning threshold before hitting `maximum` (portal UI warning). | Same as above — `oauth.clients[]`. | **No** — distinct from `maximum` too (warning threshold vs. hard cap). | Same field-level merge as `client.maximum`. |
| `client.custom_ui_enabled` | `false` | Plan gate for the custom/branded auth UI feature per client. | **Yes** — `oauth.clients[].x_custom_ui_uri` (`OAuthClientConfig.CustomUIURI`, `oauth.go`) is the per-client URI this gates; if the feature is disabled, a configured `x_custom_ui_uri` is not honored. | **No** — gate vs. per-client configuration. | Same field-level merge as `client.maximum`. |
| `client.app2app_enabled` | `false` | Plan gate for app2app SSO. | **Yes** — `oauth.clients[].x_app2app_enabled` (`OAuthClientConfig.App2appEnabled`, `oauth.go`) is the per-client toggle this gates. | **No** — gate vs. per-client configuration. | Same field-level merge as `client.maximum`. |

### `hook`

| Field | Default | Description | Overlaps? | Redundant? | Layer merge behavior |
|---|---|---|---|---|---|
| `blocking_handler.maximum` | `99` | Max number of blocking webhook handlers. | **Yes** — `hook.blocking_handlers[]` (`HookConfig.BlockingHandlers`, `hook.go`) is the actual handler list; this is the count cap. | **No** — cap vs. list. | **Whole-section replace** — a layer setting `hook:` at all replaces both `blocking_handler` and `non_blocking_handler` together. |
| `non_blocking_handler.maximum` | `99` | Max number of non-blocking (event) webhook handlers. | **Yes** — `hook.non_blocking_handlers[]` (`HookConfig.NonBlockingHandlers`, `hook.go`), same relationship. | **No** — cap vs. list. | Same whole-`hook`-section replace as above. |

### `audit_log`

| Field | Default | Description | Overlaps? | Redundant? | Layer merge behavior |
|---|---|---|---|---|---|
| `retrieval_days` | `-1` (no limit) | How many days of audit log history are retrievable via API/portal. | No app config equivalent — audit log retention isn't tenant-configurable. | **No**. | **Whole-section replace** (trivial — single field). |

### `google_tag_manager`

| Field | Default | Description | Overlaps? | Redundant? | Layer merge behavior |
|---|---|---|---|---|---|
| `disabled` | `false` | Plan gate for the Google Tag Manager integration. | **Yes** — `google_tag_manager.container_id` (`GoogleTagManagerConfig.ContainerID`, `google_tag_manager.go`) is the actual container ID; this gates whether it can be used at all. | **No** — gate vs. the container ID value itself. | **Whole-section replace** (trivial — single field). |

### `rate_limits` (top-level)

| Field | Default | Description | Overlaps? | Redundant? | Layer merge behavior |
|---|---|---|---|---|---|
| `disabled` | `false` | **Global kill-switch**: when `true`, `Limiter.doReserveN` (`pkg/lib/ratelimit/limiter.go:77`) skips *all* rate limiting for the project, regardless of any other rate-limit config. | No single equivalent in app config — app config only has many individual per-endpoint `enabled` flags (e.g. `authentication_rate_limits.go`, `messaging.go`). This is a blanket override above all of them. | **No** — no single field it could be redundant with; it's the master switch above all of them. | **Whole-section replace** (trivial — single field). |

### `messaging`

| Field | Default | Description | Overlaps? | Redundant? | Layer merge behavior |
|---|---|---|---|---|---|
| `rate_limits.sms` / `sms_per_ip` / `sms_per_target` / `email` / `email_per_ip` / `email_per_target` | see YAML above | Per-channel rate-limit **ceilings**. Confirmed in `pkg/lib/ratelimit/ratelimits.go`: both this and the app-config equivalent are read, and whichever has the lower `Rate()` (burst/period) wins. | **Yes, exact schema duplicate** — `messaging.rate_limits.sms` / `.sms_per_ip` / `.sms_per_target` / `.email` / `.email_per_ip` / `.email_per_target` (`MessagingRateLimitsConfig`, `messaging.go`), identical field names and identical default values, at the identical path. Both are live and both matter. | **No, despite identical schema** — both are actually read and enforced (the stricter wins). Removing the feature copy removes plan enforcement (a tenant could set an unlimited app-config value); removing the app copy removes tenant self-service tuning below the plan ceiling. The *schema* is duplicated but the *behavior* isn't. | **Field-level at the `messaging` level, but whole-object one level down.** `MessagingFeatureConfig.Merge` independently overrides `rate_limits` only if a higher layer sets it — but once a layer sets `rate_limits:`, the *entire* `sms`/`sms_per_ip`/`sms_per_target`/`email`/`email_per_ip`/`email_per_target` bundle from that layer replaces the lower layer's bundle wholesale (no merging within `rate_limits` itself). |
| `sms_usage` / `email_usage` / `whatsapp_usage` (`enabled`, `period`, `quota`) | `enabled: false` | **Deprecated** single-limit usage config, kept only for backward compatibility. `FeatureConfig.Migrate()` converts any of these (if enabled) into the unified `usage.limits.*` structure at parse time. | Partial — app config has no usage-quota concept (usage/billing limits are inherently plan-level, not tenant-configurable), so no `authgear.yaml` overlap; but these 3 fields are themselves superseded by **`usage.limits.sms` / `.email` / `.whatsapp`** in the same `authgear.features.yaml` file. | **Yes — redundant with `usage.limits.sms/email/whatsapp`** (same file, same `FeatureConfig`). Fully superseded; kept only so already-issued feature configs keep working. Should not appear in new `authgear.features.yaml` files or templates. | **Field-level** — `sms_usage`/`email_usage`/`whatsapp_usage` are each independently overridden by whichever layer sets that specific one; `Migrate()` then runs once, after all 4 layers are merged and defaulted. |
| `sms_usage_count_disabled` / `whatsapp_usage_count_disabled` | `false` | **Not a limit** — confirmed in `pkg/lib/messaging/sender.go`: passed as `IsNotCountedInUsage` into the message sender. It tells the sender "don't count this send toward the usage/billing metric" (e.g. for portal-triggered test sends). | No `authgear.yaml` equivalent — purely an internal billing-accuracy flag. | **No** — unique concept (billing exemption), not overlapping with any limit field despite the similar name. | **Field-level** — independently overridden per field, same as above. |
| `custom_sms_provider_disabled` | `false` | Plan gate: whether a custom SMS provider (vs. Authgear's default) can be configured. | **Partial, and not in `authgear.yaml`** — the app-config side that selects a provider is `messaging.sms_gateway.provider` (`SMSGatewayConfig.Provider`, `messaging_sms_gateway.go`, `authgear.yaml`), but the actual custom-provider **credentials** (`CustomSMSProviderConfig`) live in `authgear.secrets.yaml`, not `authgear.yaml` — confirmed by `SecretConfigSchema.Add` in `secret_custom_sms_provider.go`. | **No** — gate vs. a value split across two different files; the credentials are inert without the gate. | **Field-level** — independently overridden per field, same as above. |
| `custom_smtp_disabled` | `false` | Plan gate: whether custom SMTP (vs. default mailer) can be configured. Enforced at `pkg/lib/config/configsource/resources.go:628`, which checks for an `SMTPServerCredentials` secret. | **No `authgear.yaml` field** — the SMTP server credentials (`SMTPServerCredentials`, `secret_data.go`) live entirely in `authgear.secrets.yaml`, confirmed by `SecretConfigSchema.Add`; there is nothing named `smtp` in `authgear.yaml` itself. | **No** — same gate-vs-secret-credentials pattern, just in a different file than expected. | **Field-level** — independently overridden per field, same as above. |
| `template_customization_disabled` | `false` | Plan gate: whether email/SMS message templates can be customized. Enforced in `pkg/util/template/resource.go` and `translation.go`. | **Not a single field** — gates edits to the bundled resource files (`resources/authgear/templates/*.gotemplate`, `translation.json`), which aren't expressed as `authgear.yaml` fields at all. | **No** — gate vs. content, not vs. a config field. | **Field-level** — independently overridden per field, same as above. |

### `usage` (unified usage limits — the modern replacement for `messaging.*_usage` and `admin_api.*_usage`)

| Field | Default | Description | Overlaps? | Redundant? | Layer merge behavior |
|---|---|---|---|---|---|
| `hooks[].url` / `.match` | unset | Webhook(s) fired when a usage threshold is crossed, filtered by `match` (`*`, `sms`, `email`, `whatsapp`, `user_import`, `user_export`). | No app config equivalent (hooks in app config are auth-flow hooks, not usage hooks). | **No**. | **Appended, not replaced** — the only field in the whole schema that behaves this way. `FeatureUsageConfig.Merge` does `merged.Hooks = append(merged.Hooks, layer.Usage.Hooks...)`, so Cluster + Plan + App usage-hooks **all accumulate** into the effective config rather than a higher layer replacing a lower one. |
| `limits.sms` / `.email` / `.whatsapp` / `.user_import` / `.user_export` (arrays of `{quota, period, action}`) | unset | Structured usage quotas per resource, with `action: alert\|block` on breach. This is the unified, current mechanism; `messaging.sms_usage` etc. and `admin_api.user_import_usage`/`user_export_usage` are deprecated predecessors auto-migrated into this on parse. | No app config equivalent — usage/billing quotas are inherently plan-level. | **No** — this is the canonical, non-deprecated field; the deprecated fields are redundant *with this*, not the other way round. | **Field-level, per usage name** — `mergeFeatureUsageLimits` independently overrides `sms`/`email`/`whatsapp`/`user_import`/`user_export` only where a higher layer sets that specific array (the whole array for that name replaces, it does not append entries within one name). |

### `collaborator`

| Field | Default | Description | Overlaps? | Redundant? | Layer merge behavior |
|---|---|---|---|---|---|
| `maximum` / `soft_maximum` | unset (nil) | Max number of portal collaborators (admins) on the project. | No — collaborators are managed via the portal/site-admin API, not `authgear.yaml`. | **No**. | **Whole-section replace** — a layer setting `collaborator:` at all replaces both `maximum` and `soft_maximum` together. |

### `web3` *(deprecated)*

| Field | Default | Description | Overlaps? | Redundant? | Layer merge behavior |
|---|---|---|---|---|---|
| `nft.maximum` | `3` | Legacy Web3/NFT-gated-login feature cap. Struct is named `Deprecated_Web3FeatureConfig`. | No — Web3 identity is being phased out; no live app config path uses this. | **Effectively yes — obsolete**, not because of app-config overlap but because the whole Web3/NFT identity feature is being phased out. Candidate for removal independent of the app-config question. | **Never actually merges — confirmed gap, not just deprecation.** `Deprecated_Web3FeatureConfig` is the *only* section in `feature_*.go` with no `Merge` method and no `MergeableFeatureConfig` assertion. Since `FeatureConfig.Merge`'s reflection loop skips fields that don't implement the interface, **no Cluster/Plan/App layer can ever change `web3.nft.maximum`** — it always resolves to the hardcoded default `3`, regardless of what any layer's YAML sets. |

### `admin_api`

| Field | Default | Description | Overlaps? | Redundant? | Layer merge behavior |
|---|---|---|---|---|---|
| `create_session_enabled` | `false` | Plan gate for the Admin API `createSession` GraphQL mutation (issuing an authenticated session for a user directly via Admin API, e.g. impersonation/backend-issued login). Enforced at `pkg/admin/graphql/session_mutation.go:202`. | **No app config equivalent exists** — `admin.go` only defines `AdminAPIAuth` (`none`/`jwt`, i.e. *how* the Admin API authenticates its caller), nothing about *which mutations* are permitted. | **No** — nothing to be redundant with, and deliberately so — whether the Admin API can mint sessions is a platform/trust decision, not something a tenant should self-enable. | **Field-level** — `AdminAPIFeatureConfig.Merge` independently overrides `create_session_enabled`/`user_import_usage`/`user_export_usage`, only where a higher layer sets that specific one. |
| `user_import_usage` / `user_export_usage` (`enabled`, `period`, `quota`) | `enabled: false` | Deprecated single-limit quotas for user import/export API calls; migrated into `usage.limits.user_import`/`user_export` on parse (same mechanism as `messaging.*_usage`). | No — superseded by `usage.limits`. | **Yes — redundant with `usage.limits.user_import`/`user_export`**. Same situation as `messaging.sms_usage` etc.: kept for backward compatibility only. | Same field-level merge as `create_session_enabled`. |

### `test_mode`

| Field | Default | Description | Overlaps? | Redundant? | Layer merge behavior |
|---|---|---|---|---|---|
| `fixed_oob_otp.enabled` / `.code` | `false` / `""` | Plan-tier gate + fixed code. When both this **and** the app-level `test_mode.oob_otp.enabled` + a matching rule are true, the fixed `code` is used instead of a random OTP (`pkg/lib/authn/otp/form.go`). | **Yes, layered** — `test_mode.oob_otp.enabled` + `test_mode.oob_otp.rules[].fixed_code` (`TestModeOOBOTPConfig`, `test_mode.go`) decide *which phone/email targets* get a fixed code via regex rules; this feature flag is the plan-level switch that must also be on. Neither alone is sufficient. | **No** — both must be true simultaneously (AND, not OR); dropping either one disables the capability entirely. | **Whole-section replace, and it's a real trap here.** If any layer sets `test_mode:` at all, its entire tree (`fixed_oob_otp` + `deterministic_link_otp` + `sms`/`email`/`whatsapp.suppressed`, all 5) replaces the lower layer's wholesale. E.g. if Plan sets `test_mode.fixed_oob_otp.enabled: true` and App later sets only `test_mode.sms.suppressed: true` without repeating `fixed_oob_otp`, the App layer's `test_mode` wins outright and the Plan's `fixed_oob_otp` setting is silently lost. |
| `deterministic_link_otp.enabled` | `false` | Plan gate for deterministic (non-random) magic-link OTP generation, same gating pattern as above. | Same layered pattern, jointly with **`test_mode.oob_otp`** (`test_mode.go`) — no separate app-config field of its own. | **No** — same AND-gate reasoning. | Same whole-`test_mode`-section replace and trap as above. |
| `sms.suppressed` | `false` | **Blanket** plan-tier kill-switch: when `true`, ALL outgoing SMS are suppressed (routed to test-mode delivery), regardless of recipient. Enforced in `pkg/lib/messaging/sender.go` as an independent, unconditional check. | **Yes, but not a duplicate** — `test_mode.sms.enabled` + `test_mode.sms.rules[].suppressed` (`TestModeSMSConfig`, `test_mode.go`) suppress SMS **only for recipients matching a configured regex rule**. The feature flag suppresses **everyone, unconditionally**. Both are checked; either one suppresses (OR, not override). | **No** — they answer different questions ("suppress everyone" vs. "suppress these specific test numbers"). Removing the feature flag would remove the ability to force a trial plan into a fully sandboxed state; removing the app-config rules would remove tenant-defined test-number suppression. Confusing naming, not redundant. | Same whole-`test_mode`-section replace and trap as above. |
| `email.suppressed` | `false` | Same blanket kill-switch pattern, for email. | **`test_mode.email.enabled` + `.rules[].suppressed`** (`TestModeEmailConfig`, `test_mode.go`) — same layered relationship as `sms.suppressed`. | **No** — same reasoning as `sms.suppressed`. | Same whole-`test_mode`-section replace and trap as above. |
| `whatsapp.suppressed` | `false` | Same blanket kill-switch pattern, for WhatsApp. | **`test_mode.whatsapp.enabled` + `.rules[].suppressed`** (`TestModeWhatsappConfig`, `test_mode.go`) — same layered relationship as `sms.suppressed`. | **No** — same reasoning as `sms.suppressed`. | Same whole-`test_mode`-section replace and trap as above. |

### `fraud_protection`

| Field | Default | Description | Overlaps? | Redundant? | Layer merge behavior |
|---|---|---|---|---|---|
| `is_modifiable` | `false` | Plan gate: whether the tenant is allowed to modify fraud protection settings themselves (vs. Authgear-managed defaults). | **Yes** — the whole **`fraud_protection`** object (`FraudProtectionConfig`, `fraud_protection.go`: `enabled`, `sms.unverified_otp_budget`, `warnings[]`, `decision`) is the app-config content this gates edit-ability of. | **No** — gate vs. the rules it gates access to. | **Whole-section replace** (trivial here — the section is a single field). |

---

## Appendix: stale test fixture data found

While cross-checking `testdata/default_feature.yaml` against the live Go schema, two blocks were found
that **do not correspond to any field in the current `FeatureConfig` struct** (`feature.go` and its
sub-files), yet still appear in the fixture and pass its test:

1. A top-level `rate_limit:` (singular) block:
   ```yaml
   rate_limit:
     disabled: false
     sms:
       size: 100000
       reset_period: 2592000
   ```
   No struct anywhere in `pkg/lib/config` declares a JSON field `rate_limit` (only `rate_limits`,
   plural, exists — `RateLimitsFeatureConfig`, a single `disabled` bool). Git history shows this predates
   `31f8a21815 Move rate limit disabled config to feature config` and `aa8cf2e667 Add messaging rate
   limits feature config` — it looks like a leftover from an earlier SMS-quota design.

2. Five extra sub-fields under `authenticator.password.policy` — `min_length`, `uppercase_required`,
   `lowercase_required`, `digit_required`, `symbol_required` — that aren't declared in
   `feature_authenticator.go`'s `PasswordPolicyFeatureConfig` (which only has `minimum_guessable_level`,
   `excluded_keywords`, `history`).

**Why this doesn't break anything today:** the test at `pkg/lib/config/feature_test.go:29` compares
against this fixture using `yaml.Unmarshal(data, &defaultCfg)` directly (not the schema-validating
`ParseFeatureConfig` path), and Go's YAML/JSON unmarshaling silently drops unknown fields — so the test
still passes even though these keys are dead. If a real `authgear.features.yaml` tried to set these
fields, `ParseFeatureConfigWithoutDefaults`'s schema validation (`additionalProperties: false`) would
**reject** it. This fixture should be regenerated/trimmed so it doesn't misrepresent the current schema to
future readers.
