package clientsignals

// matchKind describes how a marker variable is evaluated.
type matchKind int

const (
	// presence means the variable just needs to be set (to any value,
	// including empty) — matching the spec's "detected on presence of the
	// variable" wording, and consistent with how CI detection treats CI=""
	// as present (see isCI in ci.go).
	presence matchKind = iota
	// exactValue means the variable must equal one of the listed values,
	// exactly (case-sensitive).
	exactValue
)

// marker is one recognized (env var, expected value) pair contributing to
// agent detection. The env name is what gets emitted verbatim (never the
// value) as AgentSource, e.g. "env:CLAUDECODE".
//
// Do not add config-style variables that humans set by hand here (e.g.
// OPENCLAW_HOME, HERMES_HOME, PICOCLAW_HOME, PI_CODING_AGENT_DIR,
// KILO_ORG_ID, AIDER_*) — their presence says nothing about how the CLI was
// invoked. Never add secret-shaped variables (anything token/key/credential
// -like).
var knownMarkers = []marker{
	{agent: "claude-code", env: "CLAUDECODE", kind: exactValue, values: []string{"1"}},
	{agent: "claude-code", env: "CLAUDE_CODE_ENTRYPOINT", kind: presence},
	{agent: "pi", env: "PI_CODING_AGENT", kind: exactValue, values: []string{"true"}},
	{agent: "openclaw", env: "OPENCLAW_SHELL", kind: exactValue, values: []string{"exec"}},
	{agent: "openclaw", env: "OPENCLAW_CLI", kind: exactValue, values: []string{"1"}},
	{agent: "goose", env: "GOOSE_TERMINAL", kind: exactValue, values: []string{"1"}},
	{agent: "hermes", env: "HERMES_SESSION_ID", kind: presence},
	{agent: "codex", env: "CODEX_SANDBOX", kind: presence},
	{agent: "codex", env: "CODEX_THREAD_ID", kind: presence},
	{agent: "cursor", env: "CURSOR_TRACE_ID", kind: presence},
	{agent: "cursor", env: "CURSOR_AGENT", kind: presence},
	{agent: "gemini-cli", env: "GEMINI_CLI", kind: presence},
	{agent: "kiro", env: "TERM_PROGRAM", kind: exactValue, values: []string{"kiro"}},
	{agent: "antigravity", env: "ANTIGRAVITY_AGENT", kind: presence},
	{agent: "augment", env: "AUGMENT_AGENT", kind: presence},
	{agent: "replit", env: "REPL_ID", kind: presence},
	{agent: "opencode", env: "OPENCODE", kind: presence},
	{agent: "opencode", env: "OPENCODE_CALLER", kind: presence},
	{agent: "opencode", env: "OPENCODE_CLIENT", kind: presence},
	{agent: "copilot", env: "COPILOT_MODEL", kind: presence},
	{agent: "copilot", env: "COPILOT_ALLOW_ALL", kind: presence},
	{agent: "kilo-code", env: "KILO_PLATFORM", kind: exactValue, values: []string{"vscode"}},
}

type marker struct {
	agent  string
	env    string
	kind   matchKind
	values []string // only used when kind == exactValue
}
