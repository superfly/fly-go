package fly

type Plan struct {
	OldAppState *AppState
	NewAppState *AppState
	Strategy    string
	Token       string
	AppName     string
	OrgSlug     string
}

type AppState struct {
	Machines []*Machine
	Volumes  []*Volume
}
