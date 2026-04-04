package model

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type PasswordResetRequest struct {
	Email string `json:"email"`
}

type PasswordResetConfirmRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

type EmailVerifyRequest struct {
	Email string `json:"email"`
	Token string `json:"token"`
}

type TokenPairResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
}

type ActionStatusResponse struct {
	Status string `json:"status"`
}

type PasswordResetRequestResponse struct {
	Status     string `json:"status"`
	ResetToken string `json:"reset_token,omitempty"`
}

type EmailVerifyResponse struct {
	Status            string `json:"status"`
	VerificationToken string `json:"verification_token,omitempty"`
}
