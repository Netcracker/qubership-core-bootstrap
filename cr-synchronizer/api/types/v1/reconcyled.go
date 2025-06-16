package v1

type DeclarationStatus struct {
	ObservedGeneration int          `json:"observedGeneration"`
	Phase              string       `json:"phase"`
	Conditions         []Conditions `json:"conditions"`
	Updated            bool         `json:"updated,omitempty"`
}

type Conditions struct {
	LastTransitionTime string `json:"lastTransitionTime"`
	LastUpdateTime     string `json:"lastUpdateTime"`
	Message            string `json:"message"`
	Reason             string `json:"reason"`
	Status             bool   `json:"status"`
	Type               string `json:"type"`
}
