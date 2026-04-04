package model

type HealthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}

type SendEmailRequest struct {
	To       string `json:"to"`
	Subject  string `json:"subject"`
	HTMLBody string `json:"html_body"`
	TextBody string `json:"text_body"`
	From     string `json:"from,omitempty"`
}

type SendEmailResponse struct {
	ID     int64  `json:"id,omitempty"`
	Status string `json:"status"`
}
