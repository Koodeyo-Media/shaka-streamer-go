// Pushes output from packager to cloud.
package streamer

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	CACHE_CONTROL_HEADER = "Cache-Control: no-store, no-transform"
)

var COMMON_GSUTIL_ARGS = []string{
	"gsutil",
	"-q",
	"-h", CACHE_CONTROL_HEADER,
	"-m",
	"rsync",
	"-C",
	"-r",
}

type CloudAccessError struct {
	BucketURL string
}

func (e CloudAccessError) Error() string {
	return fmt.Sprintf("Unable to write to cloud storage URL: %s\n\n"+
		"Please double-check that the URL is correct, that you are signed into the Google Cloud SDK or Amazon AWS CLI, "+
		"and that you have access to the destination bucket.", e.BucketURL)
}

type CloudNode struct {
	InputDir      string
	BucketURL     string
	TempDir       string
	PackagerNodes []PackagerNode
	IsVOD         bool
	Status        ProcessStatus
	Process1      *exec.Cmd
	Process2      *exec.Cmd
}

func NewCloudNode(inputDir string, bucketURL string, tempDir string, packagerNodes []PackagerNode, isVOD bool) *CloudNode {
	cn := &CloudNode{
		InputDir:      inputDir,
		BucketURL:     bucketURL,
		TempDir:       tempDir,
		PackagerNodes: packagerNodes,
		IsVOD:         isVOD,
		Status:        Finished,
	}

	return cn
}

/*
Called early to test that the user can write to the destination bucket.

	Writes an empty file called ".shaka-streamer-access-check" to the
	destination.  Raises CloudAccessError if the destination cannot be written
	to.
*/
func (cn CloudNode) CheckAccess(bucketURL string) {
	// Make sure there are not two slashes in a row here, which would create a
	// subdirectory whose name is "".
	destination := strings.TrimRight(bucketURL, "/") + "/.shaka-streamer-access-check"
	// Note that this can't be "gsutil ls" on the destination, because the user
	// might have read-only access.  In fact, some buckets grant read-only
	// access to anonymous (non-logged-in) users.  So writing to the bucket is
	// the only way to check.
	cmd := exec.Command("gsutil", "cp", "-", destination)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout

	if _, err := cmd.StderrPipe(); err != nil {
		panic(err)
	}

	if err := cmd.Start(); err != nil {
		panic(err)
	}

	exitStatus := func(err error) (int, bool) {
		if exiterr, ok := err.(*exec.ExitError); ok {
			status := exiterr.ExitCode()
			return status, true
		}

		return 0, false
	}

	if err := cmd.Wait(); err != nil {
		if status, ok := exitStatus(err); ok && status != 0 {
			panic(CloudAccessError{BucketURL: bucketURL})
		}
	}
}

func (cn *CloudNode) Upload() {
	// With recursive=True, glob's ** will also match the base dir.
	manifestFiles, err := filepath.Glob(filepath.Join(cn.InputDir, "**/*.mpd"))

	if err != nil {
		panic(err)
	}

	m3u8Files, err := filepath.Glob(filepath.Join(cn.InputDir, "**/*.m3u8"))

	if err != nil {
		panic(err)
	}

	manifestFiles = append(manifestFiles, m3u8Files...)

	// The manifest at any moment will reference existing segment files.
	// We must be careful not to upload a manifest that references segments that
	// haven't been uploaded yet.  So first we will capture manifest contents,
	// then upload current segments, then upload the manifest contents we
	// captured.
	for _, manifestPath := range manifestFiles {
		// The path within the input dir.
		subdirPath, err := filepath.Rel(cn.InputDir, manifestPath)
		if err != nil {
			panic(err)
		}

		// Capture manifest contents, and retry until the file is non-empty or
		// until the thread is killed.
		contents, err := os.ReadFile(manifestPath)

		for len(contents) == 0 && cn.CheckStatus() == Running {
			time.Sleep(100 * time.Millisecond)
			contents, err = os.ReadFile(manifestPath)
		}

		if err != nil {
			panic(err)
		}

		// Now that we have manifest contents, put them into a temp file so that
		// the manifests can be pushed en masse later.
		tempFilePath := filepath.Join(cn.TempDir, subdirPath)
		// Create any necessary intermediate folders.
		tempFileDirPath := filepath.Dir(tempFilePath)
		err = os.MkdirAll(tempFileDirPath, 0755)

		if err != nil {
			panic(err)
		}
		// Write the temp file.
		err = os.WriteFile(tempFilePath, contents, 0644)
		if err != nil {
			panic(err)
		}
	}

	// Sync all files except manifest files.
	args := append(COMMON_GSUTIL_ARGS, []string{
		"-d",           // delete remote files that are no longer needed
		"-x", ".*m3u8", // skip m3u8 files, which we'll push separately later
		"-x", ".*mpd", // skip mpd files, which we'll push separately later
		"-r", cn.InputDir, // local input folder to sync
		cn.BucketURL, // destination in cloud storage
	}...)

	// NOTE: The -d option above will not result in the files ignored by -x
	// being deleted from the remote storage location.
	cmd := exec.Command(args[0], args[1:]...)
	cn.Process1 = cmd
	err = cmd.Run()

	if err != nil {
		panic(err)
	}

	compressionArgs := []string{}

	if strings.HasPrefix(cn.BucketURL, "gs:") {
		// This arg seems to fail on S3, but still works for GCS.
		compressionArgs = []string{"-J"}
	}

	// Sync the temporary copies of the manifest files.
	args = append(COMMON_GSUTIL_ARGS, compressionArgs...)
	args = append(args, []string{
		"-r", cn.TempDir, // local input folder to sync
		cn.BucketURL, // destination in cloud storage
	}...)

	cmd2 := exec.Command(args[0], args[1:]...)
	cn.Process2 = cmd2
	err = cmd2.Run()

	if err != nil {
		panic(err)
	}
}

func (cn CloudNode) CheckStatus() ProcessStatus {
	return cn.Status
}

func (cn *CloudNode) Stop() {
	cn.Status = Finished
	cn.Process1.Process.Kill()
	cn.Process2.Process.Kill()
}

func (cn *CloudNode) Start() {
	cn.Status = Running
}
