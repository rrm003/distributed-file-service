package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	pb "github.com/rrm003/grpc/file_management/file_management"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

const (
	address         = "localhost:50051"
	uploadsFolder   = "uploads"
	downloadsFolder = "downloads"
)

var (
	rootCmd = &cobra.Command{Use: "fileclient"}
)

type client struct {
	pb.FileServiceClient
	mu sync.Mutex
}

func main() {
	// Initialize the gRPC connection and client.
	conn, err := grpc.Dial(address, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Printf("Could not connect: %v\n", err)
		return
	}

	defer conn.Close()

	c := pb.NewFileServiceClient(conn)
	client := &client{FileServiceClient: c}

	go client.SynchronizeFolder()

	// Add commands to the CLI
	rootCmd.AddCommand(uploadCmd(client))
	rootCmd.AddCommand(downloadCmd(client))
	rootCmd.AddCommand(deleteCmd(client))
	rootCmd.AddCommand(renameCmd(client))

	// Execute the CLI commands
	if err := rootCmd.Execute(); err != nil {
		log.Printf("Error executing command: %v", err)
		return
	}

	select {}
}

// Helper function to create a file stream for uploading.
func createUploadStream(client pb.FileServiceClient) (pb.FileService_UploadFileClient, error) {
	stream, err := client.UploadFile(context.Background())
	if err != nil {
		return nil, err
	}

	return stream, nil
}

// Helper function to send the file name to the server.
func sendFileName(stream pb.FileService_UploadFileClient, fileName string) error {
	return stream.Send(&pb.FileChunk{Data: []byte(fileName)})
}

// Helper function to send file data to the server.
func sendFileData(stream pb.FileService_UploadFileClient, data []byte) error {
	fmt.Println("client sending file chuncks")
	return stream.Send(&pb.FileChunk{Data: data})
}

// Helper function to close the file stream.
func closeFileStream(stream pb.FileService_UploadFileClient) error {
	_, err := stream.CloseAndRecv()
	return err
}

// Helper function to download a file from the server.
func downloadFile(client pb.FileServiceClient, fileName, filePath string) {
	stream, err := client.DownloadFile(context.Background(), &pb.FileRequest{FileName: fileName})
	if err != nil {
		log.Printf("Could not download file: %v", err)
		return
	}
	defer stream.CloseSend()

	file, err := os.Create(filePath + "/" + fileName)
	if err != nil {
		log.Printf("Could not create file: %v", err)
		return
	}

	defer file.Close()

	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("Error receiving file data: %v", err)
			return
		}

		data := resp.Data
		_, err = file.Write(data)
		if err != nil {
			log.Printf("Error writing file: %v", err)
		}
	}

	fmt.Printf("File %s downloaded to %s\n", fileName, filePath)
}

func uploadFile(client *client, filePath, fileName string) error {
	if _, err := os.Stat(filePath); err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("File path '%s' does not exist.\n", filePath)
		} else {
			fmt.Printf("Error checking file path: %v\n", err)
		}

		return err
	}

	downloadsFilePath := downloadsFolder + "/" + fileName
	if _, err := os.Stat(downloadsFilePath); os.IsNotExist(err) {
		// Copy the file to the "downloads" folder
		if err := copyFile(filePath, downloadsFilePath); err != nil {
			fmt.Printf("Error copying file to 'downloads' folder: %v\n", err)
			return err
		}
		fmt.Printf("File copied to 'downloads' folder: %s\n", downloadsFilePath)
	}

	stream, err := createUploadStream(client.FileServiceClient)
	if err != nil {
		log.Printf("Could not create upload stream: %v", err)
	}
	defer closeFileStream(stream)

	file, err := os.Open(filePath)
	if err != nil {
		log.Printf("Could not open file: %v", err)
	}
	defer file.Close()

	if err := sendFileName(stream, fileName); err != nil {
		log.Printf("Error sending file name: %v", err)
	}

	buffer := make([]byte, 1024)
	for {
		n, err := file.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("Error reading file: %v", err)
		}

		if err := sendFileData(stream, buffer[:n]); err != nil {
			log.Printf("Error sending file data: %v", err)
		}
	}

	return nil
}

// Helper function to delete a file on the server.
func deleteFile(client pb.FileServiceClient, fileName string) {
	fName := downloadsFolder + "/" + fileName

	//delete local file
	err := os.Remove(fName)
	if os.IsNotExist(err) {
		// handle the case where the file doesn't exist
		log.Println("finel not exist in local")
	} else {
		log.Printf("Could not delete local file: %v", err)
	}

	// delete server file
	resp, err := client.DeleteFile(context.Background(), &pb.FileRequest{FileName: fileName})
	if err != nil {
		log.Printf("Could not delete server file: %v", err)
	}

	fmt.Println(resp.Message)
}

// Helper function to rename a file on the server.
func renameFile(client pb.FileServiceClient, oldFileName, newFileName string) {
	if _, err := os.Stat(downloadsFolder + "/" + oldFileName); err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("File path '%s' does not exist.\n", downloadsFolder+"/"+oldFileName)
		} else {
			fmt.Printf("Error checking file path: %v\n", err)
		}

		return
	}

	// rename local file
	err := os.Rename(downloadsFolder+"/"+oldFileName, downloadsFolder+"/"+newFileName)
	if err != nil {
		log.Printf("Could not rename local file: %v", err)
	}

	// rename server file
	// go func() {
	resp, err := client.RenameFile(context.Background(), &pb.RenameRequest{OldFileName: oldFileName, NewFileName: newFileName})
	if err != nil {
		log.Printf("Could not rename server file: %v", err)
	}

	fmt.Println(resp.Message)
}

// CLI command for file upload.
func uploadCmd(client *client) *cobra.Command {
	var filePath string
	var fileName string

	cmd := &cobra.Command{
		Use:   "upload",
		Short: "Upload a file to the server",
		Run: func(cmd *cobra.Command, args []string) {
			if err := uploadFile(client, filePath, fileName); err != nil {
				log.Printf("Error uploading file: %v", err)
			}
		},
	}

	cmd.Flags().StringVarP(&filePath, "file", "f", "", "Path to the file to upload")
	cmd.Flags().StringVarP(&fileName, "name", "n", "", "Name to use for the uploaded file on the server")

	cmd.MarkFlagRequired("file")
	cmd.MarkFlagRequired("name")

	return cmd
}

// CLI command for file download.
func downloadCmd(client *client) *cobra.Command {
	var fileName string
	var filePath string

	cmd := &cobra.Command{
		Use:   "download",
		Short: "Download a file from the server",
		Run: func(cmd *cobra.Command, args []string) {
			downloadFile(client.FileServiceClient, fileName, filePath)
		},
	}

	cmd.Flags().StringVarP(&fileName, "file", "f", "", "Name of the file to download")
	cmd.Flags().StringVarP(&filePath, "path", "p", "", "Path to save the downloaded file")

	cmd.MarkFlagRequired("file")
	cmd.MarkFlagRequired("path")

	return cmd
}

// CLI command for file deletion.
func deleteCmd(client *client) *cobra.Command {
	var fileName string

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a file on the server",
		Run: func(cmd *cobra.Command, args []string) {
			deleteFile(client.FileServiceClient, fileName)
		},
	}

	cmd.Flags().StringVarP(&fileName, "file", "f", "", "Name of the file to delete")
	cmd.MarkFlagRequired("file")

	return cmd
}

// CLI command for file renaming.
func renameCmd(client *client) *cobra.Command {
	var oldFileName string
	var newFileName string

	cmd := &cobra.Command{
		Use:   "rename",
		Short: "Rename a file on the server",
		Run: func(cmd *cobra.Command, args []string) {
			renameFile(client.FileServiceClient, oldFileName, newFileName)
		},
	}

	cmd.Flags().StringVarP(&oldFileName, "old", "o", "", "Current name of the file")
	cmd.Flags().StringVarP(&newFileName, "new", "n", "", "New name for the file")

	cmd.MarkFlagRequired("old")
	cmd.MarkFlagRequired("new")

	return cmd
}

// SynchronizeFolder periodically checks for changes in the synchronized folder and synchronizes with the server.
func (c *client) SynchronizeFolder() {
	// Create watcher for downloads directory
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer watcher.Close()

	// Channel to signal when done
	done := make(chan bool)
	var wg sync.WaitGroup

	// Start watching the downloads directory
	wg.Add(1)
	go watchDirectory(downloadsFolder, watcher, &wg)

	// Handle events
	go func() {
		for {
			select {
			case event := <-watcher.Events:
				c.handleEvent(event)
			case err := <-watcher.Errors:
				fmt.Println("Error:", err)
			}
		}
	}()

	// Wait for all goroutines to finish
	wg.Wait()
	done <- true
}

func watchDirectory(dir string, watcher *fsnotify.Watcher, wg *sync.WaitGroup) {
	defer wg.Done()

	err := watcher.Add(dir)
	if err != nil {
		fmt.Println("Error watching directory:", err)
		return
	}
	fmt.Println("Watching directory:", dir)

	<-make(chan struct{})
}

func (c *client) handleEvent(event fsnotify.Event) {
	switch {
	case event.Op&fsnotify.Create == fsnotify.Create:
		fmt.Println("File created:", event.Name)
		lists := strings.Split(event.Name, "/")
		fileName := lists[len(lists)-1]
		uploadFile(c, event.Name, fileName)

	case event.Op&fsnotify.Write == fsnotify.Write:
		fmt.Println("File updated:", event.Name)
		lists := strings.Split(event.Name, "/")
		fileName := lists[len(lists)-1]
		uploadFile(c, event.Name, fileName)

	case event.Op&fsnotify.Rename == fsnotify.Rename:
		// Check if the file exists to distinguish between rename and delete
		_, err := os.Stat(event.Name)
		if err == nil {
			fmt.Printf("File renamed: %+v\n", event)
			renameFile(c, "", event.Name)
		} else if os.IsNotExist(err) {
			lists := strings.Split(event.Name, "/")
			fileName := lists[len(lists)-1]
			fmt.Println("File deleted:", event.Name)
			deleteFile(c, fileName)
		}
	}
}

func copyFile(sourceFile, destinationFile string) error {
	src, err := os.Open(sourceFile)
	if err != nil {
		fmt.Println("Error opening source file:", err)
		return err
	}
	defer src.Close()

	// Create the destination file for writing
	dst, err := os.Create(destinationFile)
	if err != nil {
		fmt.Println("Error creating destination file:", err)
		return err
	}
	defer dst.Close()

	// Copy the contents of the source file to the destination file
	_, err = io.Copy(dst, src)
	if err != nil {
		fmt.Println("Error copying file:", err)
		return err
	}

	return nil
}
