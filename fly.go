package fly

import "slices"

// MergeFiles merges the files parsed from the command line or fly.toml into the machine configuration.
func MergeFiles(machineConf *MachineConfig, files []*File) {
	for _, f := range files {
		idx := slices.IndexFunc(machineConf.Files, func(i *File) bool {
			return i.GuestPath == f.GuestPath
		})

		switch {
		case idx == -1:
			machineConf.Files = append(machineConf.Files, f)
			continue
		case f.RawValue == nil && f.SecretName == nil:
			machineConf.Files = slices.Delete(machineConf.Files, idx, idx+1)
		default:
			machineConf.Files = slices.Replace(machineConf.Files, idx, idx+1, f)
		}
	}
}
