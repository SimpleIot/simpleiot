package files

import (
	"bytes"
	"fmt"
	"github.com/simpleiot/simpleiot/assets/frontend"
	"io/ioutil"
	"os"
	"path"
)

// FileUpdate describes a file that gets updated
type FileUpdate struct {
	Dest     string
	Perm     os.FileMode
	Callback func()
}

// UpdateFiles updates various files in the system
func UpdateFiles(dataDir string) error {
	fileUpdates := []FileUpdate{
		// currently not using this, saving for future use
	}

	for _, fu := range fileUpdates {
		f := path.Base(fu.Dest)
		fBytes := frontend.Asset(path.Join("/", f))
		if fBytes == nil {
			return fmt.Errorf("Error opening update for: %v", f)
		}

		fOldBytes, _ := ioutil.ReadFile(fu.Dest)
		if bytes.Compare(fBytes, fOldBytes) != 0 {
			fmt.Println("Updating: ", fu.Dest)
			err := ioutil.WriteFile(fu.Dest, fBytes, fu.Perm)
			if err != nil {
				return fmt.Errorf("Error updating: %v", fu.Dest)
			}
			if fu.Callback != nil {
				fu.Callback()
			}
		}
	}

	return nil
}
