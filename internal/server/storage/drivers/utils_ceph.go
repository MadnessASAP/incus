package drivers

import (
	"fmt"
	"os"
	"strings"

	"github.com/lxc/incus/v6/shared/api"
	"github.com/lxc/incus/v6/shared/logger"
	"github.com/lxc/incus/v6/shared/subprocess"
)

// CephGetRBDImageName returns the RBD image name as it is used in ceph.
// Example:
// A custom block volume named vol1 in project default will return custom_default_vol1.block.
func CephGetRBDImageName(vol Volume, snapName string, zombie bool) string {
	var out string
	parentName, snapshotName, isSnapshot := api.GetParentAndSnapshotName(vol.name)

	// Only use filesystem suffix on filesystem type image volumes (for all content types).
	if vol.volType == VolumeTypeImage || vol.volType == cephVolumeTypeZombieImage {
		parentName = fmt.Sprintf("%s_%s", parentName, vol.ConfigBlockFilesystem())
	}

	if vol.contentType == ContentTypeBlock {
		parentName = fmt.Sprintf("%s%s", parentName, cephBlockVolSuffix)
	} else if vol.contentType == ContentTypeISO {
		parentName = fmt.Sprintf("%s%s", parentName, cephISOVolSuffix)
	}

	// Use volume's type as storage volume prefix, unless there is an override in cephVolTypePrefixes.
	volumeTypePrefix := string(vol.volType)
	volumeTypePrefixOverride, foundOveride := cephVolTypePrefixes[vol.volType]
	if foundOveride {
		volumeTypePrefix = volumeTypePrefixOverride
	}

	if snapName != "" {
		// Always use the provided snapshot name if specified.
		out = fmt.Sprintf("%s_%s@%s", volumeTypePrefix, parentName, snapName)
	} else {
		if isSnapshot {
			// If volumeName is a snapshot (<vol>/<snap>) and snapName is not set,
			// assume that it's a normal snapshot (not a zombie) and prefix it with
			// "snapshot_".
			out = fmt.Sprintf("%s_%s@snapshot_%s", volumeTypePrefix, parentName, snapshotName)
		} else {
			out = fmt.Sprintf("%s_%s", volumeTypePrefix, parentName)
		}
	}

	// If the volume is to be in zombie state (i.e. not tracked in the database),
	// prefix the output with "zombie_".
	if zombie {
		out = fmt.Sprintf("zombie_%s", out)
	}

	return out
}

// CephBuildMount creates a mount string and option list from mount parameters.
func CephBuildMount(user string, key string, fsid string, monitors []string, fsName string, path string) (source string, options []string) {
	// if path is blank, assume root of fs
	if path == "" {
		path = "/"
	}

	// build the source path
	source = fmt.Sprintf("%s@%s.%s=%s", user, fsid, fsName, path)

	// build the options list
	options = []string{
		"mon_addr=" + strings.Join(monitors, "/"),
	}

	// if key is blank assume cephx is disabled
	if key != "" {
		options = append(options, "secret="+key)
	}

	return source, options
}

// callCephConf makes a call to `ceph-conf` to retrieve a given lookup value.
// An empty string for `cluster`, `id`, or `conf` results in default values
// being used.
func callCephConf(cluster string, id string, conf string, lookup string) (value string, err error) {
	const cmd = "ceph-conf"
	var args []string

	if cluster != "" {
		args = append(args, "--cluster", cluster)
	}

	if id != "" {
		// Prefix client. to client if it does not have a prefix
		if !strings.Contains(id, ".") {
			id = "client." + id
		}

		args = append(args, "--name", id)
	}

	if conf != "" {
		args = append(args, "--conf", conf)
	}

	args = append(args, lookup)

	value, err = subprocess.RunCommand(cmd, args...)
	ctx := logger.Ctx{
		"cmd":    cmd,
		"args":   args,
		"err":    err,
		"output": value,
	}

	logger.Debug("callCephConf", ctx)
	return value, err
}

// CephMonitors calls `ceph-conf` for `mon_host` and parses the output for
// monitor IP:port pairs.
func CephMonitors(cluster string) ([]string, error) {
	out, err := callCephConf(cluster, "", "", "mon_host")
	if err != nil {
		return nil, fmt.Errorf(
			"Failed to get monitors for %q from ceph-conf: %w",
			cluster,
			err,
		)
	}

	// Parse the output to extract the monitor IP:port pairs
	cephMon := []string{}
	// Split monitor address groups
	monitors := strings.Split(out, " ")
	for _, mon := range monitors {
		// Monitor address groups are square bracketed and comma delimited
		mon = strings.Trim(mon, "[ ]")
		addrs := strings.Split(mon, ",")
		for _, addr := range addrs {
			// Monitor addresses begin with a version tag and end with a nonce, those
			// aren't needed i.e: `v2:1.2.3.4:3300/0`
			_, addr, _ = strings.Cut(addr, ":")
			addr, _, _ = strings.Cut(addr, "/")

			// Append the address
			cephMon = append(cephMon, addr)
		}
	}

	if len(cephMon) == 0 {
		return nil, fmt.Errorf("Couldn't find a CEPH mon")
	}

	return cephMon, nil
}

// CephKeyring gets the key for a particular Ceph cluster and client name. Not
// finding a key may not be fatal, it may indicate that the cluster does not
// require authentication.
func CephKeyring(cluster string, client string) (string, error) {
	// Sometimes the key is just, like, right there ya know
	key, err := callCephConf(cluster, client, "", "key")
	if err == nil && key != "" {
		return key, nil
	}

	// Sometimes it's in a keyfile
	keyfile, err := callCephConf(cluster, client, "", "keyfile")
	if err == nil && keyfile != "" {
		buf, err := os.ReadFile(keyfile)
		if err != nil {
			return "", fmt.Errorf("Failed to read ceph keyfile %q: %w", keyfile, err)
		}

		return string(buf), nil
	}

	// It's probably in a keyring
	keyring, err := callCephConf(cluster, client, "", "keyring")
	if err == nil && keyring != "" {
		// Use ceph-conf again to read the keyfile
		key, err := callCephConf(cluster, client, keyfile, "key")
		if err == nil && key != "" {
			return key, nil
		}
	}

	logger.Warnf("Could not find a key for %q, maybe cephx is disabled?", cluster)
	// Give up
	return "", nil
}

// CephFsid gets the FSID for a given cluster name.
func CephFsid(cluster string) (string, error) {
	fsid, err := callCephConf(cluster, "", "", "fsid")
	if err != nil {
		return "", fmt.Errorf("Couldn't get fsid for %q: %w", cluster, err)
	}

	return strings.TrimSpace(fsid), nil
}
