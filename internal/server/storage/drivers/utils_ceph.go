package drivers

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/lxc/incus/v6/shared/api"
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

// CephMonitors calls `ceph-conf` for `mon_host` and parses the output for
// monitor IP:port pairs.
func CephMonitors(cluster string) ([]string, error) {
	out, err := subprocess.RunCommand(
		"ceph-conf",
		"--cluster",
		cluster,
		"mon host",
	)
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

func getCephKeyFromFile(path string) (string, error) {
	cephKeyring, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("Failed to open %q: %w", path, err)
	}

	// Locate the keyring entry and its value.
	var cephSecret string
	scan := bufio.NewScanner(cephKeyring)
	for scan.Scan() {
		line := scan.Text()
		line = strings.TrimSpace(line)

		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "key") {
			fields := strings.SplitN(line, "=", 2)
			if len(fields) < 2 {
				continue
			}

			cephSecret = strings.TrimSpace(fields[1])
			break
		}
	}

	if cephSecret == "" {
		return "", fmt.Errorf("Couldn't find a keyring entry")
	}

	return cephSecret, nil
}

// CephKeyring gets the key for a particular Ceph cluster and client name.
func CephKeyring(cluster string, client string) (string, error) {
	// Prefix client. to client if it does not have a prefix
	if !strings.Contains(client, ".") {
		client = "client." + client
	}

	// Sometimes the key is just, like, right there ya know
	key, err := subprocess.RunCommand(
		"ceph-conf",
		"--cluster", cluster,
		"--name", client,
		"key",
	)
	if err == nil && key != "" {
		return key, nil
	}

	// Sometimes it's in a keyfile
	keyfile, err := subprocess.RunCommand(
		"ceph-conf",
		"--cluster", cluster,
		"--name", client,
		"keyfile",
	)
	if err == nil && keyfile != "" {
		buf, err := os.ReadFile(keyfile)
		if err != nil {
			return "", fmt.Errorf("Failed to read ceph keyfile %q: %w", keyfile, err)
		}

		return string(buf), nil
	}

	// It's probably in a keyring
	keyring, err := subprocess.RunCommand(
		"ceph-conf",
		"--cluster", cluster,
		"--name", client,
		"keyring",
	)
	if err == nil && keyring != "" {
		// Use ceph-conf again to read the keyfile
		key, err := subprocess.RunCommand(
			"ceph-conf",
			"--conf", keyring,
			"--cluster", cluster,
			"--name", client,
			"key",
		)
		if err == nil && key != "" {
			return key, nil
		}
	}

	// Give up
	return "", fmt.Errorf("Couldn't find a Ceph key for %q on %q", client, cluster)
}
