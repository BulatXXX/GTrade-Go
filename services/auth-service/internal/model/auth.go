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
	Role         string `json:"role,omitempty"`
}

type ActionStatusResponse struct {
	Status string `json:"status"`
}

type PasswordResetRequestResponse struct {
	Status string `json:"status"`
}

type EmailVerifyResponse struct {
	Status string `json:"status"`
}

type InternalUserEmailResponse struct {
	UserID        int64  `json:"user_id"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
}

type InternalUserContactsResponse struct {
	Users []InternalUserEmailResponse `json:"users"`
}

type UpdateUserRoleRequest struct {
	Role string `json:"role"`
}

type SetUserBlockedRequest struct {
	Blocked bool `json:"blocked"`
}

type AdminUserResponse struct {
	ID            int64  `json:"id"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Role          string `json:"role"`
	Blocked       bool   `json:"blocked"`
	CreatedAt     string `json:"created_at"`
}

type AdminUsersResponse struct {
	Users []AdminUserResponse `json:"users"`
}
