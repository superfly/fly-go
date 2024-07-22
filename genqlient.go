// Code generated by github.com/Khan/genqlient, DO NOT EDIT.

package fly

import (
	"context"

	"github.com/Khan/genqlient/graphql"
)

type BuildFinalImageInput struct {
	// Sha256 id of docker image
	Id string `json:"id"`
	// Size in bytes of the docker image
	SizeBytes int64 `json:"sizeBytes"`
	// Tag used for docker image
	Tag string `json:"tag"`
}

// GetId returns BuildFinalImageInput.Id, and is useful for accessing the field via an interface.
func (v *BuildFinalImageInput) GetId() string { return v.Id }

// GetSizeBytes returns BuildFinalImageInput.SizeBytes, and is useful for accessing the field via an interface.
func (v *BuildFinalImageInput) GetSizeBytes() int64 { return v.SizeBytes }

// GetTag returns BuildFinalImageInput.Tag, and is useful for accessing the field via an interface.
func (v *BuildFinalImageInput) GetTag() string { return v.Tag }

type BuildImageOptsInput struct {
	// Set of build time variables passed to cli
	BuildArgs interface{} `json:"buildArgs"`
	// Fly.toml build.buildpacks setting
	BuildPacks []string `json:"buildPacks"`
	// Fly.toml build.builder setting
	Builder string `json:"builder"`
	// Builtin builder to use
	BuiltIn string `json:"builtIn"`
	// Builtin builder settings
	BuiltInSettings interface{} `json:"builtInSettings"`
	// Path to dockerfile, if one exists
	DockerfilePath string `json:"dockerfilePath"`
	// Unused in cli?
	ExtraBuildArgs interface{} `json:"extraBuildArgs"`
	// Image label to use when tagging and pushing to the fly registry
	ImageLabel string `json:"imageLabel"`
	// Unused in cli?
	ImageRef string `json:"imageRef"`
	// Do not use the build cache when building the image
	NoCache bool `json:"noCache"`
	// Whether publishing to the registry was requested
	Publish bool `json:"publish"`
	// Docker tag used to publish image to registry
	Tag string `json:"tag"`
	// Set the target build stage to build if the Dockerfile has more than one stage
	Target string `json:"target"`
}

// GetBuildArgs returns BuildImageOptsInput.BuildArgs, and is useful for accessing the field via an interface.
func (v *BuildImageOptsInput) GetBuildArgs() interface{} { return v.BuildArgs }

// GetBuildPacks returns BuildImageOptsInput.BuildPacks, and is useful for accessing the field via an interface.
func (v *BuildImageOptsInput) GetBuildPacks() []string { return v.BuildPacks }

// GetBuilder returns BuildImageOptsInput.Builder, and is useful for accessing the field via an interface.
func (v *BuildImageOptsInput) GetBuilder() string { return v.Builder }

// GetBuiltIn returns BuildImageOptsInput.BuiltIn, and is useful for accessing the field via an interface.
func (v *BuildImageOptsInput) GetBuiltIn() string { return v.BuiltIn }

// GetBuiltInSettings returns BuildImageOptsInput.BuiltInSettings, and is useful for accessing the field via an interface.
func (v *BuildImageOptsInput) GetBuiltInSettings() interface{} { return v.BuiltInSettings }

// GetDockerfilePath returns BuildImageOptsInput.DockerfilePath, and is useful for accessing the field via an interface.
func (v *BuildImageOptsInput) GetDockerfilePath() string { return v.DockerfilePath }

// GetExtraBuildArgs returns BuildImageOptsInput.ExtraBuildArgs, and is useful for accessing the field via an interface.
func (v *BuildImageOptsInput) GetExtraBuildArgs() interface{} { return v.ExtraBuildArgs }

// GetImageLabel returns BuildImageOptsInput.ImageLabel, and is useful for accessing the field via an interface.
func (v *BuildImageOptsInput) GetImageLabel() string { return v.ImageLabel }

// GetImageRef returns BuildImageOptsInput.ImageRef, and is useful for accessing the field via an interface.
func (v *BuildImageOptsInput) GetImageRef() string { return v.ImageRef }

// GetNoCache returns BuildImageOptsInput.NoCache, and is useful for accessing the field via an interface.
func (v *BuildImageOptsInput) GetNoCache() bool { return v.NoCache }

// GetPublish returns BuildImageOptsInput.Publish, and is useful for accessing the field via an interface.
func (v *BuildImageOptsInput) GetPublish() bool { return v.Publish }

// GetTag returns BuildImageOptsInput.Tag, and is useful for accessing the field via an interface.
func (v *BuildImageOptsInput) GetTag() string { return v.Tag }

// GetTarget returns BuildImageOptsInput.Target, and is useful for accessing the field via an interface.
func (v *BuildImageOptsInput) GetTarget() string { return v.Target }

type BuildStrategyAttemptInput struct {
	// Optional error message from strategy
	Error string `json:"error"`
	// Optional note about this strategy or its result
	Note string `json:"note"`
	// Result attempting this strategy
	Result string `json:"result"`
	// Build strategy attempted
	Strategy string `json:"strategy"`
}

// GetError returns BuildStrategyAttemptInput.Error, and is useful for accessing the field via an interface.
func (v *BuildStrategyAttemptInput) GetError() string { return v.Error }

// GetNote returns BuildStrategyAttemptInput.Note, and is useful for accessing the field via an interface.
func (v *BuildStrategyAttemptInput) GetNote() string { return v.Note }

// GetResult returns BuildStrategyAttemptInput.Result, and is useful for accessing the field via an interface.
func (v *BuildStrategyAttemptInput) GetResult() string { return v.Result }

// GetStrategy returns BuildStrategyAttemptInput.Strategy, and is useful for accessing the field via an interface.
func (v *BuildStrategyAttemptInput) GetStrategy() string { return v.Strategy }

type BuildTimingsInput struct {
	// Time to build and push the image, measured by flyctl
	BuildAndPushMs int64 `json:"buildAndPushMs"`
	// Time to build the image including create context, measured by flyctl
	BuildMs int64 `json:"buildMs"`
	// Time to initialize client used to connect to either remote or local builder
	BuilderInitMs int64 `json:"builderInitMs"`
	// Time to create the build context tar file, measured by flyctl
	ContextBuildMs int64 `json:"contextBuildMs"`
	// Time for builder to build image after receiving context, measured by flyctl
	ImageBuildMs int64 `json:"imageBuildMs"`
	// Time to push completed image to registry, measured by flyctl
	PushMs int64 `json:"pushMs"`
}

// GetBuildAndPushMs returns BuildTimingsInput.BuildAndPushMs, and is useful for accessing the field via an interface.
func (v *BuildTimingsInput) GetBuildAndPushMs() int64 { return v.BuildAndPushMs }

// GetBuildMs returns BuildTimingsInput.BuildMs, and is useful for accessing the field via an interface.
func (v *BuildTimingsInput) GetBuildMs() int64 { return v.BuildMs }

// GetBuilderInitMs returns BuildTimingsInput.BuilderInitMs, and is useful for accessing the field via an interface.
func (v *BuildTimingsInput) GetBuilderInitMs() int64 { return v.BuilderInitMs }

// GetContextBuildMs returns BuildTimingsInput.ContextBuildMs, and is useful for accessing the field via an interface.
func (v *BuildTimingsInput) GetContextBuildMs() int64 { return v.ContextBuildMs }

// GetImageBuildMs returns BuildTimingsInput.ImageBuildMs, and is useful for accessing the field via an interface.
func (v *BuildTimingsInput) GetImageBuildMs() int64 { return v.ImageBuildMs }

// GetPushMs returns BuildTimingsInput.PushMs, and is useful for accessing the field via an interface.
func (v *BuildTimingsInput) GetPushMs() int64 { return v.PushMs }

type BuilderMetaInput struct {
	// Local or remote builder type
	BuilderType string `json:"builderType"`
	// Whther or not buildkit is enabled on builder
	BuildkitEnabled bool `json:"buildkitEnabled"`
	// Docker version reported by builder
	DockerVersion string `json:"dockerVersion"`
	// Platform reported by the builder
	Platform string `json:"platform"`
	// Remote builder app used
	RemoteAppName string `json:"remoteAppName"`
	// Remote builder machine used
	RemoteMachineId string `json:"remoteMachineId"`
}

// GetBuilderType returns BuilderMetaInput.BuilderType, and is useful for accessing the field via an interface.
func (v *BuilderMetaInput) GetBuilderType() string { return v.BuilderType }

// GetBuildkitEnabled returns BuilderMetaInput.BuildkitEnabled, and is useful for accessing the field via an interface.
func (v *BuilderMetaInput) GetBuildkitEnabled() bool { return v.BuildkitEnabled }

// GetDockerVersion returns BuilderMetaInput.DockerVersion, and is useful for accessing the field via an interface.
func (v *BuilderMetaInput) GetDockerVersion() string { return v.DockerVersion }

// GetPlatform returns BuilderMetaInput.Platform, and is useful for accessing the field via an interface.
func (v *BuilderMetaInput) GetPlatform() string { return v.Platform }

// GetRemoteAppName returns BuilderMetaInput.RemoteAppName, and is useful for accessing the field via an interface.
func (v *BuilderMetaInput) GetRemoteAppName() string { return v.RemoteAppName }

// GetRemoteMachineId returns BuilderMetaInput.RemoteMachineId, and is useful for accessing the field via an interface.
func (v *BuilderMetaInput) GetRemoteMachineId() string { return v.RemoteMachineId }

// CreateBuildCreateBuildCreateBuildPayload includes the requested fields of the GraphQL type CreateBuildPayload.
// The GraphQL type's documentation follows.
//
// Autogenerated return type of CreateBuild.
type CreateBuildCreateBuildCreateBuildPayload struct {
	// build id
	Id string `json:"id"`
	// stored build status
	Status string `json:"status"`
}

// GetId returns CreateBuildCreateBuildCreateBuildPayload.Id, and is useful for accessing the field via an interface.
func (v *CreateBuildCreateBuildCreateBuildPayload) GetId() string { return v.Id }

// GetStatus returns CreateBuildCreateBuildCreateBuildPayload.Status, and is useful for accessing the field via an interface.
func (v *CreateBuildCreateBuildCreateBuildPayload) GetStatus() string { return v.Status }

// Autogenerated input type of CreateBuild
type CreateBuildInput struct {
	// The name of the app being built
	AppName string `json:"appName"`
	// Whether builder is remote or local
	BuilderType string `json:"builderType"`
	// A unique identifier for the client performing the mutation.
	ClientMutationId string `json:"clientMutationId"`
	// Options set for building image
	ImageOpts BuildImageOptsInput `json:"imageOpts"`
	// The ID of the machine being built (only set for machine builds)
	MachineId string `json:"machineId"`
	// List of available build strategies that will be attempted
	StrategiesAvailable []string `json:"strategiesAvailable"`
}

// GetAppName returns CreateBuildInput.AppName, and is useful for accessing the field via an interface.
func (v *CreateBuildInput) GetAppName() string { return v.AppName }

// GetBuilderType returns CreateBuildInput.BuilderType, and is useful for accessing the field via an interface.
func (v *CreateBuildInput) GetBuilderType() string { return v.BuilderType }

// GetClientMutationId returns CreateBuildInput.ClientMutationId, and is useful for accessing the field via an interface.
func (v *CreateBuildInput) GetClientMutationId() string { return v.ClientMutationId }

// GetImageOpts returns CreateBuildInput.ImageOpts, and is useful for accessing the field via an interface.
func (v *CreateBuildInput) GetImageOpts() BuildImageOptsInput { return v.ImageOpts }

// GetMachineId returns CreateBuildInput.MachineId, and is useful for accessing the field via an interface.
func (v *CreateBuildInput) GetMachineId() string { return v.MachineId }

// GetStrategiesAvailable returns CreateBuildInput.StrategiesAvailable, and is useful for accessing the field via an interface.
func (v *CreateBuildInput) GetStrategiesAvailable() []string { return v.StrategiesAvailable }

// CreateBuildResponse is returned by CreateBuild on success.
type CreateBuildResponse struct {
	CreateBuild CreateBuildCreateBuildCreateBuildPayload `json:"createBuild"`
}

// GetCreateBuild returns CreateBuildResponse.CreateBuild, and is useful for accessing the field via an interface.
func (v *CreateBuildResponse) GetCreateBuild() CreateBuildCreateBuildCreateBuildPayload {
	return v.CreateBuild
}

// CreateReleaseCreateReleaseCreateReleasePayload includes the requested fields of the GraphQL type CreateReleasePayload.
// The GraphQL type's documentation follows.
//
// Autogenerated return type of CreateRelease.
type CreateReleaseCreateReleaseCreateReleasePayload struct {
	Release CreateReleaseCreateReleaseCreateReleasePayloadRelease `json:"release"`
}

// GetRelease returns CreateReleaseCreateReleaseCreateReleasePayload.Release, and is useful for accessing the field via an interface.
func (v *CreateReleaseCreateReleaseCreateReleasePayload) GetRelease() CreateReleaseCreateReleaseCreateReleasePayloadRelease {
	return v.Release
}

// CreateReleaseCreateReleaseCreateReleasePayloadRelease includes the requested fields of the GraphQL type Release.
type CreateReleaseCreateReleaseCreateReleasePayloadRelease struct {
	// Unique ID
	Id string `json:"id"`
	// The version of the release
	Version int `json:"version"`
}

// GetId returns CreateReleaseCreateReleaseCreateReleasePayloadRelease.Id, and is useful for accessing the field via an interface.
func (v *CreateReleaseCreateReleaseCreateReleasePayloadRelease) GetId() string { return v.Id }

// GetVersion returns CreateReleaseCreateReleaseCreateReleasePayloadRelease.Version, and is useful for accessing the field via an interface.
func (v *CreateReleaseCreateReleaseCreateReleasePayloadRelease) GetVersion() int { return v.Version }

// Autogenerated input type of CreateRelease
type CreateReleaseInput struct {
	// The ID of the app
	AppId string `json:"appId"`
	// A unique identifier for the client performing the mutation.
	ClientMutationId string `json:"clientMutationId"`
	// app definition
	Definition interface{} `json:"definition"`
	// The image to deploy
	Image string `json:"image"`
	// nomad or machines
	PlatformVersion string `json:"platformVersion"`
	// The strategy for replacing existing instances. Defaults to canary.
	Strategy DeploymentStrategy `json:"strategy"`
}

// GetAppId returns CreateReleaseInput.AppId, and is useful for accessing the field via an interface.
func (v *CreateReleaseInput) GetAppId() string { return v.AppId }

// GetClientMutationId returns CreateReleaseInput.ClientMutationId, and is useful for accessing the field via an interface.
func (v *CreateReleaseInput) GetClientMutationId() string { return v.ClientMutationId }

// GetDefinition returns CreateReleaseInput.Definition, and is useful for accessing the field via an interface.
func (v *CreateReleaseInput) GetDefinition() interface{} { return v.Definition }

// GetImage returns CreateReleaseInput.Image, and is useful for accessing the field via an interface.
func (v *CreateReleaseInput) GetImage() string { return v.Image }

// GetPlatformVersion returns CreateReleaseInput.PlatformVersion, and is useful for accessing the field via an interface.
func (v *CreateReleaseInput) GetPlatformVersion() string { return v.PlatformVersion }

// GetStrategy returns CreateReleaseInput.Strategy, and is useful for accessing the field via an interface.
func (v *CreateReleaseInput) GetStrategy() DeploymentStrategy { return v.Strategy }

// CreateReleaseResponse is returned by CreateRelease on success.
type CreateReleaseResponse struct {
	CreateRelease CreateReleaseCreateReleaseCreateReleasePayload `json:"createRelease"`
}

// GetCreateRelease returns CreateReleaseResponse.CreateRelease, and is useful for accessing the field via an interface.
func (v *CreateReleaseResponse) GetCreateRelease() CreateReleaseCreateReleaseCreateReleasePayload {
	return v.CreateRelease
}

type DeploymentStrategy string

const (
	// Launch all new instances before shutting down previous instances
	DeploymentStrategyBluegreen DeploymentStrategy = "BLUEGREEN"
	// Ensure new instances are healthy before continuing with a rolling deployment
	DeploymentStrategyCanary DeploymentStrategy = "CANARY"
	// Deploy new instances all at once
	DeploymentStrategyImmediate DeploymentStrategy = "IMMEDIATE"
	// Incrementally replace old instances with new ones
	DeploymentStrategyRolling DeploymentStrategy = "ROLLING"
	// Incrementally replace old instances with new ones, 1 by 1
	DeploymentStrategyRollingOne DeploymentStrategy = "ROLLING_ONE"
	// Deploy new instances all at once
	DeploymentStrategySimple DeploymentStrategy = "SIMPLE"
)

// EnsureDepotRemoteBuilderEnsureDepotRemoteBuilderEnsureDepotRemoteBuilderPayload includes the requested fields of the GraphQL type EnsureDepotRemoteBuilderPayload.
// The GraphQL type's documentation follows.
//
// Autogenerated return type of EnsureDepotRemoteBuilder.
type EnsureDepotRemoteBuilderEnsureDepotRemoteBuilderEnsureDepotRemoteBuilderPayload struct {
	BuildId    *string `json:"buildId"`
	BuildToken *string `json:"buildToken"`
}

// GetBuildId returns EnsureDepotRemoteBuilderEnsureDepotRemoteBuilderEnsureDepotRemoteBuilderPayload.BuildId, and is useful for accessing the field via an interface.
func (v *EnsureDepotRemoteBuilderEnsureDepotRemoteBuilderEnsureDepotRemoteBuilderPayload) GetBuildId() *string {
	return v.BuildId
}

// GetBuildToken returns EnsureDepotRemoteBuilderEnsureDepotRemoteBuilderEnsureDepotRemoteBuilderPayload.BuildToken, and is useful for accessing the field via an interface.
func (v *EnsureDepotRemoteBuilderEnsureDepotRemoteBuilderEnsureDepotRemoteBuilderPayload) GetBuildToken() *string {
	return v.BuildToken
}

// Autogenerated input type of EnsureDepotRemoteBuilder
type EnsureDepotRemoteBuilderInput struct {
	// The unique application name
	AppName *string `json:"appName"`
	// A unique identifier for the client performing the mutation.
	ClientMutationId *string `json:"clientMutationId"`
	// The node ID of the organization
	OrganizationId *string `json:"organizationId"`
	// Desired region for the remote builder
	Region *string `json:"region"`
}

// GetAppName returns EnsureDepotRemoteBuilderInput.AppName, and is useful for accessing the field via an interface.
func (v *EnsureDepotRemoteBuilderInput) GetAppName() *string { return v.AppName }

// GetClientMutationId returns EnsureDepotRemoteBuilderInput.ClientMutationId, and is useful for accessing the field via an interface.
func (v *EnsureDepotRemoteBuilderInput) GetClientMutationId() *string { return v.ClientMutationId }

// GetOrganizationId returns EnsureDepotRemoteBuilderInput.OrganizationId, and is useful for accessing the field via an interface.
func (v *EnsureDepotRemoteBuilderInput) GetOrganizationId() *string { return v.OrganizationId }

// GetRegion returns EnsureDepotRemoteBuilderInput.Region, and is useful for accessing the field via an interface.
func (v *EnsureDepotRemoteBuilderInput) GetRegion() *string { return v.Region }

// EnsureDepotRemoteBuilderResponse is returned by EnsureDepotRemoteBuilder on success.
type EnsureDepotRemoteBuilderResponse struct {
	EnsureDepotRemoteBuilder *EnsureDepotRemoteBuilderEnsureDepotRemoteBuilderEnsureDepotRemoteBuilderPayload `json:"ensureDepotRemoteBuilder"`
}

// GetEnsureDepotRemoteBuilder returns EnsureDepotRemoteBuilderResponse.EnsureDepotRemoteBuilder, and is useful for accessing the field via an interface.
func (v *EnsureDepotRemoteBuilderResponse) GetEnsureDepotRemoteBuilder() *EnsureDepotRemoteBuilderEnsureDepotRemoteBuilderEnsureDepotRemoteBuilderPayload {
	return v.EnsureDepotRemoteBuilder
}

// FinishBuildFinishBuildFinishBuildPayload includes the requested fields of the GraphQL type FinishBuildPayload.
// The GraphQL type's documentation follows.
//
// Autogenerated return type of FinishBuild.
type FinishBuildFinishBuildFinishBuildPayload struct {
	// build id
	Id string `json:"id"`
	// stored build status
	Status string `json:"status"`
	// wall clock time for this build
	WallclockTimeMs int `json:"wallclockTimeMs"`
}

// GetId returns FinishBuildFinishBuildFinishBuildPayload.Id, and is useful for accessing the field via an interface.
func (v *FinishBuildFinishBuildFinishBuildPayload) GetId() string { return v.Id }

// GetStatus returns FinishBuildFinishBuildFinishBuildPayload.Status, and is useful for accessing the field via an interface.
func (v *FinishBuildFinishBuildFinishBuildPayload) GetStatus() string { return v.Status }

// GetWallclockTimeMs returns FinishBuildFinishBuildFinishBuildPayload.WallclockTimeMs, and is useful for accessing the field via an interface.
func (v *FinishBuildFinishBuildFinishBuildPayload) GetWallclockTimeMs() int { return v.WallclockTimeMs }

// Autogenerated input type of FinishBuild
type FinishBuildInput struct {
	// The name of the app being built
	AppName string `json:"appName"`
	// Build id returned by createBuild() mutation
	BuildId string `json:"buildId"`
	// Metadata about the builder
	BuilderMeta BuilderMetaInput `json:"builderMeta"`
	// A unique identifier for the client performing the mutation.
	ClientMutationId string `json:"clientMutationId"`
	// Information about the docker image that was built
	FinalImage BuildFinalImageInput `json:"finalImage"`
	// Log or error output
	Logs string `json:"logs"`
	// The ID of the machine being built (only set for machine builds)
	MachineId string `json:"machineId"`
	// Indicate whether build completed or failed
	Status string `json:"status"`
	// Build strategies attempted and their result, should be in order of attempt
	StrategiesAttempted []BuildStrategyAttemptInput `json:"strategiesAttempted"`
	// Timings for different phases of the build
	Timings BuildTimingsInput `json:"timings"`
}

// GetAppName returns FinishBuildInput.AppName, and is useful for accessing the field via an interface.
func (v *FinishBuildInput) GetAppName() string { return v.AppName }

// GetBuildId returns FinishBuildInput.BuildId, and is useful for accessing the field via an interface.
func (v *FinishBuildInput) GetBuildId() string { return v.BuildId }

// GetBuilderMeta returns FinishBuildInput.BuilderMeta, and is useful for accessing the field via an interface.
func (v *FinishBuildInput) GetBuilderMeta() BuilderMetaInput { return v.BuilderMeta }

// GetClientMutationId returns FinishBuildInput.ClientMutationId, and is useful for accessing the field via an interface.
func (v *FinishBuildInput) GetClientMutationId() string { return v.ClientMutationId }

// GetFinalImage returns FinishBuildInput.FinalImage, and is useful for accessing the field via an interface.
func (v *FinishBuildInput) GetFinalImage() BuildFinalImageInput { return v.FinalImage }

// GetLogs returns FinishBuildInput.Logs, and is useful for accessing the field via an interface.
func (v *FinishBuildInput) GetLogs() string { return v.Logs }

// GetMachineId returns FinishBuildInput.MachineId, and is useful for accessing the field via an interface.
func (v *FinishBuildInput) GetMachineId() string { return v.MachineId }

// GetStatus returns FinishBuildInput.Status, and is useful for accessing the field via an interface.
func (v *FinishBuildInput) GetStatus() string { return v.Status }

// GetStrategiesAttempted returns FinishBuildInput.StrategiesAttempted, and is useful for accessing the field via an interface.
func (v *FinishBuildInput) GetStrategiesAttempted() []BuildStrategyAttemptInput {
	return v.StrategiesAttempted
}

// GetTimings returns FinishBuildInput.Timings, and is useful for accessing the field via an interface.
func (v *FinishBuildInput) GetTimings() BuildTimingsInput { return v.Timings }

// FinishBuildResponse is returned by FinishBuild on success.
type FinishBuildResponse struct {
	FinishBuild FinishBuildFinishBuildFinishBuildPayload `json:"finishBuild"`
}

// GetFinishBuild returns FinishBuildResponse.FinishBuild, and is useful for accessing the field via an interface.
func (v *FinishBuildResponse) GetFinishBuild() FinishBuildFinishBuildFinishBuildPayload {
	return v.FinishBuild
}

// LatestImageApp includes the requested fields of the GraphQL type App.
type LatestImageApp struct {
	// The latest release of this application, without any config processing
	CurrentReleaseUnprocessed LatestImageAppCurrentReleaseUnprocessed `json:"currentReleaseUnprocessed"`
}

// GetCurrentReleaseUnprocessed returns LatestImageApp.CurrentReleaseUnprocessed, and is useful for accessing the field via an interface.
func (v *LatestImageApp) GetCurrentReleaseUnprocessed() LatestImageAppCurrentReleaseUnprocessed {
	return v.CurrentReleaseUnprocessed
}

// LatestImageAppCurrentReleaseUnprocessed includes the requested fields of the GraphQL type ReleaseUnprocessed.
type LatestImageAppCurrentReleaseUnprocessed struct {
	// Unique ID
	Id string `json:"id"`
	// The version of the release
	Version int `json:"version"`
	// Docker image URI
	ImageRef string `json:"imageRef"`
}

// GetId returns LatestImageAppCurrentReleaseUnprocessed.Id, and is useful for accessing the field via an interface.
func (v *LatestImageAppCurrentReleaseUnprocessed) GetId() string { return v.Id }

// GetVersion returns LatestImageAppCurrentReleaseUnprocessed.Version, and is useful for accessing the field via an interface.
func (v *LatestImageAppCurrentReleaseUnprocessed) GetVersion() int { return v.Version }

// GetImageRef returns LatestImageAppCurrentReleaseUnprocessed.ImageRef, and is useful for accessing the field via an interface.
func (v *LatestImageAppCurrentReleaseUnprocessed) GetImageRef() string { return v.ImageRef }

// LatestImageResponse is returned by LatestImage on success.
type LatestImageResponse struct {
	// Find an app by name
	App LatestImageApp `json:"app"`
}

// GetApp returns LatestImageResponse.App, and is useful for accessing the field via an interface.
func (v *LatestImageResponse) GetApp() LatestImageApp { return v.App }

// Autogenerated input type of UpdateRelease
type UpdateReleaseInput struct {
	// A unique identifier for the client performing the mutation.
	ClientMutationId string `json:"clientMutationId"`
	// The metadata for the release
	Metadata interface{} `json:"metadata"`
	// The ID of the release
	ReleaseId string `json:"releaseId"`
	// The new status for the release
	Status string `json:"status"`
}

// GetClientMutationId returns UpdateReleaseInput.ClientMutationId, and is useful for accessing the field via an interface.
func (v *UpdateReleaseInput) GetClientMutationId() string { return v.ClientMutationId }

// GetMetadata returns UpdateReleaseInput.Metadata, and is useful for accessing the field via an interface.
func (v *UpdateReleaseInput) GetMetadata() interface{} { return v.Metadata }

// GetReleaseId returns UpdateReleaseInput.ReleaseId, and is useful for accessing the field via an interface.
func (v *UpdateReleaseInput) GetReleaseId() string { return v.ReleaseId }

// GetStatus returns UpdateReleaseInput.Status, and is useful for accessing the field via an interface.
func (v *UpdateReleaseInput) GetStatus() string { return v.Status }

// UpdateReleaseResponse is returned by UpdateRelease on success.
type UpdateReleaseResponse struct {
	UpdateRelease UpdateReleaseUpdateReleaseUpdateReleasePayload `json:"updateRelease"`
}

// GetUpdateRelease returns UpdateReleaseResponse.UpdateRelease, and is useful for accessing the field via an interface.
func (v *UpdateReleaseResponse) GetUpdateRelease() UpdateReleaseUpdateReleaseUpdateReleasePayload {
	return v.UpdateRelease
}

// UpdateReleaseUpdateReleaseUpdateReleasePayload includes the requested fields of the GraphQL type UpdateReleasePayload.
// The GraphQL type's documentation follows.
//
// Autogenerated return type of UpdateRelease.
type UpdateReleaseUpdateReleaseUpdateReleasePayload struct {
	Release UpdateReleaseUpdateReleaseUpdateReleasePayloadRelease `json:"release"`
}

// GetRelease returns UpdateReleaseUpdateReleaseUpdateReleasePayload.Release, and is useful for accessing the field via an interface.
func (v *UpdateReleaseUpdateReleaseUpdateReleasePayload) GetRelease() UpdateReleaseUpdateReleaseUpdateReleasePayloadRelease {
	return v.Release
}

// UpdateReleaseUpdateReleaseUpdateReleasePayloadRelease includes the requested fields of the GraphQL type Release.
type UpdateReleaseUpdateReleaseUpdateReleasePayloadRelease struct {
	// Unique ID
	Id string `json:"id"`
}

// GetId returns UpdateReleaseUpdateReleaseUpdateReleasePayloadRelease.Id, and is useful for accessing the field via an interface.
func (v *UpdateReleaseUpdateReleaseUpdateReleasePayloadRelease) GetId() string { return v.Id }

// __CreateBuildInput is used internally by genqlient
type __CreateBuildInput struct {
	Input CreateBuildInput `json:"input"`
}

// GetInput returns __CreateBuildInput.Input, and is useful for accessing the field via an interface.
func (v *__CreateBuildInput) GetInput() CreateBuildInput { return v.Input }

// __CreateReleaseInput is used internally by genqlient
type __CreateReleaseInput struct {
	Input CreateReleaseInput `json:"input"`
}

// GetInput returns __CreateReleaseInput.Input, and is useful for accessing the field via an interface.
func (v *__CreateReleaseInput) GetInput() CreateReleaseInput { return v.Input }

// __EnsureDepotRemoteBuilderInput is used internally by genqlient
type __EnsureDepotRemoteBuilderInput struct {
	Input *EnsureDepotRemoteBuilderInput `json:"input"`
}

// GetInput returns __EnsureDepotRemoteBuilderInput.Input, and is useful for accessing the field via an interface.
func (v *__EnsureDepotRemoteBuilderInput) GetInput() *EnsureDepotRemoteBuilderInput { return v.Input }

// __FinishBuildInput is used internally by genqlient
type __FinishBuildInput struct {
	Input FinishBuildInput `json:"input"`
}

// GetInput returns __FinishBuildInput.Input, and is useful for accessing the field via an interface.
func (v *__FinishBuildInput) GetInput() FinishBuildInput { return v.Input }

// __LatestImageInput is used internally by genqlient
type __LatestImageInput struct {
	AppName string `json:"appName"`
}

// GetAppName returns __LatestImageInput.AppName, and is useful for accessing the field via an interface.
func (v *__LatestImageInput) GetAppName() string { return v.AppName }

// __UpdateReleaseInput is used internally by genqlient
type __UpdateReleaseInput struct {
	Input UpdateReleaseInput `json:"input"`
}

// GetInput returns __UpdateReleaseInput.Input, and is useful for accessing the field via an interface.
func (v *__UpdateReleaseInput) GetInput() UpdateReleaseInput { return v.Input }

// The query or mutation executed by CreateBuild.
const CreateBuild_Operation = `
mutation CreateBuild ($input: CreateBuildInput!) {
	createBuild(input: $input) {
		id
		status
	}
}
`

func CreateBuild(
	ctx_ context.Context,
	client_ graphql.Client,
	input CreateBuildInput,
) (*CreateBuildResponse, error) {
	req_ := &graphql.Request{
		OpName: "CreateBuild",
		Query:  CreateBuild_Operation,
		Variables: &__CreateBuildInput{
			Input: input,
		},
	}
	var err_ error

	var data_ CreateBuildResponse
	resp_ := &graphql.Response{Data: &data_}

	err_ = client_.MakeRequest(
		ctx_,
		req_,
		resp_,
	)

	return &data_, err_
}

// The query or mutation executed by CreateRelease.
const CreateRelease_Operation = `
mutation CreateRelease ($input: CreateReleaseInput!) {
	createRelease(input: $input) {
		release {
			id
			version
		}
	}
}
`

func CreateRelease(
	ctx_ context.Context,
	client_ graphql.Client,
	input CreateReleaseInput,
) (*CreateReleaseResponse, error) {
	req_ := &graphql.Request{
		OpName: "CreateRelease",
		Query:  CreateRelease_Operation,
		Variables: &__CreateReleaseInput{
			Input: input,
		},
	}
	var err_ error

	var data_ CreateReleaseResponse
	resp_ := &graphql.Response{Data: &data_}

	err_ = client_.MakeRequest(
		ctx_,
		req_,
		resp_,
	)

	return &data_, err_
}

// The query or mutation executed by EnsureDepotRemoteBuilder.
const EnsureDepotRemoteBuilder_Operation = `
mutation EnsureDepotRemoteBuilder ($input: EnsureDepotRemoteBuilderInput!) {
	ensureDepotRemoteBuilder(input: $input) {
		buildId
		buildToken
	}
}
`

func EnsureDepotRemoteBuilder(
	ctx_ context.Context,
	client_ graphql.Client,
	input *EnsureDepotRemoteBuilderInput,
) (*EnsureDepotRemoteBuilderResponse, error) {
	req_ := &graphql.Request{
		OpName: "EnsureDepotRemoteBuilder",
		Query:  EnsureDepotRemoteBuilder_Operation,
		Variables: &__EnsureDepotRemoteBuilderInput{
			Input: input,
		},
	}
	var err_ error

	var data_ EnsureDepotRemoteBuilderResponse
	resp_ := &graphql.Response{Data: &data_}

	err_ = client_.MakeRequest(
		ctx_,
		req_,
		resp_,
	)

	return &data_, err_
}

// The query or mutation executed by FinishBuild.
const FinishBuild_Operation = `
mutation FinishBuild ($input: FinishBuildInput!) {
	finishBuild(input: $input) {
		id
		status
		wallclockTimeMs
	}
}
`

func FinishBuild(
	ctx_ context.Context,
	client_ graphql.Client,
	input FinishBuildInput,
) (*FinishBuildResponse, error) {
	req_ := &graphql.Request{
		OpName: "FinishBuild",
		Query:  FinishBuild_Operation,
		Variables: &__FinishBuildInput{
			Input: input,
		},
	}
	var err_ error

	var data_ FinishBuildResponse
	resp_ := &graphql.Response{Data: &data_}

	err_ = client_.MakeRequest(
		ctx_,
		req_,
		resp_,
	)

	return &data_, err_
}

// The query or mutation executed by LatestImage.
const LatestImage_Operation = `
query LatestImage ($appName: String!) {
	app(name: $appName) {
		currentReleaseUnprocessed {
			id
			version
			imageRef
		}
	}
}
`

func LatestImage(
	ctx_ context.Context,
	client_ graphql.Client,
	appName string,
) (*LatestImageResponse, error) {
	req_ := &graphql.Request{
		OpName: "LatestImage",
		Query:  LatestImage_Operation,
		Variables: &__LatestImageInput{
			AppName: appName,
		},
	}
	var err_ error

	var data_ LatestImageResponse
	resp_ := &graphql.Response{Data: &data_}

	err_ = client_.MakeRequest(
		ctx_,
		req_,
		resp_,
	)

	return &data_, err_
}

// The query or mutation executed by UpdateRelease.
const UpdateRelease_Operation = `
mutation UpdateRelease ($input: UpdateReleaseInput!) {
	updateRelease(input: $input) {
		release {
			id
		}
	}
}
`

func UpdateRelease(
	ctx_ context.Context,
	client_ graphql.Client,
	input UpdateReleaseInput,
) (*UpdateReleaseResponse, error) {
	req_ := &graphql.Request{
		OpName: "UpdateRelease",
		Query:  UpdateRelease_Operation,
		Variables: &__UpdateReleaseInput{
			Input: input,
		},
	}
	var err_ error

	var data_ UpdateReleaseResponse
	resp_ := &graphql.Response{Data: &data_}

	err_ = client_.MakeRequest(
		ctx_,
		req_,
		resp_,
	)

	return &data_, err_
}
