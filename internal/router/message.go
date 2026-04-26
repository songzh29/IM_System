package router

import "encoding/json"

type ForwardMessage struct {
	TargetUserID uint            `json:"target_user_id"`
	TargetNodeID string          `json:"target_node_id"`
	Payload      json.RawMessage `json:"payload"`
}

const ForwardChannel = "im:forward"
