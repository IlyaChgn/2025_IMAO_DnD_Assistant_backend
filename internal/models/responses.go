package models

type ErrResponse struct {
	Status string `json:"status"`
}

type WSErrResponse struct {
	Type  string `json:"type"`
	Error string `json:"error"`
}

type WSResponse struct {
	Type WSMsgType `json:"type"`
	Data any       `json:"data"`
}

type WSMsgType string

const (
	BattleInfo       WSMsgType = "battleInfo"
	ParticipantsInfo WSMsgType = "participantsInfo"

	// Patch message types - relayed directly without merging
	EncounterPatch   WSMsgType = "encounter_patch"
	FogHistoryPatch  WSMsgType = "fog_history_patch"
	WalkabilityPatch WSMsgType = "walkability_patch"
	OcclusionPatch   WSMsgType = "occlusion_patch"
	EdgesPatch     WSMsgType = "edges_patch"
	// InventoryPatch is injected server-side (HTTP→WS bridge), not sent by WS clients.
	// Listed here so IsPatchMessage returns true: if a client accidentally sends this type,
	// it gets relayed harmlessly instead of corrupting encounter state via merger.Merge.
	InventoryPatch WSMsgType = "inventory_patch"
	VisionPatch    WSMsgType = "vision_patch"
)

// IsPatchMessage returns true if the message type is a patch that should be relayed directly
func IsPatchMessage(msgType WSMsgType) bool {
	switch msgType {
	case EncounterPatch, FogHistoryPatch, WalkabilityPatch, OcclusionPatch, EdgesPatch, InventoryPatch, VisionPatch:
		return true
	default:
		return false
	}
}
