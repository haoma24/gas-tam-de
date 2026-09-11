package main

// Types in this file describe JSON envelopes that handlers currently build
// with maps. Keeping named shapes makes the generated gateway Swagger useful
// to clients without changing the wire format.

type otpRequestResponse struct {
	OK             bool   `json:"ok"`
	PhoneMasked    string `json:"phone_masked"`
	ExpiresInSec   int    `json:"expires_in_sec"`
	ResendAfterSec int    `json:"resend_after_sec"`
	DevCode        string `json:"dev_code,omitempty"`
}

type authUserView struct {
	ID          string `json:"id"`
	Role        string `json:"role"`
	PhoneMasked string `json:"phone_masked,omitempty"`
	Username    string `json:"username,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
}

type authTokenResponse struct {
	OK           bool         `json:"ok"`
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	TokenType    string       `json:"token_type"`
	ExpiresIn    int          `json:"expires_in"`
	User         authUserView `json:"user"`
}

type refreshResponse struct {
	OK           bool   `json:"ok"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

type okResponse struct {
	OK bool `json:"ok"`
}

type deleteAdminPhoneResponse struct {
	OK bool   `json:"ok"`
	ID string `json:"id"`
}

type adminPhoneListResponse struct {
	AdminPhones []adminPhoneView `json:"admin_phones"`
}

type adminAccountListResponse struct {
	AdminAccounts []adminAccountView `json:"admin_accounts"`
}
