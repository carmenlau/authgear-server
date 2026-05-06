# Plan: Username-Based Password Reset via Account Recovery Flow

## Context

Account recovery flows currently support only `email` and `phone` identification. The goal is to add `username` as a valid identification method, allowing a user to enter their username to trigger a password reset link sent to their associated email.

The architecture gap: username identifies the user but is not itself a delivery address. Once the user is found by username, the system must enumerate their other identities (email/phone) to find where to send the reset link. This is already supported via the `EnumerateDestinations: true` flag on the `select_destination` step.

## Files to Modify

### 1. Config schema + Go types
**`pkg/lib/config/authentication_flow.go`**

- Line 648: Add `"username"` to the `AuthenticationFlowAccountRecoveryIdentification` JSON schema enum alongside `"email"` and `"phone"`.
- After line 1393: Add Go constant:
  ```go
  AuthenticationFlowAccountRecoveryIdentificationUsername = AuthenticationFlowAccountRecoveryIdentification(model.AuthenticationFlowIdentificationUsername)
  ```

### 2. Auto-generated flow config
**`pkg/lib/authenticationflow/declarative/generate_config_account_recovery_flow.go`**

Add `hasUsername` tracking alongside `hasEmail`/`hasPhone`. When `hasUsername` is true, append a `OneOf` branch:
```go
&config.AuthenticationFlowAccountRecoveryFlowOneOf{
    Identification:      config.AuthenticationFlowAccountRecoveryIdentificationUsername,
    OnFailure_WriteOnly: config.AuthenticationFlowAccountRecoveryIdentificationOnFailureIgnore,
    Steps: []*config.AuthenticationFlowAccountRecoveryFlowStep{
        {
            Type:                 config.AuthenticationFlowAccountRecoveryFlowTypeSelectDestination,
            EnumerateDestinations: true,
            AllowedChannels:      <combined email+phone channels>,
        },
    },
}
```
For `AllowedChannels`, combine `cfg.UI.ForgotPassword.Email` and `cfg.UI.ForgotPassword.Phone`.

### 3. Option building + routing
**`pkg/lib/authenticationflow/declarative/intent_account_recovery_flow_step_identify.go`**

- In `NewIntentAccountRecoveryFlowStepIdentify()` (line 69-76): add `case config.AuthenticationFlowAccountRecoveryIdentificationUsername:` with fallthrough to include it in the options slice.
- In `ReactTo()` (line 141-150): add `case config.AuthenticationFlowAccountRecoveryIdentificationUsername:` to dispatch to `IntentUseAccountRecoveryIdentity`.

### 4. Input schema validation
**`pkg/lib/authenticationflow/declarative/input_step_account_recovery_identify.go`**

In `SchemaBuilder()` switch (line 58-67), add:
```go
case config.AuthenticationFlowAccountRecoveryIdentificationUsername:
    requireString("login_id")
    setRequiredAndAppendOneOf()
```

### 5. Config validation — JSON Schema + Go
**`pkg/lib/config/authentication_flow.go`** (JSON Schema)

Add an `if-then` rule to the `AuthenticationFlowAccountRecoveryFlowOneOf` schema to enforce that if `identification: username`, nested `select_destination` steps **must** have `enumerate_destinations: true`:

```json
"allOf": [
  {
    "if": {
      "properties": { "identification": { "const": "username" } },
      "required": ["identification"]
    },
    "then": {
      "properties": {
        "steps": {
          "type": "array",
          "items": {
            "if": {
              "properties": { "type": { "const": "select_destination" } }
            },
            "then": {
              "required": ["enumerate_destinations"],
              "properties": {
                "enumerate_destinations": { "const": true }
              }
            }
          }
        }
      }
    }
  }
]
```

**Find `AuthgearYAMLDescriptor`** (likely in `pkg/lib/config/authgear.go` or similar)

Add validation method to catch flat config patterns:
```go
func (d *AuthgearYAMLDescriptor) ValidateAccountRecoveryFlows(config *AuthenticationFlowConfig) error {
    if config == nil || config.AccountRecoveryFlows == nil {
        return nil
    }
    
    for _, flow := range config.AccountRecoveryFlows {
        if err := d.validateAccountRecoveryFlow(flow); err != nil {
            return err
        }
    }
    return nil
}

func (d *AuthgearYAMLDescriptor) validateAccountRecoveryFlow(flow *AuthenticationFlowAccountRecoveryFlow) error {
    // Check if any identify step has username identification
    var hasUsernameIdentify bool
    
    for _, step := range flow.Steps {
        if step.Type == AuthenticationFlowAccountRecoveryFlowTypeIdentify {
            for _, branch := range step.OneOf {
                if branch.Identification == AuthenticationFlowAccountRecoveryIdentificationUsername {
                    hasUsernameIdentify = true
                    break
                }
            }
        }
    }
    
    // If username identify exists, validate that select_destination has enumerate_destinations: true
    if hasUsernameIdentify {
        for _, step := range flow.Steps {
            if step.Type == AuthenticationFlowAccountRecoveryFlowTypeSelectDestination {
                if !step.EnumerateDestinations {
                    return fmt.Errorf(
                        "account recovery flow %q: identification 'username' requires enumerate_destinations: true in select_destination step",
                        flow.Name,
                    )
                }
            }
        }
    }
    
    return nil
}
```

Call this validation method from the descriptor's main validation flow (when config is loaded).

**Validation Coverage:**
- **Nested config:** JSON schema prevents `identification: username` without `enumerate_destinations: true`
- **Flat config:** AuthgearYAMLDescriptor validation at config load time prevents invalid username recovery flows
- **Result:** Invalid configs are caught at startup, before any requests are served. No runtime errors possible.

### 6. Code-level defensive safety
**`pkg/lib/authenticationflow/declarative/intent_account_recovery_flow_step_select_destination.go`**

Add a defensive check in `deriveAccountRecoveryDestinationOptions()` (line 174) as a safety net. Even though config validation prevents invalid configs, if somehow a username identification reaches this point without enumeration, force it:

```go
forceEnumerate := iden.Identification == config.AuthenticationFlowAccountRecoveryIdentificationUsername
if iden.MaybeIdentity != nil && (step.EnumerateDestinations || forceEnumerate) {
    // Enumerate user's email/phone identities
}
```

This prevents runtime failures if validation is somehow bypassed or if code changes unintentionally create invalid flows.

### 7. Webapp handler
**`pkg/auth/handler/webapp/authflowv2/forgot_password.go`**

- Add `"username"` to `AuthflowV2ForgotPasswordSchema` enum (line 32).
- Add `ForgotPasswordLoginIDInputTypeUsername ForgotPasswordLoginIDInputType = "username"` constant.
- Update `IsValid()` to include `ForgotPasswordLoginIDInputTypeUsername`.
- Add `AuthFlowV2ForgotPasswordAlternativeTypeUsername` constant.
- Update `forgotPasswordGetInitialLoginIDInputType()` to return `ForgotPasswordLoginIDInputTypeUsername` for username.
- Update `deriveForgotPasswordAlternatives()` to add username alternative when current type is not username and `usernameLoginIDEnabled`.
- Update `NewAuthFlowV2ForgotPasswordViewModel()`:
  - Track `usernameLoginIDEnabled` in the options loop.
  - Update `loginIDDisabled` to also check `usernameLoginIDEnabled`.
  - Pass `usernameLoginIDEnabled` through `AuthFlowV2ForgotPasswordViewModel`.
- Add `UsernameLoginIDEnabled bool` field to `AuthFlowV2ForgotPasswordViewModel`.

### 8. View model — bot protection
**`pkg/auth/handler/webapp/viewmodels/authflow.go`**

In `NewWithAccountRecoveryAuthflow()` (line 325-331), add:
```go
case config.AuthenticationFlowAccountRecoveryIdentificationUsername:
    bpRequiredUsername = opt.BotProtection.IsRequired()
```
And return it in `UsernameLoginIDBotProtectionRequired: bpRequiredUsername`.

### 9. HTML template
**`resources/authgear/templates/en/web/authflowv2/forgot_password.html`**

- Add `$show_username_input := false` variable.
- Add username show logic: `{{ if and (eq $.LoginIDInputType "text") ($.UsernameLoginIDEnabled) }}`.
- Render a plain text `<input type="text">` with `name="x_login_id"` and `x_login_id_type=username` hidden input, similar to the email input block.
- Add username description and placeholder strings (i18n keys: `v2.page.forgot-password.default.username-input-description`).
- Add bot protection captcha for username if `UsernameLoginIDBotProtectionRequired`.
- Update alternatives template to handle `AuthFlowV2ForgotPasswordAlternativeTypeUsername`.

### 10. Tests
**`pkg/lib/authenticationflow/declarative/generate_config_account_recovery_flow_test.go`**

Add test cases:
- Username only → generates username branch with `EnumerateDestinations: true`.
- Username + email → generates both branches.

**`pkg/lib/config/authentication_flow_test.go`** (if it exists)

Add test cases for AuthgearYAMLDescriptor validation:
- Invalid: Username identify with flat select_destination without enumerate_destinations → validation error
- Invalid: Username identify with no select_destination step → validation error
- Valid: Username identify with enumerate_destinations: true
- Valid: Username identify with nested select_destination having enumerate_destinations: true

## Key reused code (no changes needed)

- `makeLoginIDSpec()` in `utils_common.go`: already handles `AuthenticationFlowIdentificationUsername`.
- `SearchBySpec()` / `findExactOneIdentityInfo()`: already handle username identity specs.
- `enumerateAllowedAccountRecoveryDestinationOptions()`: already correctly derives email/phone options from user identity list.
- `forgotpassword.Service.SendCode()`: operates on the resolved email/phone target, not the username.

## Verification

### Config Validation Tests
1. Try creating a config with `identification: username` without `enumerate_destinations: true` → **Should fail with clear error during config load**
2. Try flat pattern: username identify at top level, flat select_destination without enumerate_destinations → **Should fail at config validation**
3. Try nested pattern: username identify with nested select_destination without enumerate_destinations → **Should fail at JSON schema validation**
4. Valid nested pattern: username with enumerate_destinations: true → **Should succeed**
5. Valid flat pattern: flat select_destination with enumerate_destinations: true → **Should succeed**

### Runtime Flow Tests
1. Configure an app with only `username` login ID key.
2. System auto-generates account_recovery_flow with username branch (enumerate_destinations: true).
3. User visits forgot password page → sees username input.
4. User enters username → system finds user → shows their email destinations.
5. User picks email → receives reset link.

### Test Commands
- `go test ./pkg/lib/authenticationflow/declarative/... -run TestGenerateAccountRecoveryFlowConfig` — Verify generated config
- `go test ./pkg/lib/config/... -run TestAuthenticationFlowAccountRecoveryFlow` — Verify schema validation
- `go test ./pkg/lib/config/... -run ValidateAccountRecoveryFlows` — Verify AuthgearYAMLDescriptor validation

## Configuration Example

```yaml
authentication:
  account_recovery_flows:
    - name: default
      steps:
        - type: identify
          one_of:
            - identification: username
              on_failure: ignore
              steps:
                - type: select_destination
                  enumerate_destinations: true
                  allowed_channels:
                    - channel: email
                      otp_form: link
                    - channel: sms
                      otp_form: code
        - type: verify_account_recovery_code
        - type: reset_password
```
