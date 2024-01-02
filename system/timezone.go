package system

import (
	"os"
	"path"
	"regexp"
)

// ReadTimezones returns a list of possible time zones
// from the system
// Possible arguments for zoneInfoDir:
//
//	"" (root dir)
//	"US"
//	"posix/America"
func ReadTimezones(zoneInfoDir string) (listZones []string, err error) {

	fileInfo, err := os.ReadDir(path.Join(zoneInfoPath, zoneInfoDir))
	if err != nil {
		return nil, err
	}

	for _, fi := range fileInfo {
		if !fi.IsDir() { // if file, not directory
			listZones = append(listZones, fi.Name())
		}
	}

	return listZones, nil
}

// GetTimezone returns the current system timezone and
// its path after zoneInfoPath
func GetTimezone() (zoneInfoDir, zone string, err error) {

	link, err := os.Readlink(zoneLink)
	if err != nil {
		return "", "", err
	}

	// extract the timezone and the zoneInfoDir from the full path
	pattern := regexp.MustCompile(`/usr/share/zoneinfo/(.+)`)
	matches := pattern.FindStringSubmatch(link)

	if len(matches) < 2 {
		return "", "", nil
	}

	pattern2 := regexp.MustCompile(`(.+)/(.+)`)
	matches2 := pattern2.FindStringSubmatch(matches[1])

	if len(matches2) < 3 {
		return "", matches[1], nil
	}

	return matches2[1], matches2[2], nil
}

// SetTimezone sets the current system time zone
func SetTimezone(zoneInfoDir, zone string) (err error) {

	if _, err := os.Lstat(zoneLink); err == nil {
		err := os.Remove(zoneLink)
		if err != nil {
			return err
		}
	}

	err = os.Symlink(path.Join(zoneInfoPath, zoneInfoDir, zone), zoneLink)
	if err != nil {
		return err
	}

	return nil
}

// Path to zoneinfo
const zoneInfoPath = "/usr/share/zoneinfo/"

// Symbolic link for the system timezone
const zoneLink = "/etc/localtime"
