package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"sync"

	pb "github.com/rrm003/grpc/file_management/file_management"

	"google.golang.org/grpc"
)

const (
	port             = ":50051"
	uploadsFolder    = "uploads"
	downloadsFolder  = "downloads"
	bufferSize       = 1024
	maxConcurrentOps = 10
)

// server implements the FileServiceServer interface.
type server struct {
	pb.UnimplementedFileServiceServer
	mu sync.Mutex
}

// UploadFile handles the file upload operation.
func (s *server) UploadFile(stream pb.FileService_UploadFileServer) error {
	fileName, err := receiveFileName(stream)
	if err != nil {
		return err
	}

	fmt.Println("upload: received file : ", fileName)
	filePath := filepath.Join(uploadsFolder, fileName)
	fmt.Println(filePath)

	err = receiveFile(stream, filePath)
	if err != nil {
		log.Println("error receiving files", err)
		return err
	}

	return nil
}

// DownloadFile handles the file download operation.
func (s *server) DownloadFile(req *pb.FileRequest, stream pb.FileService_DownloadFileServer) error {
	filePath := filepath.Join(uploadsFolder, req.FileName)
	fmt.Println("file path :", filePath)

	if _, err := os.Stat(filePath); err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("File path '%s' does not exist.\n", filePath)
		} else {
			fmt.Printf("Error checking file path: %v\n", err)
		}

		return err
	}

	err := sendFile(filePath, stream)
	if err != nil {
		fmt.Println("error seding file", err)
		return err
	}

	return nil
}

// DeleteFile handles the file delete operation.
func (s *server) DeleteFile(ctx context.Context, req *pb.FileRequest) (*pb.Response, error) {
	fmt.Println("server deleting file", req.FileName)
	filePath := filepath.Join(uploadsFolder, req.FileName)

	err := deleteFile(filePath)
	if err != nil {
		fmt.Println("error deleting file", err)
		return nil, err
	}

	return &pb.Response{Message: fmt.Sprintf("File %s deleted successfully", req.FileName)}, nil
}

// RenameFile handles the file rename operation.
func (s *server) RenameFile(ctx context.Context, req *pb.RenameRequest) (*pb.Response, error) {
	oldFilePath := filepath.Join(uploadsFolder, req.OldFileName)
	newFilePath := filepath.Join(uploadsFolder, req.NewFileName)

	err := renameFile(oldFilePath, newFilePath)
	if err != nil {
		fmt.Println("error renaming file", err)
		return nil, err
	}

	return &pb.Response{Message: fmt.Sprintf("File %s renamed to %s", req.OldFileName, req.NewFileName)}, nil
}

// Helper function to list files on the server.
func (s *server) ListFiles(ctx context.Context, _ *pb.Empty) (*pb.FileList, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	files, err := os.ReadDir(uploadsFolder)
	if err != nil {
		fmt.Printf("erorr reading dir %s :%v\n", uploadsFolder, err)
		return nil, err
	}

	var fileList pb.FileList
	for _, file := range files {
		info, _ := file.Info()
		fileInfo := &pb.FileInfo{
			FileName:         info.Name(),
			ModificationTime: info.ModTime().Unix(),
		}
		fileList.Files = append(fileList.Files, fileInfo)
	}

	return &fileList, nil
}

// Helper function to receive the file name from the client.
func receiveFileName(stream pb.FileService_UploadFileServer) (string, error) {
	req, err := stream.Recv()
	if err != nil {
		fmt.Println("error receving stream", err)
		return "", err
	}

	// Assuming FileName is within a message called FileNameMessage
	return string(req.GetData()), nil
}

// Helper function to receive the file data from the client and save it to the server.
func receiveFile(stream pb.FileService_UploadFileServer, filePath string) error {
	fmt.Println("upload request: writing to the file")

	file, err := os.Create(filePath)
	if err != nil {
		fmt.Println("error creating file", filePath)
		return err
	}

	defer file.Close()

	i := 0
	for {
		fmt.Println("chunks: ", i)
		i++

		req, err := stream.Recv()
		fmt.Println(err)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		data := req.Data
		_, err = file.Write(data)
		if err != nil {
			return err
		}
	}

	return nil
}

// Helper function to send the file data to the client.
func sendFile(filePath string, stream pb.FileService_DownloadFileServer) error {
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Println("error opeing file", filePath, err)
		return err
	}
	defer file.Close()

	fmt.Println("send file ", file.Name())

	buffer := make([]byte, 1024)
	for {
		n, err := file.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Println("error reafing file from buffer", err)
			return err
		}

		err = stream.Send(&pb.FileChunk{Data: buffer[:n]})
		if err != nil {
			fmt.Println("error sending file chunks in stream", err)
			return err
		}
	}

	return nil
}

// Helper function to delete a file from the server.
func deleteFile(filePath string) error {
	return os.Remove(filePath)
}

// Helper function to rename a file on the server.
func renameFile(oldFilePath, newFilePath string) error {
	return os.Rename(oldFilePath, newFilePath)
}

func main() {
	if err := os.MkdirAll(uploadsFolder, 0755); err != nil {
		log.Printf("Failed to create uploads folder: %v\n", err)
		return
	}
	if err := os.MkdirAll(downloadsFolder, 0755); err != nil {
		log.Printf("Failed to create downloads folder: %v\n", err)
		return
	}

	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Printf("Failed to listen: %v\n", err)
		return
	}
	defer lis.Close()

	s := grpc.NewServer()
	pb.RegisterFileServiceServer(s, &server{})

	fmt.Printf("Server listening on %s\n", port)
	if err := s.Serve(lis); err != nil {
		log.Printf("Failed to serve: %v", err)
		return
	}

	return
}
