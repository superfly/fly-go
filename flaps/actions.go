package flaps

//go:generate go run golang.org/x/tools/cmd/stringer@latest -type=flapsAction

// flapsAction is used to record actions in traces' attributes.
type flapsAction int

const (
	none flapsAction = iota
	appCreate
	machineLaunch
	machineUpdate
	machineStart
	machineWait
	machineStop
	machineRestart
	machineGet
	machineList
	machineDestroy
	machineKill
	machineFindLease
	machineAcquireLease
	machineRefreshLease
	machineReleaseLease
	machineExec
	machinePs
	machineCordon
	machineUncordon
	machineSuspend
	secretCreate
	secretDelete
	secretGenerate
	secretsList
	volumeList
	volumeCreate
	volumetUpdate
	volumeGet
	volumeSnapshotCreate
	volumeSnapshotList
	volumeExtend
	volumeDelete
	metadataSet
	metadataGet
	metadataDel
	getS3LogsToken
)
