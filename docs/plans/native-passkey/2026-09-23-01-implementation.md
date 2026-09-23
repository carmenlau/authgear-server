# Native Passkey Implementation Plan

Companion to the spec, which defines the behaviour this plan implements. This document covers only *how*; anything a customer can observe belongs in the spec, not here.

Repositories: `authgear-server`, `authgear-sdk-ios`, `authgear-sdk-android`.

## Context

Passkey already works end to end on the web. The ceremony, the credential model, the authflow steps, and the account-management service calls all exist. Native adds four things and changes one:

| | |
|---|---|
| **Blocking** | The relying party origin is a single value derived from the request `Host`. Android sends a non-URL origin, so every Android ceremony fails verification until this is a list |
| **New** | Relying party ID as explicit configuration, rather than derived per request |
| **New** | Native app registration, and serving the two association files |
| **New** | An HTTP surface for passkey add/list/delete (the service layer already exists) |
| **Undecided** | How a native client performs a *sign-in* ceremony — see [Part 4](#part-4-server--native-sign-in-transport-undecided) |

Everything except Part 4 is independent of that decision and can proceed now.

### What already exists

| Capability | Location | Notes |
|---|---|---|
| Ceremony verification | `pkg/lib/feature/passkey/service.go` | Single origin only |
| Creation / request options | `pkg/lib/feature/passkey/{creation,request}_options_service.go` | Complete |
| Challenge session, keyed by challenge | `pkg/lib/feature/passkey/{session,store}.go` | Redis; transport-agnostic |
| Passkey as Identity + Primary Authenticator | `pkg/lib/authn/identity/passkey/`, `pkg/lib/authn/authenticator/passkey/` | Complete |
| Authflow nodes and inputs | `pkg/lib/authenticationflow/declarative/*passkey*.go` | Complete |
| Passkey over the Authentication Flow API | `create_passkey_data.creation_options`; `authentication: primary_passkey` → `request_options` | Complete for web |
| Add / delete passkey service calls | `pkg/lib/accountmanagement/service_identity.go:947`, `:1019` | **In production use** by `pkg/auth/handler/webapp/authflowv2/settings_passkey.go:141,163` |
| List passkeys | `Identities.Passkey.List` | Same |
| AuthUI-only options endpoints | `POST /_internals/passkey/{creation,request}_options` | Bound to the webapp session; not reusable by SDKs |
| Association files | — | Not served. Only `/.well-known/openid-configuration` and `/.well-known/oauth-authorization-server` exist |

Note that passkey add/delete does **not** use the `StartAdding`/`FinishAdding` token pair — that is the OAuth-identity path. Passkey is one-shot, because the challenge session already binds the two round trips. Do not add a token.

---

## Part 1: Server — relying party foundations

### 1.1 Multi-origin verification

`pkg/lib/feature/passkey/config.go` holds `RPOrigin string`. It becomes a list. Both verification call sites (`service.go:52`, `service.go:163`) construct `rpOrigins := []string{config.RPOrigin}` and must take the list instead.

Populate with:

- the web origin, `https://<relying_party_id>`, and
- one entry per registered Android signing certificate, formed as `android:apk-key-hash:<value>` where the value is the certificate's SHA-256 fingerprint with colons stripped, hex-decoded, then base64url-encoded without padding.

`rpTopOrigins` currently aliases `rpOrigins` under `protocol.TopOriginExplicitVerificationMode`. Keep the aliasing.

> **Risk, resolve first.** `go-webauthn` may attempt to parse each expected origin as a URL, in which case `android:apk-key-hash:…` will not round-trip and top-origin verification will misbehave. Write a test with a captured Android `clientDataJSON` before building anything on top of this. If the library rejects the form, the fallback is to verify Android origins ourselves before delegating.

### 1.2 Relying party ID as configuration

`pkg/lib/feature/passkey/config_service.go` builds the origin from `httputil.GetProto`/`GetHost` and sets `RPID: origin.Hostname()`. Replace with configuration, defaulting to the hostname of `http.public_origin` (`pkg/lib/config/http.go:25`).

This default is exactly backward compatible, not merely sensible. `PublicOriginMiddleware` (`pkg/auth/webapp/public_origin_middleware.go`) 307-redirects any request whose scheme and host do not match `http.public_origin`, and it is on every chain a ceremony can reach — `apiChain` (`pkg/auth/routes.go:129`), which `authenticationFlowChain` and `accountManagementChain` inherit; `webappChain` (`:173`); and `newOAuthAPIChain` (`:112`). A passkey handler therefore only ever runs with the public-origin host, so the derived value is already the public-origin hostname for every credential in existence.

The real exposure is sequential, not concurrent: `http.public_origin` changes over a project's life — a project moving to a custom domain, or `deleteDomainUpdatePublicOrigin` (`pkg/portal/graphql/domain_mutation.go:220`) rewriting it when the matching domain is deleted. Each change invalidates the passkeys enrolled under the previous hostname, today, with no record of what that hostname was. See [1.4](#14-the-relying-party-id-is-already-recorded).

### 1.3 Config schema

Add `Passkey *PasskeyConfig` to `IdentityConfig` (`pkg/lib/config/identity.go:24`), beside `LDAP`, `LoginID`, `OAuth` and `Biometric`. `BiometricConfig` (`identity.go:474`) is the nearest model for the Go struct plus JSON Schema pair, and the reason for this placement is that no identity type has a top-level block.

```yaml
identity:
  passkey:
    relying_party_id: example.com   # defaults to the host of http.public_origin
    native_apps: [...]
```

### 1.4 The relying party ID is already recorded

No schema change is needed. `_auth_identity_passkey.creation_options` and `_auth_authenticator_passkey.creation_options` store the full `model.WebAuthnCreationOptions` as JSON, which carries `rp.id` (`pkg/api/model/webauthn_creation_options.go:15,27`). Both stores select and unmarshal it (`pkg/lib/authn/identity/passkey/store.go:31`, `pkg/lib/authn/authenticator/passkey/store.go:32`).

So the relying party ID every credential was enrolled under is available retroactively, for passkeys created before this work. Reporting "how many users hold passkeys that a domain change will invalidate" is a query, not a migration.

If that report becomes a frequent or latency-sensitive query, consider a generated column or an index over the JSON path. Neither is a prerequisite.

---

## Part 2: Server — native app registration and association files

### 2.1 Config schema

`native_apps` lives in `identity.passkey`, not on the OAuth client. The association files are project-level artefacts — one file per domain, served at a fixed path with no client dimension — and `pkg/lib/feature/passkey` has no client context to scope them by (`ConfigService` takes only the request, trust-proxy flag and translation service). Placing them on the client would mean aggregating every client to build one file, and threading client identity into a package that has never needed it.

Schema goes with the rest of `identity.passkey` from [1.3](#13-config-schema). The list-with-`platform`-discriminator form needs a schema that varies by discriminator.

Validation worth enforcing at config load, because each failure is silent at runtime:

| Rule | Why |
|---|---|
| `relying_party_id` is a verified domain of the project, or the apex of one | the only ownership proof available. `DomainService.verifyDomain` (`pkg/portal/service/domain.go:299`) resolves the TXT record on `ApexDomain`, so verifying `auth.example.com` already proves control of `example.com`. A default domain records itself as its apex (`domain.go`, `CreateDomain`, the `!isCustom` branch), so `authgearapps.com` can never be reached this way and needs no blocklist |
| iOS team ID and bundle ID both present when `platform: ios`; package name and at least one fingerprint when `platform: android` | a half-filled entry produces an association file that verifies nothing |
| Fingerprints are 32 colon-separated hex octets | a malformed fingerprint yields an origin no device will ever match |
| Fingerprints unique within a client | duplicates silently bloat the file |

### 2.2 Serve the association files

Two unauthenticated, cacheable routes, registered where `pkg/auth/handler/oauth/metadata.go` registers the OIDC metadata routes:

| Route | Content type | Body |
|---|---|---|
| `GET /.well-known/apple-app-site-association` | `application/json` | `webcredentials.apps`, one `<team_id>.<bundle_id>` per iOS entry across all native clients |
| `GET /.well-known/assetlinks.json` | `application/json` | one statement per Android entry, `relation: ["delegate_permission/common.get_login_creds"]`, `target.namespace: android_app` |

Must be served without redirect. The Apple file must have no `.json` extension.

> **Conflict to resolve.** `docs/specs/app2app.md:141-167` assumes the *customer* hosts `assetlinks.json` and uses `delegate_permission/common.handle_all_urls`. If a project uses both features, one file must carry both relations. Decide whether Authgear's generated file includes app2app's relation, or whether the two features are documented as mutually exclusive on one domain. This arises only when the relying party ID is a domain that also handles an app's links; the spec covers the merge rule for customer-hosted files, but not what Authgear generates. Tracked here as Q-IMPL-4.

---

## Part 3: Server — enrolment HTTP API

The service layer exists and is exercised by the web settings page. This part is an HTTP surface over it, following `pkg/auth/handler/api/accountmanagement_v1_identification.go`.

| Operation | Existing call |
|---|---|
| list | `Identities.Passkey.List(ctx, userID)` |
| creation options | `PasskeyCreationOptionsService.MakeCreationOptions(ctx, userID)` |
| add | `accountmanagement.Service.AddPasskey` |
| delete | `accountmanagement.Service.DeletePasskey` |

Requires an authenticated session, cookie or bearer, as the rest of the Account Management API does. Responses use the `result`/`error` envelope.

Smallest and least risky part of the work. Shippable before any SDK exists, and usable by custom web UIs immediately.

---

## Part 4: Server — native sign-in transport (UNDECIDED)

**This is the one open server decision, and nothing else in this plan depends on it.**

The problem: enrolment happens against an authenticated session, so it needs no flow engine. Sign-in must run identity and authenticator logic and issue a session, which today is only reachable through a flow engine.

Two facts constrain the answer:

1. Every existing grant that *authenticates* a user — anonymous, biometric — runs on the legacy `interaction.Graph`. `h.Graphs.DryRun` appears exactly three times in `pkg/lib/oauth/handler/handler_token.go` (1072, 1270, 1333), and those are the three. Grants that merely re-key an existing session (app2app, native SSO, pre-authenticated URL, id-token) use no engine, which is why their OAuth-layer design does not license the same shortcut here.
2. `grep authenticationflow pkg/lib/oauth/handler/` returns nothing. The token endpoint has no path to the authflow engine today.

So "add a grant type" and "do not extend the legacy graph" are in conflict, and the conflict must be resolved before implementation, not during.

### Options

| | A. Authflow + native redemption | B. New grant type | C. Dedicated endpoints + grant type |
|---|---|---|---|
| Ceremony payload schemas | reuse existing | re-declare | re-declare |
| Engine | authflow | legacy graph, or unproven authflow-from-token-handler | same |
| Honours flow config, 2FA, bot protection, lockout | yes | no | no |
| New server surface | a way to redeem a finished flow for tokens without a browser | grant + ceremony reimplementation | 2 endpoints + grant + reimplementation |
| SDK work | moderate — a minimal authflow client | small | small |
| Extends the legacy graph | no | almost certainly | yes |

Groundwork for option A already exists: flow creation without an OAuth session is supported (`pkg/auth/handler/api/authenticationflow_v1_create.go:129` guards the binding behind `if … ok`), a finished flow yields an `authenticationinfo.Entry` (`pkg/lib/authenticationflow/service.go:640,646`), and the authorize endpoint already redeems such an entry for a code (`pkg/lib/oauth/handler/handler_authz.go:463`). What is missing is a native path to that redemption.

> **Q-IMPL-1.** Which option? Recommended: A, on the grounds that it reuses validated schemas, keeps passkey off the legacy graph, and inherits flow policy for free.
>
> **Spike required before committing.** Confirm that a login flow created with no `url_query` reaches the `primary_passkey` branch, and size the work to redeem a finished flow's authentication info for tokens without a browser round trip. If that proves substantially harder than it reads, B or C return — but then the "no new legacy-graph code" constraint must be relaxed explicitly.

> **Q-IMPL-2.** Should native passkey sign-in be restricted to clients with full access scope, as biometric is (`handler_token.go:1224`)? This is spec question Q6 seen from the server side.

---

## Part 5: SDK — the shared shape

Both SDKs perform the same five steps. Only step 3 differs by platform.

```
1. ask the server for options          (creation options, or request options)
2. hand the options to the platform    ← the only platform-specific step
3. platform shows the system sheet, user approves with biometric
4. platform returns a credential
5. send the credential to the server   (add, or sign in)
```

Step 2 is where the platforms diverge, and it is the single largest asymmetry in the work:

| | iOS | Android |
|---|---|---|
| Input | discrete fields — the SDK must decode `challenge` and `user.id` from base64url into bytes | the options JSON string, verbatim |
| Output | typed objects with separate `rawClientDataJSON`, `rawAttestationObject`, `signature`, `userID` fields | `registrationResponseJson` / `authenticationResponseJson`, already in WebAuthn JSON |
| SDK work | decode inbound, re-assemble outbound JSON | pass through both ways |

So Android is close to a pipe, and iOS needs a real bridging layer. Build Android first: it validates the server with far less client code, and any server bug it surfaces would otherwise be indistinguishable from an iOS bridging bug.

---

## Part 6: iOS SDK

### 6.1 Layering

Follow the existing shape. The public method lives on `Authgear` (`Sources/Authgear.swift`), network calls go through `AuthAPIClient` (`Sources/APIClient.swift`), and the ceremony lives in a new `Sources/passkey/` directory mirroring `Sources/app2app/`.

### 6.2 Threading — the one non-obvious constraint

The existing pattern is `withMainQueueHandler(handler)` then `workerQueue.async { … }`, with synchronous API-client calls inside, exactly as `authenticateBiometric` does (`Sources/Authgear.swift:1655`).

Passkey cannot follow that pattern unchanged, because `ASAuthorizationController` is delegate-based and must be driven from the main thread. The flow has to hop:

```
caller (main)
  └─ workerQueue.async
       └─ syncRequest…Options()                    network
            └─ DispatchQueue.main.async
                 └─ ASAuthorizationController.performRequests()
                      └─ delegate callback (main)
                           └─ workerQueue.async
                                └─ syncRequest…(credential)   network
                                     └─ handler on main
```

Two consequences:

- The controller, its delegate and its continuation must be retained for the lifetime of the ceremony. A local `ASAuthorizationController` is deallocated before the delegate fires and the ceremony silently never completes. Hold it on the ceremony object.
- Only one ceremony may be in flight. A second concurrent call should fail rather than interleave.

`ASAuthorizationController` also needs a `presentationContextProvider`; reuse the window-finding logic in `ASWebAuthenticationSessionUIImplementation.swift` rather than writing a second copy.

### 6.3 Bridging

Inbound, per ceremony:

| From options JSON | To |
|---|---|
| `challenge` (base64url string) | `Data` |
| `user.id` (base64url string) | `Data`, as `userID` |
| `user.name` | `name` |
| `rp.id` | `relyingPartyIdentifier` on the provider |

Outbound, assembled by the SDK into the WebAuthn JSON the server already accepts:

| Registration | Assertion |
|---|---|
| `id`, `rawId` ← `credentialID` | same |
| `type: "public-key"` | same |
| `response.clientDataJSON` | `response.clientDataJSON` |
| `response.attestationObject` | `response.authenticatorData`, `response.signature`, `response.userHandle` |

`Sources/Base64.swift` already provides the encoding half. The decoding half is new.

> **Verify.** The server's input schemas require base64url without padding (`x_base64_url` format). Confirm the SDK's encoder matches, since a padded value will fail validation with a message that does not mention padding.

### 6.4 Version gating

The package targets iOS 11 (`Package.swift`, `Authgear.podspec:9`) and passkey needs iOS 16. Gate each new method with `@available(iOS 16.0, *)`, exactly as biometric uses `@available(iOS 11.3, *)`. The package minimum does not change, so existing integrators are unaffected.

### 6.5 Errors

Extend `AuthgearError` (`Sources/AuthgearError.swift`):

| Platform condition | Case |
|---|---|
| `ASAuthorizationError.canceled` | `.cancel` — existing |
| `ASAuthorizationError.failed`, `.invalidResponse`, `.notHandled` | new `.passkeyFailed(Error)` |
| No credential available | new `.passkeyNotFound` |
| Associated domain missing or wrong | new `.passkeyDomainNotAssociated` |
| Below iOS 16 | new `.passkeyNotSupported` |

The domain case earns its own value: it is the most likely integration mistake, and the platform reports it indistinguishably from a generic failure, so the SDK must infer it.

---

## Part 7: Android SDK

### 7.1 Layering

`Authgear.kt` exposes the listener-based method and delegates with `scope.launch { core.… }`; the work lives in `AuthgearCore.kt` as a `suspend fun`; the coroutine extension at the bottom of `Authgear.kt` wraps the listener form. This is exactly how `enableBiometric`/`authenticateBiometric` are built (`Authgear.kt:669,695`; `AuthgearCore.kt:922,1015`).

New code goes in `sdk/src/main/java/com/oursky/authgear/passkey/`, alongside `app2app/` and `dpop/`.

### 7.2 Dependencies

```kotlin
implementation("androidx.credentials:credentials:<pin a version>")
implementation("androidx.credentials:credentials-play-services-auth:<same>")
```

> **Not researched.** Pin an exact version, and check what it does to the SDK's minimum supported API level and method count before adding it.

### 7.3 Ceremony

`CredentialManager` offers suspending `createCredential` / `getCredential`, which fit `AuthgearCore`'s existing `suspend fun` style; prefer them to the `…Async` callback forms. Both need the `Activity` carried on the options object, mirroring `BiometricOptions` holding a `FragmentActivity`.

No bridging: pass the options JSON straight into `CreatePublicKeyCredentialRequest` / `GetPublicKeyCredentialOption`, and post `registrationResponseJson` / `authenticationResponseJson` straight back.

### 7.4 Version gating

The library declares no `minSdk`, only `aarMetadata { minCompileSdk = 21 }`; the sample sets 23. Passkey needs API 28, so gate at runtime on `Build.VERSION.SDK_INT` and throw below it, rather than raising the library minimum.

### 7.5 Errors

One exception class per condition, following `BiometricLockoutException` and siblings:

| Credential Manager | Authgear |
|---|---|
| `GetCredentialCancellationException`, `CreateCredentialCancellationException` | `CancelException` — existing |
| `NoCredentialException` | `PasskeyNotFoundException` |
| `CreatePublicKeyCredentialDomException` with a domain error | `PasskeyDomainNotAssociatedException` |
| `CreateCredentialUnknownException`, `GetCredentialUnknownException` | `PasskeyException` |
| API < 28 | `PasskeyNotSupportedException` |

> **Verify.** Confirm which exception Credential Manager actually raises when `assetlinks.json` is missing or the signing certificate is unlisted. It is reported as a DOM-shaped error rather than a dedicated type, and the mapping above is an assumption.

---

## Critical files

| File | Change |
|---|---|
| `pkg/lib/feature/passkey/config.go` | `RPOrigin string` → list |
| `pkg/lib/feature/passkey/config_service.go` | stop deriving relying party ID from the request host |
| `pkg/lib/feature/passkey/service.go:52,163` | verify against the list |
| `pkg/lib/config/identity.go` | add `Passkey` to `IdentityConfig`: `relying_party_id` and `native_apps` |
| `pkg/auth/handler/oauth/metadata.go` | pattern for registering the two new well-known routes |
| `pkg/auth/handler/api/` | new enrolment handler |
| `authgear-sdk-ios/Sources/Authgear.swift`, `APIClient.swift`, new `Sources/passkey/` | iOS |
| `authgear-sdk-android/.../Authgear.kt`, `AuthgearCore.kt`, new `passkey/` | Android |

## Implementation roadmap

| | Contents | Size | Ships alone |
|---|---|---|---|
| **M0** | Relying party ID as config; origin list; origin tests | S | Yes — unblocks Android verification |
| **M1** | Native app registration; serve both association files | S | Yes — also unblocks app2app's file question |
| **M2** | Enrolment HTTP API over the existing service calls | S | Yes — custom web UIs can use it |
| **M3** | Sign-in transport, per Q-IMPL-1 | M–L | Yes — custom native clients can use it |
| **M4** | Android SDK | M | Yes |
| **M5** | iOS SDK | M–L | Yes |

M0 before everything: without the origin list, no Android ceremony verifies.

M4 before M5: Android needs no bridging, so it proves the server with the least client code in the way.

## Verification

| Area | How |
|---|---|
| Origin verification | Unit test with a captured Android `clientDataJSON` carrying an `android:apk-key-hash:` origin, plus a web origin, against the same credential. Highest risk, cheapest test |
| Association files | Fetch both over HTTPS and assert a `200` with no redirect and the correct content type; assert the Apple file has no extension |
| Fingerprint derivation | Table test from a known keytool fingerprint to its base64url form |
| iOS bridging | Unit-test decode and re-assembly against fixtures. The ceremony itself cannot be unit-tested |
| Android | **Emulators cannot perform passkey ceremonies.** Physical device, API 28+, with a credential provider. CI covers error mapping only |
| iOS | Physical device on iOS 16+, or a simulator with iCloud Keychain configured. Needs a reachable relying party domain; Apple caches the association file 24–48h, so use developer mode while testing |
| End to end | Enrol in AuthUI, sign in from the app, and the reverse. This is the assertion that the credential really is shared, and it is the one that catches a wrong relying party ID |

Example apps (`authgear-sdk-ios/example`, `authgear-sdk-android/javasample`) need a passkey screen, the entitlement and manifest changes, and a documented signing fingerprint.

## Open implementation questions

| | Question | Part |
|---|---|---|
| Q-IMPL-1 | Which native sign-in transport, and does the spike support it? | [4](#part-4-server--native-sign-in-transport-undecided) |
| Q-IMPL-2 | Is native passkey sign-in restricted to full-access clients? | [4](#part-4-server--native-sign-in-transport-undecided) |
| Q-IMPL-3 | Does `go-webauthn` tolerate a non-URL expected origin? | [1.1](#11-multi-origin-verification) |
| Q-IMPL-4 | Does Authgear's generated `assetlinks.json` also carry app2app's relation? | [2.2](#22-serve-the-association-files) |
| Q-IMPL-5 | Which `androidx.credentials` version, and what does it cost in minimum API level? | [7.2](#72-dependencies) |
| Q-IMPL-6 | Which Credential Manager exception signals a domain-association failure? | [7.5](#75-errors) |

Product decisions live in the spec and are not repeated here. Q6 there — whether native sign-in is restricted to first-party public clients — is the same question as Q-IMPL-2 seen from the other side; the rest are resolved.
