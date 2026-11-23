package dto

type SendDialogMessageRequest struct {
	Text *string `json:"text"`
}

type DialogMessageResponse struct {
	From string `json:"from"`
	To   string `json:"to"`
	Text string `json:"text"`
}

type DialogPageResponse struct {
	Items      []DialogMessageResponse `json:"items"`
	NextCursor *string                 `json:"next_cursor,omitempty"`
}
