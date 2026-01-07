package socket

type MessageType string

const (
	MsgLocation MessageType = "location"
)

type LocationMessage struct {
	Type     MessageType `json:"type"`
	UserID   string      `json:"userId"`
	Lat      float64     `json:"lat"`
	Lng      float64     `json:"lng"`
	Accuracy float64     `json:"accuracy"`
	TS       int64       `json:"ts"`
}
