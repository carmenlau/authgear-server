package authenticator

const (
	// AuthenticatorClaimTOTPDisplayName is a claim with string value for TOTP display name.
	AuthenticatorClaimTOTPDisplayName string = "https://authgear.com/claims/totp/display_name"
)

const (
	// AuthenticatorClaimOOBOTPChannelType is a claim with string value for OOB OTP channel type.
	AuthenticatorClaimOOBOTPChannelType string = "https://authgear.com/claims/oob_otp/channel_type"
	// AuthenticatorClaimOOBOTPEmail is a claim with string value for OOB OTP email channel.
	AuthenticatorClaimOOBOTPEmail string = "https://authgear.com/claims/oob_otp/email"
	// AuthenticatorClaimOOBOTPPhone is a claim with string value for OOB OTP phone channel.
	AuthenticatorClaimOOBOTPPhone string = "https://authgear.com/claims/oob_otp/phone"
)

const (
	// AuthenticatorStateOOBOTPCode is a claim with string value for OOB OTP code secret of current interaction.
	// nolint:gosec
	AuthenticatorStateOOBOTPSecret string = "https://authgear.com/claims/oob_otp/secret"
)
